package service

import (
	"encoding/xml"
	"fmt"
	"html"
	"os/exec"

	"virtpanel/internal/model"

	libvirt "github.com/digitalocean/go-libvirt"
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

// RevertSnapshotToNew reverts a snapshot and clones the result to a new VM.
// Steps: revert snapshot -> clone to newName -> revert back to current snapshot.
func (s *LibvirtService) RevertSnapshotToNew(vmName, snapName, newName string) error {
	if !safeNameRe.MatchString(newName) {
		return fmt.Errorf("invalid vm name: %s", newName)
	}

	// 1. Check VM is shut off
	s.mu.Lock()
	if err := s.ensureConnected(); err != nil {
		s.mu.Unlock()
		return err
	}
	d, err := s.l.DomainLookupByName(vmName)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	state, _, _, _, _, err := s.l.DomainGetInfo(d)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if libvirt.DomainState(state) != libvirt.DomainShutoff {
		s.mu.Unlock()
		return fmt.Errorf("虚拟机必须处于关机状态才能执行此操作")
	}
	currentSnap, currentErr := s.l.DomainSnapshotCurrent(d, 0)

	// 2. Revert to target snapshot
	snap, err := s.l.DomainSnapshotLookupByName(d, snapName, 0)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if err := s.l.DomainRevertToSnapshot(snap, 0); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("revert failed: %w", err)
	}
	s.mu.Unlock()

	// 3. Clone (runs outside lock, may be slow)
	cmd := exec.Command("virt-clone", "--original", vmName, "--name", newName, "--auto-clone")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try to restore original state before returning error
		s.mu.Lock()
		if currentErr == nil {
			d2, _ := s.l.DomainLookupByName(vmName)
			cs, _ := s.l.DomainSnapshotLookupByName(d2, currentSnap.Name, 0)
			s.l.DomainRevertToSnapshot(cs, 0)
		}
		s.mu.Unlock()
		return fmt.Errorf("clone failed: %s", string(output))
	}

	// 4. Restore original VM to its previous snapshot
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConnected(); err != nil {
		return nil // clone succeeded, just can't restore — acceptable
	}
	if currentErr == nil {
		d2, err := s.l.DomainLookupByName(vmName)
		if err == nil {
			cs, err := s.l.DomainSnapshotLookupByName(d2, currentSnap.Name, 0)
			if err == nil {
				s.l.DomainRevertToSnapshot(cs, 0)
			}
		}
	}
	return nil
}
