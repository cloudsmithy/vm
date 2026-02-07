package service

import (
	"encoding/xml"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"virtpanel/internal/model"

	libvirt "github.com/digitalocean/go-libvirt"
)

var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type cpuSample struct {
	time    uint64
	ts      time.Time
}

type LibvirtService struct {
	l          *libvirt.Libvirt
	mu         sync.Mutex
	cpuCache   map[string]cpuSample // domain name -> last cpu sample
	hostCPU    float64              // cached host CPU usage
	hostCPUMu  sync.RWMutex
	stopCh     chan struct{}
}

func NewLibvirtService() (*LibvirtService, error) {
	svc := &LibvirtService{cpuCache: make(map[string]cpuSample), stopCh: make(chan struct{})}
	if err := svc.connect(); err != nil {
		return nil, err
	}
	svc.hostCPU = readCPUUsage() // initial sample
	go svc.cpuSampleLoop()
	return svc, nil
}

// cpuSampleLoop samples host CPU usage in background every 2s
func (s *LibvirtService) cpuSampleLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			v := readCPUUsage()
			s.hostCPUMu.Lock()
			s.hostCPU = v
			s.hostCPUMu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

func (s *LibvirtService) connect() error {
	c, err := net.DialTimeout("unix", "/var/run/libvirt/libvirt-sock", 2*time.Second)
	if err != nil {
		return fmt.Errorf("dial libvirt: %w", err)
	}
	l := libvirt.New(c)
	if err := l.Connect(); err != nil {
		return fmt.Errorf("connect libvirt: %w", err)
	}
	s.l = l
	return nil
}

// ensureConnected checks and reconnects if needed. Caller must hold s.mu.
func (s *LibvirtService) ensureConnected() error {
	if _, err := s.l.ConnectGetLibVersion(); err == nil {
		return nil
	}
	_ = s.l.Disconnect()
	return s.connect()
}

func (s *LibvirtService) Close() {
	close(s.stopCh)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.l.Disconnect()
}

var stateMap = map[libvirt.DomainState]string{
	libvirt.DomainRunning:  "running",
	libvirt.DomainShutoff:  "shutoff",
	libvirt.DomainPaused:   "paused",
	libvirt.DomainShutdown: "shutdown",
	libvirt.DomainCrashed:  "crashed",
	libvirt.DomainBlocked:  "blocked",
}

func stateName(s libvirt.DomainState) string {
	if n, ok := stateMap[s]; ok {
		return n
	}
	return "unknown"
}

// domainXML is a minimal struct to extract CPU/memory from domain XML
type domainXML struct {
	XMLName xml.Name `xml:"domain"`
	VCPU    int      `xml:"vcpu"`
	Memory  struct {
		Value int    `xml:",chardata"`
		Unit  string `xml:"unit,attr"`
	} `xml:"memory"`
}

func parseDomainInfo(xmlStr string) (cpu int, memMB int) {
	var d domainXML
	if xml.Unmarshal([]byte(xmlStr), &d) == nil {
		cpu = d.VCPU
		switch d.Memory.Unit {
		case "GiB":
			memMB = d.Memory.Value * 1024
		case "MiB":
			memMB = d.Memory.Value
		case "KiB", "":
			memMB = d.Memory.Value / 1024
		case "bytes", "b":
			memMB = d.Memory.Value / 1024 / 1024
		default:
			memMB = d.Memory.Value / 1024
		}
	}
	return
}

func (s *LibvirtService) ListVMs() ([]model.VM, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	return s.listVMsLocked()
}

