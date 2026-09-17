package config

import configloader "github.com/vernal96/go-cms-kernel/config"

// Load reads an optional local .env file, then loads the project configuration
// from the process environment. Values already present in the environment take
// precedence over values from .env.
func Load() (*Config, error) {
	return configloader.LoadDotEnv[Config]("")
}
