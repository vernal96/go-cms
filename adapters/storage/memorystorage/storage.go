package memorystorage

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/vernal96/go-cms/core"
)

const DiskName core.FileDisk = "memory"

type Storage struct {
	mu    sync.RWMutex
	files map[string][]byte
}

func NewStorage() *Storage {
	return &Storage{
		files: make(map[string][]byte),
	}
}

func (s *Storage) Save(ctx context.Context, path string, content io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	copiedData := make([]byte, len(data))
	copy(copiedData, data)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.files[path] = copiedData

	return nil
}

func (s *Storage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	data, exists := s.files[path]
	s.mu.RUnlock()

	if !exists {
		return nil, core.ErrFileNotFound
	}

	copiedData := make([]byte, len(data))
	copy(copiedData, data)

	return io.NopCloser(bytes.NewReader(copiedData)), nil
}

func (s *Storage) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.files, path)

	return nil
}

func (s *Storage) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.files[path]

	return exists, nil
}

var _ core.FileStorage = (*Storage)(nil)