// listVMsLocked requires s.mu to be held.
func (s *LibvirtService) listVMsLocked() ([]model.VM, error) {
	domains, _, err := s.l.ConnectListAllDomains(-1, 0)
	if err != nil {
		return nil, err
	}
	vms := make([]model.VM, 0, len(domains))
	now := time.Now()
	for _, d := range domains {
		state, _, _, _, _, err := s.l.DomainGetInfo(d)
		if err != nil {
			continue
		}
		xmlStr, err := s.l.DomainGetXMLDesc(d, 0)
		if err != nil {
			continue
		}
		cpu, mem := parseDomainInfo(xmlStr)
		uuidStr := fmt.Sprintf("%x", d.UUID)
		st := stateName(libvirt.DomainState(state))

		var cpuUsage float64
		var memUsed int

		if st == "running" {
			// CPU usage: compare with cached sample
			_, _, cpuTimeNs, _, _, err := s.l.DomainGetInfo(d)
			if err == nil {
				prev, ok := s.cpuCache[d.Name]
				if ok {
					dt := now.Sub(prev.ts).Seconds()
					if dt > 0 && cpu > 0 {
						cpuUsage = float64(cpuTimeNs-prev.time) / (dt * 1e9 * float64(cpu)) * 100
						if cpuUsage > 100 {
							cpuUsage = 100
						}
						if cpuUsage < 0 {
							cpuUsage = 0
						}
						cpuUsage = math.Round(cpuUsage*10) / 10
					}
				}
				s.cpuCache[d.Name] = cpuSample{time: cpuTimeNs, ts: now}
			}

			// Memory: try balloon stats via dommemstat
			memStats, err := s.l.DomainMemoryStats(d, 11, 0)
			if err == nil {
				var available, unused uint64
				for _, ms := range memStats {
					switch ms.Tag {
					case 6: // available
						available = ms.Val
					case 4: // unused
						unused = ms.Val
					}
				}
				if available > 0 && unused > 0 {
					memUsed = int((available - unused) / 1024) // KiB -> MiB
				}
			}
		}

		vms = append(vms, model.VM{
			Name:     d.Name,
			UUID:     uuidStr,
			State:    st,
			CPU:      cpu,
			Memory:   mem,
			CPUUsage: cpuUsage,
			MemUsed:  memUsed,
		})
	}
	return vms, nil
}

func (s *LibvirtService) GetVM(name string) (*model.VM, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return nil, err
	}
	state, _, _, _, _, err := s.l.DomainGetInfo(d)
	if err != nil {
		return nil, err
	}
	xmlStr, err := s.l.DomainGetXMLDesc(d, 0)
	if err != nil {
		return nil, err
	}
	cpu, mem := parseDomainInfo(xmlStr)
	return &model.VM{
		Name:   d.Name,
		UUID:   fmt.Sprintf("%x", d.UUID),
		State:  stateName(libvirt.DomainState(state)),
		CPU:    cpu,
		Memory: mem,
	}, nil
}

func (s *LibvirtService) StartVM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	if err := s.l.DomainCreate(d); err != nil {
		return err
	}
	// After first boot from cdrom, switch boot order to hd-first
	// so next reboot (after OS install) boots from disk
	xmlStr, err := s.l.DomainGetXMLDesc(d, libvirt.DomainXMLInactive)
	if err != nil {
		return nil // VM started, non-fatal
	}
	bootRe := regexp.MustCompile(`<boot dev=['"]cdrom['"]/>\s*<boot dev=['"]hd['"]/>`)
	if bootRe.MatchString(xmlStr) {
		newXML := bootRe.ReplaceAllString(xmlStr, "<boot dev='hd'/><boot dev='cdrom'/>")
		s.l.DomainDefineXML(newXML)
	}
	return nil
}

func (s *LibvirtService) ShutdownVM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.DomainShutdown(d)
}

func (s *LibvirtService) DestroyVM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.DomainDestroy(d)
}

func (s *LibvirtService) RebootVM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.DomainReboot(d, 0)
}

