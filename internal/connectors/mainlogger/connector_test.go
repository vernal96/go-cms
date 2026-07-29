package mainlogger

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vernal96/go-cms/connectors/filelogger"
	"github.com/vernal96/go-cms/connectors/lokilogger"
)

func validConfig() Config {
	return Config{
		Driver:      "file",
		Level:       "info",
		ServiceName: "cms-test",
		Environment: "test",
		File: FileConfig{
			Path:       "cms.log",
			Timezone:   "UTC",
			MaxSize:    1 << 20,
			MaxBackups: 14,
			MaxAge:     14 * 24 * time.Hour,
			Compress:   true,
		},
		Loki: LokiConfig{
			Endpoint:         "http://localhost:3100",
			ReadinessTimeout: time.Second,
			ExportTimeout:    time.Second,
			ShutdownTimeout:  time.Second,
		},
	}
}

func TestFactoryOpensOnlySelectedBackend(t *testing.T) {
	fileConfig := validConfig()
	fileConfig.File.Path = filepath.Join(t.TempDir(), "cms.log")
	fileConfig.Loki.Endpoint = "not a URL"
	fileConnector, err := NewFactory(fileConfig).Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fileConnector.(*filelogger.Connector); !ok {
		t.Fatalf("file connector type = %T", fileConnector)
	}
	if err := fileConnector.Close(); err != nil {
		t.Fatal(err)
	}

	lokiConfig := validConfig()
	lokiConfig.Driver = "loki"
	lokiConfig.File.Path = ""
	lokiConfig.File.Timezone = "not/a/timezone"
	lokiConnector, err := NewFactory(lokiConfig).Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lokiConnector.(*lokilogger.Connector); !ok {
		t.Fatalf("Loki connector type = %T", lokiConnector)
	}
	if err := lokiConnector.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryRejectsMissingUnknownAndInvalidCommonConfig(
	t *testing.T,
) {
	tests := []struct {
		name     string
		mutate   func(*Config)
		contains string
	}{
		{
			name: "missing driver",
			mutate: func(config *Config) {
				config.Driver = ""
			},
			contains: "driver is required",
		},
		{
			name: "unknown driver",
			mutate: func(config *Config) {
				config.Driver = "console"
			},
			contains: "unsupported logger driver",
		},
		{
			name: "invalid level",
			mutate: func(config *Config) {
				config.Level = "verbose"
			},
			contains: "parse logger level",
		},
		{
			name: "empty service",
			mutate: func(config *Config) {
				config.ServiceName = " "
			},
			contains: "service name is empty",
		},
		{
			name: "empty environment",
			mutate: func(config *Config) {
				config.Environment = " "
			},
			contains: "environment is empty",
		},
		{
			name: "invalid selected file timezone",
			mutate: func(config *Config) {
				config.File.Timezone = "not/a/timezone"
			},
			contains: "load logger timezone",
		},
		{
			name: "missing selected Loki endpoint",
			mutate: func(config *Config) {
				config.Driver = "loki"
				config.Loki.Endpoint = ""
			},
			contains: "endpoint is empty",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			config := validConfig()
			config.File.Path = filepath.Join(t.TempDir(), "cms.log")
			testCase.mutate(&config)
			connector, err := NewFactory(config).Open(context.Background())
			if connector != nil {
				_ = connector.Close()
			}
			if err == nil || !strings.Contains(err.Error(), testCase.contains) {
				t.Fatalf("open error = %v", err)
			}
		})
	}
}
