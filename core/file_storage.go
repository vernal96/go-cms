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
}
