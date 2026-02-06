package service

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

type graphicsXML struct {
	Type   string `xml:"type,attr"`
	Port   string `xml:"port,attr"`
	Listen string `xml:"listen,attr"`
}

type fullDomainXML struct {
	XMLName  xml.Name `xml:"domain"`
	Devices  struct {
		Graphics []graphicsXML `xml:"graphics"`
	} `xml:"devices"`
}

// GetVNCPort returns the VNC port for a running VM
func (s *LibvirtService) GetVNCPort(name string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return 0, err
	}
	d, err := s.l.DomainLookupByName(name)
	if err != nil {
		return 0, err
	}
	xmlStr, err := s.l.DomainGetXMLDesc(d, 0)
	if err != nil {
		return 0, err
	}
	var dx fullDomainXML
	if err := xml.Unmarshal([]byte(xmlStr), &dx); err != nil {
		return 0, err
	}
	for _, g := range dx.Devices.Graphics {
		if g.Type == "vnc" {
			port, err := strconv.Atoi(g.Port)
			if err != nil || port < 0 {
				return 0, fmt.Errorf("vnc port not allocated (vm may not be running)")
			}
			return port, nil
		}
	}
	return 0, fmt.Errorf("no vnc graphics configured")
}
