package service

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"virtpanel/internal/model"
)

type poolXML struct {
	XMLName xml.Name `xml:"pool"`
	Type    string   `xml:"type,attr"`
	Target  struct {
		Path string `xml:"path"`
	} `xml:"target"`
}

func (s *LibvirtService) ListStoragePools() ([]model.StoragePool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	pools, _, err := s.l.ConnectListAllStoragePools(-1, 0)
	if err != nil {
		return nil, err
	}
	result := make([]model.StoragePool, 0, len(pools))
	for _, p := range pools {
		active, err := s.l.StoragePoolIsActive(p)
		if err != nil {
			continue
		}
		xmlStr, err := s.l.StoragePoolGetXMLDesc(p, 0)
		if err != nil {
			continue
		}
		var px poolXML
		xml.Unmarshal([]byte(xmlStr), &px)

		sp := model.StoragePool{
			Name:   p.Name,
			UUID:   fmt.Sprintf("%x", p.UUID),
			Active: active == 1,
			Type:   px.Type,
			Path:   px.Target.Path,
		}

		if active == 1 {
			_, capacity, allocation, available, err := s.l.StoragePoolGetInfo(p)
			if err == nil {
				const gb = 1024 * 1024 * 1024
				sp.Capacity = (capacity + gb - 1) / gb
				sp.Allocation = (allocation + gb - 1) / gb
				sp.Available = available / gb
			}
		}
		result = append(result, sp)
	}
	return result, nil
}

func (s *LibvirtService) StartStoragePool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	p, err := s.l.StoragePoolLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.StoragePoolCreate(p, 0)
}

func (s *LibvirtService) StopStoragePool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	p, err := s.l.StoragePoolLookupByName(name)
	if err != nil {
		return err
	}
	return s.l.StoragePoolDestroy(p)
}

func (s *LibvirtService) DeleteStoragePool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	p, err := s.l.StoragePoolLookupByName(name)
	if err != nil {
		return err
	}
	_ = s.l.StoragePoolDestroy(p)
	return s.l.StoragePoolUndefine(p)
}

func (s *LibvirtService) CreateStoragePool(req model.CreateStoragePoolRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if !safeNameRe.MatchString(req.Name) {
		return fmt.Errorf("invalid pool name: %s", req.Name)
	}
	if req.Path == "" {
		req.Path = "/var/lib/libvirt/images/" + req.Name
	}
	cleanPath := filepath.Clean(req.Path)
	if !strings.HasPrefix(cleanPath, "/var/lib/libvirt/") {
		return fmt.Errorf("pool path must be under /var/lib/libvirt/")
	}

	xmlDef := fmt.Sprintf(`<pool type='dir'>
  <name>%s</name>
  <target>
    <path>%s</path>
  </target>
</pool>`, req.Name, cleanPath)

	_, err := s.l.StoragePoolDefineXML(xmlDef, 0)
	return err
}
