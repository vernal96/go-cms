package corefiles

import (
	"testing"

	"github.com/vernal96/go-cms-kernel/filesystem"
)

func TestArbitraryNamedDiskFactories(t *testing.T) {
	configs := []Config{
		{Code: "media", Label: "Медиатека", Driver: "s3", Visibility: filesystem.VisibilityPublic,
			S3: S3Config{Region: "eu-central-1", Bucket: "cms-media"}},
		{Code: "archive", Label: "Архив", Driver: "local", Visibility: filesystem.VisibilityPrivate,
			Local: LocalConfig{Root: "var/archive", BaseURL: "http://localhost:8080", SigningKey: "0123456789abcdef0123456789abcdef"}},
	}
	for _, config := range configs {
		factory := NewFactory(config)
		if factory.Code() != config.Code || factory.Label() != config.Label {
			t.Fatal("factory lost disk identity or label")
		}
	}
}

func TestDiskConfigRequiresCodeLabelDriverAndVisibility(t *testing.T) {
	tests := []Config{
		{Label: "Media", Driver: "s3", Visibility: filesystem.VisibilityPublic},
		{Code: "media", Driver: "s3", Visibility: filesystem.VisibilityPublic},
		{Code: "media", Label: "Media", Visibility: filesystem.VisibilityPublic},
		{Code: "media", Label: "Media", Driver: "s3", Visibility: "unknown"},
	}
	for index, config := range tests {
		if err := validateConfig(config); err == nil {
			t.Fatalf("config %d unexpectedly valid: %#v", index, config)
		}
	}
}
