package service

import (
	"encoding/xml"
	"fmt"
	"kvmmm/internal/model"
	"os/exec"
	"path/filepath"
	"strings"

	libvirt "github.com/digitalocean/go-libvirt"
)

// Full domain XML structs for detail parsing
type detailDomainXML struct {
	XMLName xml.Name `xml:"domain"`
	Name    string   `xml:"name"`
	VCPU    int      `xml:"vcpu"`
	Memory  struct {
		Value int    `xml:",chardata"`
		Unit  string `xml:"unit,attr"`
	} `xml:"memory"`
	OS struct {
		Type struct {
			Arch string `xml:"arch,attr"`
		} `xml:"type"`
		Boot []struct {
			Dev string `xml:"dev,attr"`
		} `xml:"boot"`
	} `xml:"os"`
	Devices struct {
		Disks      []detailDiskXML      `xml:"disk"`
		Interfaces []detailInterfaceXML `xml:"interface"`
	} `xml:"devices"`
}

type detailDiskXML struct {
	Device string `xml:"device,attr"`
	Driver struct {
		Type string `xml:"type,attr"`
	} `xml:"driver"`
	Source struct {
		File string `xml:"file,attr"`
	} `xml:"source"`
	Target struct {
		Dev string `xml:"dev,attr"`
		Bus string `xml:"bus,attr"`
	} `xml:"target"`
}

type detailInterfaceXML struct {
	Type   string `xml:"type,attr"`
	Source struct {
		Network string `xml:"network,attr"`
		Bridge  string `xml:"bridge,attr"`
	} `xml:"source"`
	MAC struct {
		Address string `xml:"address,attr"`
	} `xml:"mac"`
	Model struct {
		Type string `xml:"type,attr"`
	} `xml:"model"`
}

func bootOrder(boots []struct{ Dev string `xml:"dev,attr"` }) string {
	devs := make([]string, 0, len(boots))
	for _, b := range boots {
		devs = append(devs, b.Dev)
	}
	return strings.Join(devs, ", ")
}

func (s *LibvirtService) GetVMDetail(name string) (*model.VMDetail, error) {
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

	var dx detailDomainXML
	if err := xml.Unmarshal([]byte(xmlStr), &dx); err != nil {
		return nil, err
	}

	cpu, memMB := parseDomainInfo(xmlStr)

	detail := &model.VMDetail{
		Name:   name,
		UUID:   fmt.Sprintf("%x", d.UUID),
		State:  stateName(libvirt.DomainState(state)),
		CPU:    cpu,
		Memory: memMB,
		Boot:   bootOrder(dx.OS.Boot),
		Arch:   dx.OS.Type.Arch,
		Disks:  []model.VMDisk{},
		NICs:   []model.VMNIC{},
	}

	for _, disk := range dx.Devices.Disks {
		detail.Disks = append(detail.Disks, model.VMDisk{
			Device: disk.Device,
			Source: disk.Source.File,
			Target: disk.Target.Dev,
			Bus:    disk.Target.Bus,
			Format: disk.Driver.Type,
		})
	}

	for _, iface := range dx.Devices.Interfaces {
		src := iface.Source.Network
		if src == "" {
			src = iface.Source.Bridge
		}
		detail.NICs = append(detail.NICs, model.VMNIC{
			Type:   iface.Type,
			Source: src,
			MAC:    iface.MAC.Address,
			Model:  iface.Model.Type,
		})
	}

	return detail, nil
}

func (s *LibvirtService) AttachDisk(vmName string, req model.AttachDiskRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	// Validate disk path is under libvirt images
	cleanPath := filepath.Clean(req.Source)
	if !strings.HasPrefix(cleanPath, "/var/lib/libvirt/") {
		return fmt.Errorf("disk source must be under /var/lib/libvirt/")
	}
	if strings.ContainsAny(cleanPath, `<>&'"`) {
		return fmt.Errorf("disk source contains invalid characters")
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}
	if req.Target == "" {
		req.Target = "vdb"
	}
	if req.Bus == "" {
		req.Bus = "virtio"
	}
	if !safeNameRe.MatchString(req.Target) {
		return fmt.Errorf("invalid target device: %s", req.Target)
	}
	validBus := map[string]bool{"virtio": true, "ide": true, "scsi": true, "sata": true}
	if !validBus[req.Bus] {
		return fmt.Errorf("invalid bus type: %s", req.Bus)
	}

	// Detect format from extension
	format := "qcow2"
	if strings.HasSuffix(req.Source, ".raw") || strings.HasSuffix(req.Source, ".img") {
		format = "raw"
	}

	xmlDef := fmt.Sprintf(`<disk type='file' device='disk'>
  <driver name='qemu' type='%s'/>
  <source file='%s'/>
  <target dev='%s' bus='%s'/>
</disk>`, format, cleanPath, req.Target, req.Bus)

	// Try live attach first, fall back to config-only
	state, _, _, _, _, _ := s.l.DomainGetInfo(d)
	var flags libvirt.DomainDeviceModifyFlags
	if libvirt.DomainState(state) == libvirt.DomainRunning {
		flags = libvirt.DomainDeviceModifyLive | libvirt.DomainDeviceModifyConfig
	} else {
		flags = libvirt.DomainDeviceModifyConfig
	}
	return s.l.DomainAttachDeviceFlags(d, xmlDef, uint32(flags))
}

