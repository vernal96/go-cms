package public

import (
	"github.com/vernal96/go-cms/internal/connectors/corefiles"
	"github.com/vernal96/go-cms/kernel/filesystem"
)

const Code filesystem.Code = "public"

// Config contains only the settings this declaration obtains from the environment.
// Callers may also supply these values directly in Go.
type Config struct {
	Root    string `envconfig:"ROOT" default:"var/files/public"`
	BaseURL string `envconfig:"BASE_URL" default:"http://localhost:8080"`
}

// NewFactory declares one application-owned physical disk.
func NewFactory(config Config) filesystem.Factory {
	return corefiles.NewFactory(corefiles.Config{
		Code:       Code,
		Label:      "Публичные файлы",
		Driver:     "local",
		Visibility: filesystem.VisibilityPublic,
		Local: corefiles.LocalConfig{
			Root:    config.Root,
			BaseURL: config.BaseURL,
		},
	})
}