func (s *LibvirtService) DeleteVM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}

	// Collect disk paths before undefining
	xmlStr, xmlErr := s.l.DomainGetXMLDesc(d, 0)
	var diskPaths []string
	if xmlErr == nil && xmlStr != "" {
		var dx detailDomainXML
		if xml.Unmarshal([]byte(xmlStr), &dx) == nil {
			for _, disk := range dx.Devices.Disks {
				if disk.Device == "disk" && disk.Source.File != "" {
					diskPaths = append(diskPaths, disk.Source.File)
				}
			}
		}
	}

	// 先尝试强制关闭
	_ = s.l.DomainDestroy(d)
	// Undefine with snapshots metadata cleanup
	err = s.l.DomainUndefineFlags(d, libvirt.DomainUndefineSnapshotsMetadata)
	if err != nil {
		// Fallback to simple undefine if flags not supported
		err = s.l.DomainUndefine(d)
		if err != nil {
			return err
		}
	}

	// Clean up disk files (only if not used by other VMs)
	if len(diskPaths) > 0 {
		usedPaths := make(map[string]bool)
		domains, _, listErr := s.l.ConnectListAllDomains(-1, 0)
		if listErr == nil {
			for _, od := range domains {
				ox, err := s.l.DomainGetXMLDesc(od, libvirt.DomainXMLInactive)
				if err != nil {
					continue
				}
				var odx detailDomainXML
				if xml.Unmarshal([]byte(ox), &odx) == nil {
					for _, disk := range odx.Devices.Disks {
						if disk.Source.File != "" {
							usedPaths[disk.Source.File] = true
						}
					}
				}
			}
		}
		for _, p := range diskPaths {
			if !usedPaths[p] {
				os.Remove(p)
			}
		}
	}
	return nil
}

func (s *LibvirtService) SuspendVM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.DomainSuspend(d)
}

func (s *LibvirtService) ResumeVM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.DomainResume(d)
}

func (s *LibvirtService) GetAutostart(name string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return false, err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return false, err
	}
	v, err := s.l.DomainGetAutostart(d)
	return v != 0, err
}

func (s *LibvirtService) SetAutostart(name string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	var v int32
	if enabled {
		v = 1
	}
	return s.l.DomainSetAutostart(d, v)
}

