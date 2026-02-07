package service

import (
	"fmt"
	"io"
	"virtpanel/internal/model"
	"os"
	"path/filepath"
	"strings"
)

const isoDir = "/var/lib/libvirt/images/iso"

func (s *LibvirtService) ListISOs() ([]model.ISOFile, error) {
	os.MkdirAll(isoDir, 0755)
	entries, err := os.ReadDir(isoDir)
	if err != nil {
		return nil, err
	}
	result := make([]model.ISOFile, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".iso") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, model.ISOFile{
			Name: e.Name(),
			Path: filepath.Join(isoDir, e.Name()),
			Size: info.Size(),
		})
	}
	return result, nil
}

func (s *LibvirtService) UploadISO(filename string, reader io.Reader) error {
	filename = filepath.Base(filename)
	if !strings.HasSuffix(strings.ToLower(filename), ".iso") {
		return fmt.Errorf("only .iso files allowed")
	}
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	if !safeNameRe.MatchString(nameWithoutExt) {
		return fmt.Errorf("invalid filename: %s", filename)
	}
	os.MkdirAll(isoDir, 0755)
	dstPath := filepath.Join(isoDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, reader)
	dst.Close()
	if err != nil {
		os.Remove(dstPath)
	}
	return err
}

func (s *LibvirtService) DeleteISO(filename string) error {
	base := filepath.Base(filename)
	if !strings.HasSuffix(strings.ToLower(base), ".iso") {
		return fmt.Errorf("not an iso file: %s", base)
	}
	path := filepath.Join(isoDir, base)
	// Ensure resolved path is still under isoDir
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(isoDir)+string(filepath.Separator)) {
		return fmt.Errorf("invalid filename")
	}
	return os.Remove(path)
}
