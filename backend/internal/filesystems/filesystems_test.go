package filesystems_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	configloader "github.com/vernal96/go-cms-kernel/config"
	"github.com/vernal96/go-cms-kernel/filesystem"
	"github.com/vernal96/go-cms/internal/connectors/corefiles"
	privatefiles "github.com/vernal96/go-cms/internal/filesystems/private"
	publicfiles "github.com/vernal96/go-cms/internal/filesystems/public"
)

func TestDeclarationsMixCodeAndEnvironmentAndKeepObjectsIsolated(t *testing.T) {
	ctx := context.Background()
	t.Setenv("FILES_PUBLIC_ROOT", t.TempDir())
	t.Setenv("FILES_PUBLIC_BASE_URL", "https://public.example")
	config, err := configloader.Load[publicfiles.Config]("FILES_PUBLIC")
	if err != nil {
		t.Fatal(err)
	}
	factories := []filesystem.Factory{
		publicfiles.NewFactory(*config),
		privatefiles.NewFactory(privatefiles.Config{
			Root: t.TempDir(), BaseURL: "https://private.example", SigningKey: strings.Repeat("a", 32),
		}),
	}
	manager, err := filesystem.NewManager(ctx, factories)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	expected := []filesystem.DiskInfo{
		{Code: "public", Label: "Публичные файлы", Visibility: filesystem.VisibilityPublic},
		{Code: "private", Label: "Приватные файлы", Visibility: filesystem.VisibilityPrivate},
	}
	for index, info := range manager.Disks() {
		if info != expected[index] {
			t.Fatalf("disk = %#v", info)
		}
		disk, ok := manager.Disk(info.Code)
		if !ok {
			t.Fatalf("missing disk %q", info.Code)
		}
		if err := disk.PutNew(ctx, "same/path.txt", strings.NewReader(string(info.Code)), "text/plain"); err != nil {
			t.Fatal(err)
		}
	}
	for _, info := range expected {
		disk, _ := manager.Disk(info.Code)
		body, err := disk.Open(ctx, "same/path.txt")
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(body)
		closeErr := body.Close()
		if err != nil || closeErr != nil || string(content) != string(info.Code) {
			t.Fatalf("cross-disk content: %q, %v, %v", content, err, closeErr)
		}
	}
}

func TestPrivateDeclarationRequiresSigningKey(t *testing.T) {
	t.Setenv("FILES_PRIVATE_SIGNING_KEY", "")
	if err := os.Unsetenv("FILES_PRIVATE_SIGNING_KEY"); err != nil {
		t.Fatal(err)
	}
	if _, err := configloader.Load[privatefiles.Config]("FILES_PRIVATE"); err == nil {
		t.Fatal("missing signing key accepted")
	}
	_, err := privatefiles.NewFactory(privatefiles.Config{
		Root: t.TempDir(), BaseURL: "https://private.example", SigningKey: "short",
	}).Open(context.Background())
	if err == nil {
		t.Fatal("invalid signing key accepted")
	}
}

func TestInvalidDeclarationsFailBeforeOpeningStorage(t *testing.T) {
	for _, config := range []corefiles.Config{
		{Code: "media", Label: "Media", Driver: "unsupported", Visibility: filesystem.VisibilityPublic},
		{Code: "media", Label: "Media", Driver: "local", Visibility: "invalid"},
		{Code: "media", Label: "Media", Driver: "local", Visibility: filesystem.VisibilityPublic},
		{Code: "media", Label: "Media", Driver: "s3", Visibility: filesystem.VisibilityPublic},
	} {
		if _, err := corefiles.NewFactory(config).Open(context.Background()); err == nil {
			t.Fatalf("invalid disk accepted: %s/%s", config.Driver, config.Visibility)
		}
	}
}