func (s *LibvirtService) RenameVM(oldName, newName string) error {
	if !safeNameRe.MatchString(newName) {
		return fmt.Errorf("invalid vm name: %s", newName)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(oldName)
	if err != nil {
		return err
	}
	// VM must be shut off
	state, _, _, _, _, err := s.l.DomainGetInfo(d)
	if err != nil {
		return err
	}
	if libvirt.DomainState(state) != libvirt.DomainShutoff {
		return fmt.Errorf("vm must be shut off to rename")
	}
	_, err = s.l.DomainRename(d, libvirt.OptString{newName}, 0)
	return err
}

func (s *LibvirtService) ImportVM(req model.ImportVMRequest) error {
	if !safeNameRe.MatchString(req.Name) {
		return fmt.Errorf("invalid vm name: %s", req.Name)
	}
	if req.CPU <= 0 {
		req.CPU = 2
	}
	if req.Memory <= 0 {
		req.Memory = 2048
	}

	diskBus := req.DiskBus
	if diskBus == "" {
		diskBus = "virtio"
	}
	validBus := map[string]bool{"virtio": true, "sata": true, "scsi": true, "ide": true}
	if !validBus[diskBus] {
		return fmt.Errorf("unsupported disk bus: %s", diskBus)
	}
	diskDev := map[string]string{"virtio": "vda", "scsi": "sda", "sata": "sda", "ide": "hdc"}[diskBus]

	// Validate disk path exists
	cleanPath := filepath.Clean(req.DiskPath)
	if _, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("磁盘文件不存在: %s", cleanPath)
	}

	format := "qcow2"
	if strings.HasSuffix(cleanPath, ".raw") || strings.HasSuffix(cleanPath, ".img") {
		format = "raw"
	}

	xmlDef := fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <memory unit='MiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os><type arch='x86_64'>hvm</type><boot dev='hd'/></os>
  <features><acpi/><apic/></features>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='%s'/>
      <source file='%s'/>
      <target dev='%s' bus='%s'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <target dev='hda' bus='ide'/>
      <readonly/>
    </disk>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'/>
    <video>
      <model type='qxl' ram='65536' vram='65536' vgamem='32768' heads='1' primary='yes'/>
    </video>
    <input type='tablet' bus='usb'/>
    <console type='pty'/>
  </devices>
</domain>`, req.Name, req.Memory, req.CPU, format, cleanPath, diskDev, diskBus)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	_, err := s.l.DomainDefineXML(xmlDef)
	return err
}

func (s *LibvirtService) UpdateVM(name string, req model.UpdateVMRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	xmlStr, err := s.l.DomainGetXMLDesc(d, libvirt.DomainXMLInactive)
	if err != nil {
		return err
	}

	newXML := xmlStr
	if req.CPU > 0 {
		newXML = replaceXMLTag(newXML, "vcpu", fmt.Sprintf("%d", req.CPU))
	}
	if req.Memory > 0 {
		memKiB := req.Memory * 1024
		newXML = replaceXMLMemory(newXML, memKiB)
	}
	if newXML == xmlStr {
		return nil
	}

	_, err = s.l.DomainDefineXML(newXML)
	return err
}

func replaceXMLTag(xmlStr, tag, value string) string {
	re := regexp.MustCompile(fmt.Sprintf(`<%s[^>]*>[^<]*</%s>`, tag, tag))
	return re.ReplaceAllStringFunc(xmlStr, func(match string) string {
		// Preserve attributes from opening tag
		openEnd := strings.Index(match, ">")
		closeStart := strings.LastIndex(match, "</")
		return match[:openEnd+1] + value + match[closeStart:]
	})
}

func replaceXMLMemory(xmlStr string, kiB int) string {
	xmlStr = replaceXMLTag(xmlStr, "memory", fmt.Sprintf("%d", kiB))
	re := regexp.MustCompile(`<memory[^>]*>`)
	xmlStr = re.ReplaceAllString(xmlStr, `<memory unit='KiB'>`)
	// Handle currentMemory: replace if exists, insert if not
	if strings.Contains(xmlStr, "<currentMemory") {
		xmlStr = replaceXMLTag(xmlStr, "currentMemory", fmt.Sprintf("%d", kiB))
		re2 := regexp.MustCompile(`<currentMemory[^>]*>`)
		xmlStr = re2.ReplaceAllString(xmlStr, `<currentMemory unit='KiB'>`)
	} else {
		xmlStr = strings.Replace(xmlStr, fmt.Sprintf("<memory unit='KiB'>%d</memory>", kiB),
			fmt.Sprintf("<memory unit='KiB'>%d</memory>\n  <currentMemory unit='KiB'>%d</currentMemory>", kiB, kiB), 1)
	}
	return xmlStr
}

func (s *LibvirtService) CreateVM(req model.CreateVMRequest) error {
	if !safeNameRe.MatchString(req.Name) {
		return fmt.Errorf("invalid vm name: %s", req.Name)
	}
	if req.CPU <= 0 {
		req.CPU = 2
	}
	if req.Memory <= 0 {
		req.Memory = 2048
	}
	if req.Disk <= 0 {
		req.Disk = 20
	}

	// Defaults from OS type preset
	diskBus, netModel := "virtio", "virtio"
	machine, cpuModel, clock := "i440fx", "", "utc"
	switch req.OSType {
	case "windows":
		diskBus, netModel = "sata", "e1000"
		machine, cpuModel, clock = "q35", "host-passthrough", "localtime"
	case "legacy":
		diskBus, netModel = "ide", "rtl8139"
	}
	// Explicit overrides
	if req.DiskBus != "" {
		diskBus = req.DiskBus
	}
	if req.NetModel != "" {
		netModel = req.NetModel
	}
	if req.Machine != "" {
		machine = req.Machine
	}
	if req.CPUModel != "" {
		cpuModel = req.CPUModel
	}
	if req.Clock != "" {
		clock = req.Clock
	}

	// Validate
	validBus := map[string]bool{"virtio": true, "sata": true, "scsi": true, "ide": true}
	validNet := map[string]bool{"virtio": true, "e1000": true, "rtl8139": true}
	if !validBus[diskBus] {
		return fmt.Errorf("unsupported disk bus: %s", diskBus)
	}
	if !validNet[netModel] {
		return fmt.Errorf("unsupported net model: %s", netModel)
	}

	// Determine disk target device name by bus type
	diskDev := map[string]string{"virtio": "vda", "scsi": "sda", "sata": "sda", "ide": "hdc"}[diskBus]

	// SCSI controller XML (only needed for scsi bus)
	scsiCtrl := ""
	if diskBus == "scsi" {
		scsiCtrl = "\n    <controller type='scsi' model='virtio-scsi'/>"
	}

	// Machine type
	machineAttr := ""
	if machine == "q35" {
		machineAttr = " machine='pc-q35-7.2'"
	}

	// CPU model
	cpuXML := ""
	if cpuModel == "host-passthrough" {
		cpuXML = "\n  <cpu mode='host-passthrough'/>"
	} else if cpuModel == "host-model" {
		cpuXML = "\n  <cpu mode='host-model'/>"
	}

	// Clock
	clockXML := fmt.Sprintf("\n  <clock offset='%s'/>", clock)
	if clock == "localtime" {
		clockXML = "\n  <clock offset='localtime'>\n    <timer name='rtc' tickpolicy='catchup'/>\n    <timer name='pit' tickpolicy='delay'/>\n    <timer name='hpet' present='no'/>\n    <timer name='hypervclock' present='yes'/>\n  </clock>"
	}

	// Check if disk already exists
	diskPath := fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", req.Name)
	if _, err := os.Stat(diskPath); err == nil {
		return fmt.Errorf("磁盘文件已存在: %s，请使用其他名称", diskPath)
	}

	// Create qcow2 disk image outside the lock
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", diskPath, fmt.Sprintf("%dG", req.Disk))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create disk failed: %s", string(output))
	}

	// CDROM bus: q35 has no IDE, use sata
	cdromBus, cdromDev := "ide", "hda"
	virtioCdromDev := "hdb"
	if machine == "q35" {
		cdromBus, cdromDev = "sata", "sdb"
		virtioCdromDev = "sdc"
	}

	// Primary CDROM with optional install ISO
	cdromSource := ""
	if req.ISO != "" {
		cleanISO := filepath.Clean(req.ISO)
		if !strings.HasPrefix(cleanISO, isoDir+"/") {
			return fmt.Errorf("iso path must be under %s", isoDir)
		}
		cdromSource = fmt.Sprintf("\n      <source file='%s'/>", cleanISO)
	}

	// Optional second CDROM for VirtIO drivers ISO
	virtioCD := ""
	if req.VirtioISO != "" {
		cleanVirtio := filepath.Clean(req.VirtioISO)
		if !strings.HasPrefix(cleanVirtio, isoDir+"/") {
			return fmt.Errorf("virtio iso path must be under %s", isoDir)
		}
		virtioCD = fmt.Sprintf(`
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='%s' bus='%s'/>
      <readonly/>
    </disk>`, cleanVirtio, virtioCdromDev, cdromBus)
	}

	// Network interface XML based on mode
	netXML := fmt.Sprintf(`<interface type='network'>
      <source network='default'/>
      <model type='%s'/>
    </interface>`, netModel)
	switch req.NetMode {
	case "bridge":
		bridgeName := req.BridgeName
		if bridgeName == "" {
			bridgeName = "br0"
		}
		netXML = fmt.Sprintf(`<interface type='bridge'>
      <source bridge='%s'/>
      <model type='%s'/>
    </interface>`, bridgeName, netModel)
	case "macvtap":
		dev := req.MacvtapDev
		if dev == "" {
			return fmt.Errorf("macvtap 模式需要指定物理网卡")
		}
		netXML = fmt.Sprintf(`<interface type='direct'>
      <source dev='%s' mode='bridge'/>
      <model type='%s'/>
    </interface>`, dev, netModel)
	}

	xmlDef := fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <memory unit='MiB'>%d</memory>
  <vcpu>%d</vcpu>%s%s
  <os><type arch='x86_64'%s>hvm</type><boot dev='cdrom'/><boot dev='hd'/></os>
  <features><acpi/><apic/></features>
  <devices>%s
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='/var/lib/libvirt/images/%s.qcow2'/>
      <target dev='%s' bus='%s'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>%s
      <target dev='%s' bus='%s'/>
      <readonly/>
    </disk>%s
    %s
    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'/>
    <video>
      <model type='qxl' ram='65536' vram='65536' vgamem='32768' heads='1' primary='yes'/>
    </video>
    <input type='tablet' bus='usb'/>
    <console type='pty'/>
  </devices>
</domain>`, req.Name, req.Memory, req.CPU, cpuXML, clockXML, machineAttr, scsiCtrl, req.Name, diskDev, diskBus, cdromSource, cdromDev, cdromBus, virtioCD, netXML)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		os.Remove(diskPath)
		return err
	}
	_, err := s.l.DomainDefineXML(xmlDef)
	if err != nil {
		os.Remove(diskPath)
	}
	return err
}

