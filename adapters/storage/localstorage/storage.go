package localstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vernal96/go-cms/core"
)

type Storage struct {
	root string
}

func NewStorage(root string) (*Storage, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("local storage root is empty")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local storage root: %w", err)
	}

	return &Storage{
		root: filepath.Clean(absoluteRoot),
	}, nil
}

func (s *Storage) Save(ctx context.Context, path string, content io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if content == nil {
		return errors.New("local storage content is nil")
	}

	filename, err := s.resolve(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create local storage directory: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create local storage file: %w", err)
	}

	_, copyErr := io.Copy(file, content)
	closeErr := file.Close()

	if copyErr != nil {
		return fmt.Errorf("write local storage file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close local storage file: %w", closeErr)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	filename, err := s.resolve(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, core.ErrFileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open local storage file: %w", err)
	}

	return file, nil
}

func (s *Storage) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	filename, err := s.resolve(path)
	if err != nil {
		return err
	}

	if err := os.Remove(filename); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("delete local storage file: %w", err)
	}

	return nil
}

func (s *Storage) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	filename, err := s.resolve(path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat local storage file: %w", err)
	}

	return true, nil
}

func (s *Storage) resolve(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("local storage path is empty")
	}

	cleanPath := filepath.Clean(path)
	if cleanPath == "." || filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("local storage path %q is invalid", path)
	}

	filename := filepath.Join(s.root, cleanPath)
	relativePath, err := filepath.Rel(s.root, filename)
	if err != nil {
		return "", fmt.Errorf("resolve local storage path %q: %w", path, err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("local storage path %q escapes storage root", path)
	}

	return filename, nil
}

var _ core.FileStorage = (*Storage)(nil)
