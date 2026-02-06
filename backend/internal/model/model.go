package model

type VM struct {
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	State  string `json:"state"`
	CPU    int    `json:"cpu"`
	Memory int    `json:"memory"` // MB
}

type CreateVMRequest struct {
	Name      string `json:"name" binding:"required"`
	CPU       int    `json:"cpu"`
	Memory    int    `json:"memory"`     // MB
	Disk      int    `json:"disk"`       // GB
	OSType    string `json:"os_type"`    // preset: "linux","windows","legacy"
	DiskBus   string `json:"disk_bus"`   // virtio,sata,scsi,ide (overrides os_type)
	NetModel  string `json:"net_model"`  // virtio,e1000,rtl8139 (overrides os_type)
	Machine   string `json:"machine"`    // i440fx,q35 (default: q35 for windows, i440fx for others)
	CPUModel  string `json:"cpu_model"`  // host-passthrough,host-model,qemu64 (default: host-passthrough for windows)
	Clock     string `json:"clock"`      // utc,localtime (default: localtime for windows)
	VirtioISO string `json:"virtio_iso"` // optional second ISO for virtio drivers
}

type HostInfo struct {
	Hostname    string `json:"hostname"`
	CPUModel    string `json:"cpu_model"`
	CPUCount    int    `json:"cpu_count"`
	MemoryTotal int    `json:"memory_total"` // MB
	MemoryFree  int    `json:"memory_free"`  // MB
	VMRunning   int    `json:"vm_running"`
	VMTotal     int    `json:"vm_total"`
}

type Network struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Active  bool   `json:"active"`
	Forward string `json:"forward"`
	Bridge  string `json:"bridge"`
	Subnet  string `json:"subnet"`
}

type CreateNetworkRequest struct {
	Name      string `json:"name" binding:"required"`
	Bridge    string `json:"bridge"`
	Subnet    string `json:"subnet"`
	Netmask   string `json:"netmask"`
	DHCPStart string `json:"dhcp_start"`
	DHCPEnd   string `json:"dhcp_end"`
}

type StoragePool struct {
	Name       string `json:"name"`
	UUID       string `json:"uuid"`
	Active     bool   `json:"active"`
	Type       string `json:"type"`
	Path       string `json:"path"`
	Capacity   uint64 `json:"capacity"`   // GB
	Allocation uint64 `json:"allocation"` // GB
	Available  uint64 `json:"available"`  // GB
}

type CreateStoragePoolRequest struct {
	Name string `json:"name" binding:"required"`
	Path string `json:"path"`
}

type Snapshot struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedAt   int64  `json:"created_at"`
	IsCurrent   bool   `json:"is_current"`
}

type CreateSnapshotRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type StorageVolume struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Capacity   uint64 `json:"capacity"`   // GB
	Allocation uint64 `json:"allocation"` // GB
}

type CreateVolumeRequest struct {
	Pool     string `json:"pool" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Capacity int    `json:"capacity"` // GB
	Format   string `json:"format"`   // qcow2, raw
}

type ISOFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"` // bytes
}

type UpdateVMRequest struct {
	CPU    int `json:"cpu"`
	Memory int `json:"memory"` // MB
}

type VMDetail struct {
	Name   string       `json:"name"`
	UUID   string       `json:"uuid"`
	State  string       `json:"state"`
	CPU    int          `json:"cpu"`
	Memory int          `json:"memory"`
	Disks  []VMDisk     `json:"disks"`
	NICs   []VMNIC      `json:"nics"`
	Boot   string       `json:"boot"`
	Arch   string       `json:"arch"`
}

type VMDisk struct {
	Device string `json:"device"`
	Source string `json:"source"`
	Target string `json:"target"`
	Bus    string `json:"bus"`
	Format string `json:"format"`
}

type VMNIC struct {
	Type    string `json:"type"`
	Source  string `json:"source"`
	MAC     string `json:"mac"`
	Model   string `json:"model"`
}

type AttachDiskRequest struct {
	Source string `json:"source" binding:"required"` // disk image path
	Target string `json:"target"`                    // vdb, vdc...
	Bus    string `json:"bus"`                       // virtio, ide, scsi
}

type AttachNICRequest struct {
	Network string `json:"network" binding:"required"`
	Model   string `json:"model"` // virtio
}

type AttachISORequest struct {
	Path string `json:"path" binding:"required"` // ISO file path
}

type CloneVMRequest struct {
	NewName string `json:"new_name" binding:"required"`
}