func (s *LibvirtService) GetHostInfo() (*model.HostInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	hostname, err := s.l.ConnectGetHostname()
	if err != nil {
		return nil, err
	}

	_, rMemory, rCpus, _, _, _, _, _, err := s.l.NodeGetInfo()
	if err != nil {
		return nil, err
	}

	// Read real CPU model from /proc/cpuinfo
	cpuModel := "unknown"
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "model name") {
				if idx := strings.Index(line, ":"); idx >= 0 {
					cpuModel = strings.TrimSpace(line[idx+1:])
				}
				break
			}
		}
	}

	// Read MemAvailable from /proc/meminfo
	memAvailMiB := 0
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemAvailable:") {
				var kb int
				fmt.Sscanf(line, "MemAvailable: %d kB", &kb)
				memAvailMiB = kb / 1024
				break
			}
		}
	}

	// Read cached CPU usage (sampled in background)
	s.hostCPUMu.RLock()
	cpuUsage := s.hostCPU
	s.hostCPUMu.RUnlock()

	vms, err := s.listVMsLocked()
	if err != nil {
		return nil, err
	}

	running := 0
	for _, v := range vms {
		if v.State == "running" {
			running++
		}
	}

	// Uptime
	var uptime int64
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		var up float64
		fmt.Sscanf(string(data), "%f", &up)
		uptime = int64(up)
	}

	// Load average
	var loadAvg [3]float64
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fmt.Sscanf(string(data), "%f %f %f", &loadAvg[0], &loadAvg[1], &loadAvg[2])
	}

	// Disk usage
	disks := readDiskUsage()

	return &model.HostInfo{
		Hostname:    hostname,
		CPUModel:    cpuModel,
		CPUCount:    int(rCpus),
		CPUUsage:    cpuUsage,
		MemoryTotal: int(rMemory / 1024),
		MemoryFree:  memAvailMiB,
		VMRunning:   running,
		VMTotal:     len(vms),
		Uptime:      uptime,
		LoadAvg:     loadAvg,
		Disks:       disks,
	}, nil
}

