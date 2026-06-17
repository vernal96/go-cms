package project

import (
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type FileStorageManager struct {
	disks map[core.FileDisk]core.FileStorage
}

func NewFileStorageManager(registrations []FileDiskRegistration) (*FileStorageManager, error) {
	disks := make(map[core.FileDisk]core.FileStorage, len(registrations))

	for _, registration := range registrations {
		if registration.Name == "" {
			return nil, errors.New("file disk name is empty")
		}

		if registration.Storage == nil {
			return nil, fmt.Errorf("file disk %q storage is nil", registration.Name)
		}

		if _, exists := disks[registration.Name]; exists {
			return nil, fmt.Errorf("file disk %q is already registered", registration.Name)
		}

		disks[registration.Name] = registration.Storage
	}

	return &FileStorageManager{
		disks: disks,
	}, nil
}

func (m *FileStorageManager) Disk(name core.FileDisk) (core.FileStorage, error) {
	if name == "" {
		return nil, errors.New("file disk name is empty")
	}

	disk, exists := m.disks[name]
	if !exists {
		return nil, fmt.Errorf("file disk %q is not registered", name)
	}

	return disk, nil
}

func (m *FileStorageManager) Null() core.FileStorage {
	return core.NullFileStorage{}
}

var _ core.FileStorageManager = (*FileStorageManager)(nil)