func (s *LibvirtService) DetachDisk(vmName, target string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}

	// Get current XML to find the disk
	xmlStr, err := s.l.DomainGetXMLDesc(d, 0)
	if err != nil {
		return err
	}
	var dx detailDomainXML
	if err := xml.Unmarshal([]byte(xmlStr), &dx); err != nil {
		return fmt.Errorf("parse domain xml: %w", err)
	}

	for _, disk := range dx.Devices.Disks {
		if disk.Target.Dev == target {
			format := disk.Driver.Type
			if format == "" {
				format = "qcow2"
			}
			xmlDef := fmt.Sprintf(`<disk type='file' device='disk'>
  <driver name='qemu' type='%s'/>
  <source file='%s'/>
  <target dev='%s' bus='%s'/>
</disk>`, format, disk.Source.File, disk.Target.Dev, disk.Target.Bus)

			state, _, _, _, _, _ := s.l.DomainGetInfo(d)
			var flags libvirt.DomainDeviceModifyFlags
			if libvirt.DomainState(state) == libvirt.DomainRunning {
				flags = libvirt.DomainDeviceModifyLive | libvirt.DomainDeviceModifyConfig
			} else {
				flags = libvirt.DomainDeviceModifyConfig
			}
			return s.l.DomainDetachDeviceFlags(d, xmlDef, uint32(flags))
		}
	}
	return fmt.Errorf("disk %s not found", target)
}

func (s *LibvirtService) AttachNIC(vmName string, req model.AttachNICRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if !safeNameRe.MatchString(req.Network) {
		return fmt.Errorf("invalid network name: %s", req.Network)
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}
	if req.Model == "" {
		req.Model = "virtio"
	}
	validModel := map[string]bool{"virtio": true, "e1000": true, "rtl8139": true}
	if !validModel[req.Model] {
		return fmt.Errorf("invalid nic model: %s", req.Model)
	}
	xmlDef := fmt.Sprintf(`<interface type='network'>
  <source network='%s'/>
  <model type='%s'/>
</interface>`, req.Network, req.Model)

	state, _, _, _, _, _ := s.l.DomainGetInfo(d)
	var flags libvirt.DomainDeviceModifyFlags
	if libvirt.DomainState(state) == libvirt.DomainRunning {
		flags = libvirt.DomainDeviceModifyLive | libvirt.DomainDeviceModifyConfig
	} else {
		flags = libvirt.DomainDeviceModifyConfig
	}
	return s.l.DomainAttachDeviceFlags(d, xmlDef, uint32(flags))
}

func (s *LibvirtService) DetachNIC(vmName, mac string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}
	xmlStr, err := s.l.DomainGetXMLDesc(d, 0)
	if err != nil {
		return err
	}
	var dx detailDomainXML
	if err := xml.Unmarshal([]byte(xmlStr), &dx); err != nil {
		return fmt.Errorf("parse domain xml: %w", err)
	}

	for _, iface := range dx.Devices.Interfaces {
		if iface.MAC.Address == mac {
			src := iface.Source.Network
			srcType := "network"
			if src == "" {
				src = iface.Source.Bridge
				srcType = "bridge"
			}
			var srcAttr string
			if srcType == "network" {
				srcAttr = fmt.Sprintf(`network='%s'`, src)
			} else {
				srcAttr = fmt.Sprintf(`bridge='%s'`, src)
			}
			xmlDef := fmt.Sprintf(`<interface type='%s'>
  <source %s/>
  <mac address='%s'/>
  <model type='%s'/>
</interface>`, iface.Type, srcAttr, mac, iface.Model.Type)

			state, _, _, _, _, _ := s.l.DomainGetInfo(d)
			var flags libvirt.DomainDeviceModifyFlags
			if libvirt.DomainState(state) == libvirt.DomainRunning {
				flags = libvirt.DomainDeviceModifyLive | libvirt.DomainDeviceModifyConfig
			} else {
				flags = libvirt.DomainDeviceModifyConfig
			}
			return s.l.DomainDetachDeviceFlags(d, xmlDef, uint32(flags))
		}
	}
	return fmt.Errorf("nic with mac %s not found", mac)
}