func readCPUUsage() float64 {
	read := func() (idle, total uint64) {
		data, err := os.ReadFile("/proc/stat")
		if err != nil {
			return
		}
		line := strings.Split(string(data), "\n")[0] // "cpu  ..."
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return
		}
		for i := 1; i < len(fields); i++ {
			var v uint64
			fmt.Sscanf(fields[i], "%d", &v)
			total += v
			if i == 4 { // idle is 4th field
				idle = v
			}
		}
		return
	}
	idle1, total1 := read()
	time.Sleep(200 * time.Millisecond)
	idle2, total2 := read()
	dt := total2 - total1
	if dt == 0 {
		return 0
	}
	return math.Round((1-float64(idle2-idle1)/float64(dt))*1000) / 10
}

func readDiskUsage() []model.DiskInfo {
	out, err := exec.Command("df", "-BG", "--output=target,source,size,used,avail,pcent", "-x", "tmpfs", "-x", "devtmpfs", "-x", "overlay", "-x", "squashfs").Output()
	if err != nil {
		return nil
	}
	var disks []model.DiskInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mount := fields[0]
		// Skip non-physical mounts
		if strings.HasPrefix(mount, "/snap") || strings.HasPrefix(mount, "/boot/efi") {
			continue
		}
		parseG := func(s string) uint64 {
			s = strings.TrimSuffix(s, "G")
			var v uint64
			fmt.Sscanf(s, "%d", &v)
			return v
		}
		pct := strings.TrimSuffix(fields[5], "%")
		var p int
		fmt.Sscanf(pct, "%d", &p)
		disks = append(disks, model.DiskInfo{
			Mount:     mount,
			Device:    fields[1],
			Total:     parseG(fields[2]),
			Used:      parseG(fields[3]),
			Available: parseG(fields[4]),
			Percent:   p,
		})
	}
	return disks
}
