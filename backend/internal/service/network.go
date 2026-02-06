package service

import (
	"encoding/xml"
	"fmt"
	"net"

	"kvmmm/internal/model"
)

type networkXML struct {
	XMLName xml.Name `xml:"network"`
	Name    string   `xml:"name"`
	Forward *struct {
		Mode string `xml:"mode,attr"`
	} `xml:"forward"`
	Bridge *struct {
		Name string `xml:"name,attr"`
	} `xml:"bridge"`
	IP *struct {
		Address string `xml:"address,attr"`
		Netmask string `xml:"netmask,attr"`
		DHCP    *struct {
			Range struct {
				Start string `xml:"start,attr"`
				End   string `xml:"end,attr"`
			} `xml:"range"`
		} `xml:"dhcp"`
	} `xml:"ip"`
}

func (s *LibvirtService) ListNetworks() ([]model.Network, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	nets, _, err := s.l.ConnectListAllNetworks(-1, 0)
	if err != nil {
		return nil, err
	}
	result := make([]model.Network, 0, len(nets))
	for _, n := range nets {
		active, err := s.l.NetworkIsActive(n)
		if err != nil {
			continue
		}
		xmlStr, err := s.l.NetworkGetXMLDesc(n, 0)
		if err != nil {
			continue
		}
		var nx networkXML
		xml.Unmarshal([]byte(xmlStr), &nx)

		nw := model.Network{
			Name:   n.Name,
			UUID:   fmt.Sprintf("%x", n.UUID),
			Active: active == 1,
		}
		if nx.Forward != nil {
			nw.Forward = nx.Forward.Mode
		}
		if nx.Bridge != nil {
			nw.Bridge = nx.Bridge.Name
		}
		if nx.IP != nil {
			mask := net.ParseIP(nx.IP.Netmask)
			if mask != nil {
				mask4 := mask.To4()
				if mask4 != nil {
					ones, _ := net.IPMask(mask4).Size()
					nw.Subnet = fmt.Sprintf("%s/%d", nx.IP.Address, ones)
				} else {
					nw.Subnet = nx.IP.Address
				}
			} else {
				nw.Subnet = nx.IP.Address
			}
		}
		result = append(result, nw)
	}
	return result, nil
}

func (s *LibvirtService) StartNetwork(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	n, err := s.l.NetworkLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.NetworkCreate(n)
}

func (s *LibvirtService) StopNetwork(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	n, err := s.l.NetworkLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.NetworkDestroy(n)
}

func (s *LibvirtService) DeleteNetwork(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	n, err := s.l.NetworkLookupByName(name)
	if err != nil {
		return err
	}
	_ = s.l.NetworkDestroy(n)
	return s.l.NetworkUndefine(n)
}

func (s *LibvirtService) CreateNetwork(req model.CreateNetworkRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if !safeNameRe.MatchString(req.Name) {
		return fmt.Errorf("invalid network name: %s", req.Name)
	}
	if req.Bridge == "" {
		req.Bridge = "virbr-" + req.Name
	}
	if !safeNameRe.MatchString(req.Bridge) {
		return fmt.Errorf("invalid bridge name: %s", req.Bridge)
	}
	if req.Subnet == "" {
		req.Subnet = "192.168.100.1"
	}
	if req.Netmask == "" {
		req.Netmask = "255.255.255.0"
	}
	if req.DHCPStart == "" {
		req.DHCPStart = "192.168.100.100"
	}
	if req.DHCPEnd == "" {
		req.DHCPEnd = "192.168.100.200"
	}
	// Validate IP addresses
	for _, ip := range []string{req.Subnet, req.Netmask, req.DHCPStart, req.DHCPEnd} {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
	}

	xmlDef := fmt.Sprintf(`<network>
  <name>%s</name>
  <forward mode='nat'/>
  <bridge name='%s' stp='on' delay='0'/>
  <ip address='%s' netmask='%s'>
    <dhcp>
      <range start='%s' end='%s'/>
    </dhcp>
  </ip>
</network>`, req.Name, req.Bridge, req.Subnet, req.Netmask, req.DHCPStart, req.DHCPEnd)

	_, err := s.l.NetworkDefineXML(xmlDef)
	return err
}
