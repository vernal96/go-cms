package private

import (
	"github.com/vernal96/go-cms/internal/connectors/corefiles"
	"github.com/vernal96/go-cms/kernel/filesystem"
)

const Code filesystem.Code = "private"

// Config contains only the settings this declaration obtains from the environment.
// Callers may also supply these values directly in Go.
type Config struct {
	Root       string `envconfig:"ROOT" default:"var/files/private"`
	BaseURL    string `envconfig:"BASE_URL" default:"http://localhost:8080"`
	SigningKey string `envconfig:"SIGNING_KEY" required:"true"`
}

// NewFactory declares one application-owned physical disk.
func NewFactory(config Config) filesystem.Factory {
	return corefiles.NewFactory(corefiles.Config{
		Code:       Code,
		Label:      "Приватные файлы",
		Driver:     "local",
		Visibility: filesystem.VisibilityPrivate,
		Local: corefiles.LocalConfig{
			Root:       config.Root,
			BaseURL:    config.BaseURL,
			SigningKey: config.SigningKey,
		},
	})
}
