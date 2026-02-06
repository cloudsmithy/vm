package service

import (
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"kvmmm/internal/model"

	libvirt "github.com/digitalocean/go-libvirt"
)

var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type LibvirtService struct {
	l  *libvirt.Libvirt
	mu sync.Mutex // protects all libvirt operations
}

func NewLibvirtService() (*LibvirtService, error) {
	svc := &LibvirtService{}
	if err := svc.connect(); err != nil {
		return nil, err
	}
	return svc, nil
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
		vms = append(vms, model.VM{
			Name:   d.Name,
			UUID:   uuidStr,
			State:  stateName(libvirt.DomainState(state)),
			CPU:    cpu,
			Memory: mem,
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
	return s.l.DomainCreate(d)
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

	// Clean up disk files
	for _, p := range diskPaths {
		os.Remove(p)
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
	xmlStr, err := s.l.DomainGetXMLDesc(d, 0)
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
	// Ensure unit attribute is KiB
	re := regexp.MustCompile(`<memory[^>]*>`)
	xmlStr = re.ReplaceAllString(xmlStr, fmt.Sprintf(`<memory unit='KiB'>`))
	xmlStr = replaceXMLTag(xmlStr, "currentMemory", fmt.Sprintf("%d", kiB))
	re2 := regexp.MustCompile(`<currentMemory[^>]*>`)
	xmlStr = re2.ReplaceAllString(xmlStr, fmt.Sprintf(`<currentMemory unit='KiB'>`))
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

	// Create qcow2 disk image outside the lock
	diskPath := fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", req.Name)
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

	// Optional second CDROM for VirtIO drivers ISO
	virtioCD := ""
	if req.VirtioISO != "" {
		virtioCD = fmt.Sprintf(`
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='%s' bus='%s'/>
      <readonly/>
    </disk>`, req.VirtioISO, virtioCdromDev, cdromBus)
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
      <driver name='qemu' type='raw'/>
      <target dev='%s' bus='%s'/>
      <readonly/>
    </disk>%s
    <interface type='network'>
      <source network='default'/>
      <model type='%s'/>
    </interface>
    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'/>
    <console type='pty'/>
  </devices>
</domain>`, req.Name, req.Memory, req.CPU, cpuXML, clockXML, machineAttr, scsiCtrl, req.Name, diskDev, diskBus, cdromDev, cdromBus, virtioCD, netModel)

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

	rModel, rMemory, rCpus, _, _, _, _, _, err := s.l.NodeGetInfo()
	if err != nil {
		return nil, err
	}

	// Convert [32]int8 to string
	modelBytes := make([]byte, 0, 32)
	for _, b := range rModel {
		if b == 0 {
			break
		}
		modelBytes = append(modelBytes, byte(b))
	}

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

	memFree, err := s.l.NodeGetFreeMemory()
	if err != nil {
		memFree = 0
	}

	return &model.HostInfo{
		Hostname:    hostname,
		CPUModel:    string(modelBytes),
		CPUCount:    int(rCpus),
		MemoryTotal: int(rMemory / 1024), // rMemory is KiB, convert to MiB
		MemoryFree:  int(memFree / 1024 / 1024),  // bytes -> MiB
		VMRunning:   running,
		VMTotal:     len(vms),
	}, nil
}
