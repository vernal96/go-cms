package core

import (
	"context"
	"io"
)

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

var _ FileStorage = NullFileStorage{}
