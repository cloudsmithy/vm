package service

import (
	"encoding/xml"
	"fmt"
	"html"
	"kvmmm/internal/model"
)

type snapshotXML struct {
	XMLName     xml.Name `xml:"domainsnapshot"`
	Name        string   `xml:"name"`
	Description string   `xml:"description"`
	State       string   `xml:"state"`
	CreationTime int64   `xml:"creationTime"`
}

func (s *LibvirtService) ListSnapshots(vmName string) ([]model.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return nil, err
	}
	snaps, _, err := s.l.DomainListAllSnapshots(d, -1, 0)
	if err != nil {
		return nil, err
	}

	current, currentErr := s.l.DomainSnapshotCurrent(d, 0)

	result := make([]model.Snapshot, 0, len(snaps))
	for _, snap := range snaps {
		xmlStr, err := s.l.DomainSnapshotGetXMLDesc(snap, 0)
		if err != nil {
			continue
		}
		var sx snapshotXML
		xml.Unmarshal([]byte(xmlStr), &sx)
		result = append(result, model.Snapshot{
			Name:        sx.Name,
			Description: sx.Description,
			State:       sx.State,
			CreatedAt:   sx.CreationTime,
			IsCurrent:   currentErr == nil && snap.Name == current.Name,
		})
	}
	return result, nil
}

func (s *LibvirtService) CreateSnapshot(vmName string, req model.CreateSnapshotRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if !safeNameRe.MatchString(req.Name) {
		return fmt.Errorf("invalid snapshot name: %s", req.Name)
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}
	xmlDef := fmt.Sprintf(`<domainsnapshot><name>%s</name><description>%s</description></domainsnapshot>`, req.Name, html.EscapeString(req.Description))
	_, err = s.l.DomainSnapshotCreateXML(d, xmlDef, 0)
	return err
}

func (s *LibvirtService) DeleteSnapshot(vmName, snapName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}
	snap, err := s.l.DomainSnapshotLookupByName(d, snapName, 0)
	if err != nil {
		return err
	}
	return s.l.DomainSnapshotDelete(snap, 0)
}

func (s *LibvirtService) RevertSnapshot(vmName, snapName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return err
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		return err
	}
	snap, err := s.l.DomainSnapshotLookupByName(d, snapName, 0)
	if err != nil {
		return err
	}
	return s.l.DomainRevertToSnapshot(snap, 0)
}
