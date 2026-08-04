package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	configloader "github.com/vernal96/go-cms/kernel/config"
)

// Load reads an optional local .env file, then loads the project configuration
// from the process environment. Values already present in the environment take
// precedence over values from .env.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	return configloader.Load[Config]("")
}
