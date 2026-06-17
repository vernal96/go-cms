package core

import (
	"context"
	"errors"
	"io"
)

type FileDisk string

var ErrFileNotFound = errors.New("file not found")

type FileStorage interface {
	Save(ctx context.Context, path string, content io.Reader) error
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
}

type FileStorageManager interface {
	Disk(name FileDisk) (FileStorage, error)
	Null() FileStorage
}

type NullFileStorageManager struct{}

func (m NullFileStorageManager) Disk(name FileDisk) (FileStorage, error) {
	return NullFileStorage{}, nil
}

func (m NullFileStorageManager) Null() FileStorage {
	return NullFileStorage{}
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
