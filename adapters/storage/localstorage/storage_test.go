package localstorage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestStorageLifecycle(t *testing.T) {
	t.Parallel()

	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	path := "images/example.txt"

	if err := storage.Save(ctx, path, bytes.NewBufferString("content")); err != nil {
		t.Fatal(err)
	}

	exists, err := storage.Exists(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("saved file does not exist")
	}

	file, err := storage.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "content" {
		t.Fatalf("unexpected content: %q", content)
	}

	if err := storage.Delete(ctx, path); err != nil {
		t.Fatal(err)
	}
	if err := storage.Delete(ctx, path); err != nil {
		t.Fatal(err)
	}

	_, err = storage.Open(ctx, path)
	if !errors.Is(err, core.ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

func TestStorageRejectsTraversal(t *testing.T) {
	t.Parallel()

	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := storage.Save(context.Background(), "../outside.txt", bytes.NewBufferString("content")); err == nil {
		t.Fatal("expected path traversal error")
	}
}
