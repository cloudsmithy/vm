package service

import (
	"fmt"
	"kvmmm/internal/model"
)

func (s *LibvirtService) ListVolumes(poolName string) ([]model.StorageVolume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	pool, err := s.l.StoragePoolLookupByName(poolName)
	if err != nil {
		return nil, err
	}
	_ = s.l.StoragePoolRefresh(pool, 0)

	vols, _, err := s.l.StoragePoolListAllVolumes(pool, -1, 0)
	if err != nil {
		return nil, err
	}
	result := make([]model.StorageVolume, 0, len(vols))
	for _, v := range vols {
		vType, capacity, allocation, err := s.l.StorageVolGetInfo(v)
		if err != nil {
			continue
		}
		path, _ := s.l.StorageVolGetPath(v)
		typeName := "unknown"
		switch vType {
		case 0:
			typeName = "file"
		case 1:
			typeName = "block"
		case 2:
			typeName = "dir"
		case 3:
			typeName = "network"
		}
		result = append(result, model.StorageVolume{
			Name:       v.Name,
			Path:       path,
			Type:       typeName,
			Capacity:   (capacity + 1024*1024*1024 - 1) / (1024 * 1024 * 1024),
			Allocation: (allocation + 1024*1024*1024 - 1) / (1024 * 1024 * 1024),
		})
	}
	return result, nil
}

func (s *LibvirtService) CreateVolume(req model.CreateVolumeRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if !safeNameRe.MatchString(req.Name) {
		return fmt.Errorf("invalid volume name: %s", req.Name)
	}
	if !safeNameRe.MatchString(req.Pool) {
		return fmt.Errorf("invalid pool name: %s", req.Pool)
	}
	pool, err := s.l.StoragePoolLookupByName(req.Pool)
	if err != nil {
		return err
	}
	if req.Capacity <= 0 {
		req.Capacity = 20
	}
	if req.Format == "" {
		req.Format = "qcow2"
	}
	validFormat := map[string]bool{"qcow2": true, "raw": true, "vmdk": true, "vdi": true}
	if !validFormat[req.Format] {
		return fmt.Errorf("invalid format: %s", req.Format)
	}
	xmlDef := fmt.Sprintf(`<volume>
  <name>%s</name>
  <capacity unit='G'>%d</capacity>
  <target><format type='%s'/></target>
</volume>`, req.Name, req.Capacity, req.Format)

	_, err = s.l.StorageVolCreateXML(pool, xmlDef, 0)
	return err
}

func (s *LibvirtService) DeleteVolume(poolName, volName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	pool, err := s.l.StoragePoolLookupByName(poolName)
	if err != nil {
		return err
	}
	vol, err := s.l.StorageVolLookupByName(pool, volName)
	if err != nil {
		return err
	}
	return s.l.StorageVolDelete(vol, 0)
}
