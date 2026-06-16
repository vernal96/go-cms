package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

type FileDisk string

const (
	FileDiskDefault FileDisk = "default"
	FileDiskNull    FileDisk = "null"
)

var ErrFileNotFound = errors.New("file not found")

type FileStorage interface {
	Save(ctx context.Context, path string, content io.Reader) error
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
}

type FileStorageManager interface {
	Disk(name FileDisk) (FileStorage, error)
	Default() (FileStorage, error)
	Null() FileStorage

	RegisterDisk(name FileDisk, storage FileStorage) error
	SetDefaultDisk(name FileDisk) error
}

type DefaultFileStorageManager struct {
	mu          sync.RWMutex
	disks       map[FileDisk]FileStorage
	defaultDisk FileDisk
	nullDisk    FileStorage
}

func NewDefaultFileStorageManager() *DefaultFileStorageManager {
	nullDisk := NullFileStorage{}

	return &DefaultFileStorageManager{
		disks: map[FileDisk]FileStorage{
			FileDiskNull: nullDisk,
		},
		defaultDisk: FileDiskNull,
		nullDisk:    nullDisk,
	}
}

func (m *DefaultFileStorageManager) Disk(name FileDisk) (FileStorage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if name == FileDiskDefault {
		name = m.defaultDisk
	}

	disk, exists := m.disks[name]
	if !exists {
		return nil, fmt.Errorf("file disk %q is not registered", name)
	}

	return disk, nil
}

func (m *DefaultFileStorageManager) Default() (FileStorage, error) {
	return m.Disk(FileDiskDefault)
}

func (m *DefaultFileStorageManager) Null() FileStorage {
	return m.nullDisk
}

func (m *DefaultFileStorageManager) RegisterDisk(name FileDisk, storage FileStorage) error {
	if name == "" {
		return errors.New("file disk name is empty")
	}

	if name == FileDiskDefault {
		return errors.New("file disk name \"default\" is reserved")
	}

	if name == FileDiskNull {
		return errors.New("file disk name \"null\" is reserved")
	}

	if storage == nil {
		return errors.New("file storage is nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.disks[name]; exists {
		return fmt.Errorf("file disk %q is already registered", name)
	}

	m.disks[name] = storage

	return nil
}

func (m *DefaultFileStorageManager) SetDefaultDisk(name FileDisk) error {
	if name == FileDiskDefault {
		return errors.New("file disk name \"default\" cannot be used as default target")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.disks[name]; !exists {
		return fmt.Errorf("file disk %q is not registered", name)
	}

	m.defaultDisk = name

	return nil
}

type NullFileStorage struct{}

func (s NullFileStorage) Save(ctx context.Context, path string, content io.Reader) error {
	return nil
}

func (s NullFileStorage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, ErrFileNotFound
}

func (s NullFileStorage) Delete(ctx context.Context, path string) error {
	return nil
}

func (s NullFileStorage) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}
