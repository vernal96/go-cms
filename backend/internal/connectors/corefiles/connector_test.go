package corefiles

import (
	"testing"

	"github.com/vernal96/go-cms/kernel/filesystem"
)

func TestConfigsDecodeBuildsArbitraryNamedDiskFactories(t *testing.T) {
	var configs Configs
	if err := configs.Decode(`[
		{"code":"media","label":"Медиатека","driver":"s3","visibility":"public","s3":{"region":"eu-central-1","bucket":"cms-media"}},
		{"code":"archive","label":"Архив","driver":"local","visibility":"private","local":{"root":"var/archive","base_url":"http://localhost:8080","signing_key":"0123456789abcdef0123456789abcdef"}}
	]`); err != nil {
		t.Fatal(err)
	}
	if len(configs) != 2 {
		t.Fatalf("configs = %#v", configs)
	}
	if configs[0].Code != "media" || configs[0].Label != "Медиатека" || configs[0].Visibility != filesystem.VisibilityPublic {
		t.Fatalf("media config = %#v", configs[0])
	}
	if configs[1].Code != "archive" || configs[1].Label != "Архив" || configs[1].Visibility != filesystem.VisibilityPrivate {
		t.Fatalf("archive config = %#v", configs[1])
	}

	factories := configs.Factories()
	if len(factories) != 2 || factories[0].Code() != "media" || factories[1].Code() != "archive" {
		t.Fatalf("factories = %#v", factories)
	}
	mediaLabel, ok := factories[0].(filesystem.FactoryLabelProvider)
	if !ok || mediaLabel.Label() != "Медиатека" {
		t.Fatalf("media label provider = %#v, %t", mediaLabel, ok)
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