func (s *LibvirtService) AttachISO(vmName string, isoPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	cleanPath := filepath.Clean(isoPath)
	if !strings.HasPrefix(cleanPath, isoDir+"/") {
		return fmt.Errorf("iso path must be under %s", isoDir)
	}
	if strings.ContainsAny(cleanPath, `<>&'"`) {
		return fmt.Errorf("iso path contains invalid characters")
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}

	// Detect existing cdrom bus from VM XML
	cdromDev, cdromBus := s.findCdromBus(d)

	xmlDef := fmt.Sprintf(`<disk type='file' device='cdrom'>
  <driver name='qemu' type='raw'/>
  <source file='%s'/>
  <target dev='%s' bus='%s'/>
  <readonly/>
</disk>`, cleanPath, cdromDev, cdromBus)

	state, _, _, _, _, _ := s.l.DomainGetInfo(d)
	var flags libvirt.DomainDeviceModifyFlags
	if libvirt.DomainState(state) == libvirt.DomainRunning {
		flags = libvirt.DomainDeviceModifyLive | libvirt.DomainDeviceModifyConfig
	} else {
		flags = libvirt.DomainDeviceModifyConfig
	}

	err = s.l.DomainUpdateDeviceFlags(d, xmlDef, flags)
	if err != nil {
		err = s.l.DomainAttachDeviceFlags(d, xmlDef, uint32(flags))
		if err != nil {
			return err
		}
	}

	fullXML, err := s.l.DomainGetXMLDesc(d, libvirt.DomainXMLInactive)
	if err != nil {
		return nil
	}
	if !strings.Contains(fullXML, `<boot dev='cdrom'/>`) && !strings.Contains(fullXML, `<boot dev="cdrom"/>`) {
		newXML := strings.Replace(fullXML, `<boot dev='hd'/>`, `<boot dev='cdrom'/><boot dev='hd'/>`, 1)
		if newXML != fullXML {
			s.l.DomainDefineXML(newXML)
		}
	}
	return nil
}

// findCdromBus detects the first cdrom's target dev and bus from VM XML. Caller must hold s.mu.
func (s *LibvirtService) findCdromBus(d libvirt.Domain) (dev, bus string) {
	dev, bus = "hda", "ide" // default for i440fx
	xmlStr, err := s.l.DomainGetXMLDesc(d, libvirt.DomainXMLInactive)
	if err != nil {
		return
	}
	var dx detailDomainXML
	if xml.Unmarshal([]byte(xmlStr), &dx) == nil {
		for _, disk := range dx.Devices.Disks {
			if disk.Device == "cdrom" {
				return disk.Target.Dev, disk.Target.Bus
			}
		}
		// No cdrom found: check if q35 (no IDE support)
		if strings.Contains(xmlStr, "q35") {
			return "sdb", "sata"
		}
	}
	return
}

func (s *LibvirtService) DetachISO(vmName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}

	cdromDev, cdromBus := s.findCdromBus(d)

	xmlDef := fmt.Sprintf(`<disk type='file' device='cdrom'>
  <driver name='qemu' type='raw'/>
  <target dev='%s' bus='%s'/>
  <readonly/>
</disk>`, cdromDev, cdromBus)

	state, _, _, _, _, _ := s.l.DomainGetInfo(d)
	var flags libvirt.DomainDeviceModifyFlags
	if libvirt.DomainState(state) == libvirt.DomainRunning {
		flags = libvirt.DomainDeviceModifyLive | libvirt.DomainDeviceModifyConfig
	} else {
		flags = libvirt.DomainDeviceModifyConfig
	}
	return s.l.DomainUpdateDeviceFlags(d, xmlDef, flags)
}

func (s *LibvirtService) CloneVM(srcName string, req model.CloneVMRequest) error {
	s.mu.Lock()
	if err := s.ensureConnected(); err != nil {
		s.mu.Unlock()
		return err
	}
	if !safeNameRe.MatchString(req.NewName) {
		s.mu.Unlock()
		return fmt.Errorf("invalid vm name: %s", req.NewName)
	}
	if !safeNameRe.MatchString(srcName) {
		s.mu.Unlock()
		return fmt.Errorf("invalid source vm name: %s", srcName)
	}
	s.mu.Unlock()
	// virt-clone runs outside the lock — it may take minutes for large disks
	cmd := exec.Command("virt-clone",
		"--original", srcName,
		"--name", req.NewName,
		"--auto-clone",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("clone failed: %s", string(output))
	}
	return nil
}
