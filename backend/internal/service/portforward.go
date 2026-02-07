package service

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type PortForward struct {
	ID          string `json:"id"`
	Protocol    string `json:"protocol"`
	HostPort    int    `json:"host_port"`
	HostPortEnd int    `json:"host_port_end,omitempty"` // >0 means range
	VMIP        string `json:"vm_ip"`
	VMPort      int    `json:"vm_port"`
	Comment     string `json:"comment"`
}

const pfFile = "/etc/virtpanel/portforwards.json"

var pfMu sync.Mutex

func loadPF() ([]PortForward, error) {
	data, err := os.ReadFile(pfFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rules []PortForward
	return rules, json.Unmarshal(data, &rules)
}

func savePF(rules []PortForward) error {
	os.MkdirAll("/etc/virtpanel", 0755)
	data, _ := json.MarshalIndent(rules, "", "  ")
	return os.WriteFile(pfFile, data, 0644)
}

// checkPortConflict checks if any port in the range is used by the host
func checkPortConflict(protocol string, start, end int) error {
	for port := start; port <= end; port++ {
		addr := fmt.Sprintf(":%d", port)
		if protocol == "tcp" {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("宿主机端口 %d/tcp 已被占用", port)
			}
			ln.Close()
		} else {
			ln, err := net.ListenPacket("udp", addr)
			if err != nil {
				return fmt.Errorf("宿主机端口 %d/udp 已被占用", port)
			}
			ln.Close()
		}
	}
	return nil
}

func (s *LibvirtService) ListPortForwards() ([]PortForward, error) {
	pfMu.Lock()
	defer pfMu.Unlock()
	rules, err := loadPF()
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []PortForward{}
	}
	return rules, nil
}

func (s *LibvirtService) AddPortForward(pf PortForward) error {
	pfMu.Lock()
	defer pfMu.Unlock()

	end := pf.HostPort
	if pf.HostPortEnd > 0 {
		end = pf.HostPortEnd
	}

	rules, _ := loadPF()
	// check overlap with existing rules
	for _, r := range rules {
		rEnd := r.HostPort
		if r.HostPortEnd > 0 {
			rEnd = r.HostPortEnd
		}
		if r.Protocol == pf.Protocol && pf.HostPort <= rEnd && end >= r.HostPort {
			return fmt.Errorf("端口范围与已有规则冲突")
		}
	}

	// check host port conflict
	if err := checkPortConflict(pf.Protocol, pf.HostPort, end); err != nil {
		return err
	}

	if err := addIptablesRule(pf); err != nil {
		return err
	}

	pf.ID = fmt.Sprintf("%s-%d-%d-%s-%d", pf.Protocol, pf.HostPort, end, pf.VMIP, pf.VMPort)
	rules = append(rules, pf)
	return savePF(rules)
}

func (s *LibvirtService) DeletePortForward(id string) error {
	pfMu.Lock()
	defer pfMu.Unlock()

	rules, _ := loadPF()
	var target *PortForward
	var newRules []PortForward
	for _, r := range rules {
		if r.ID == id {
			r2 := r
			target = &r2
		} else {
			newRules = append(newRules, r)
		}
	}
	if target == nil {
		return fmt.Errorf("规则不存在")
	}

	removeIptablesRule(*target)
	if newRules == nil {
		newRules = []PortForward{}
	}
	return savePF(newRules)
}

func dportArg(pf PortForward) string {
	if pf.HostPortEnd > 0 {
		return fmt.Sprintf("%d:%d", pf.HostPort, pf.HostPortEnd)
	}
	return fmt.Sprintf("%d", pf.HostPort)
}

func destArg(pf PortForward) string {
	if pf.HostPortEnd > 0 {
		return fmt.Sprintf("%s:%d-%d", pf.VMIP, pf.VMPort, pf.VMPort+(pf.HostPortEnd-pf.HostPort))
	}
	return fmt.Sprintf("%s:%d", pf.VMIP, pf.VMPort)
}

func fwdDportArg(pf PortForward) string {
	if pf.HostPortEnd > 0 {
		vmEnd := pf.VMPort + (pf.HostPortEnd - pf.HostPort)
		return fmt.Sprintf("%d:%d", pf.VMPort, vmEnd)
	}
	return fmt.Sprintf("%d", pf.VMPort)
}

func addIptablesRule(pf PortForward) error {
	dnat := exec.Command("iptables", "-t", "nat", "-A", "PREROUTING",
		"-p", pf.Protocol, "--dport", dportArg(pf),
		"-j", "DNAT", "--to-destination", destArg(pf))
	if out, err := dnat.CombinedOutput(); err != nil {
		return fmt.Errorf("DNAT 规则添加失败: %s", strings.TrimSpace(string(out)))
	}

	exec.Command("iptables", "-I", "FORWARD",
		"-p", pf.Protocol, "-d", pf.VMIP, "--dport", fwdDportArg(pf),
		"-j", "ACCEPT").Run()

	return nil
}

func removeIptablesRule(pf PortForward) {
	exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
		"-p", pf.Protocol, "--dport", dportArg(pf),
		"-j", "DNAT", "--to-destination", destArg(pf)).Run()

	exec.Command("iptables", "-D", "FORWARD",
		"-p", pf.Protocol, "-d", pf.VMIP, "--dport", fwdDportArg(pf),
		"-j", "ACCEPT").Run()
}

// RestorePortForwards re-applies all saved rules (call on startup)
func (s *LibvirtService) RestorePortForwards() {
	rules, _ := loadPF()
	for _, r := range rules {
		addIptablesRule(r)
	}
}
