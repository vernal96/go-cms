package mainlogger

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vernal96/go-cms/connectors/filelogger"
	"github.com/vernal96/go-cms/connectors/lokilogger"
	"github.com/vernal96/go-cms/kernel/logging"
)

type Config struct {
	Driver      string     `envconfig:"DRIVER" required:"true"`
	Level       string     `envconfig:"LEVEL" required:"true"`
	ServiceName string     `envconfig:"SERVICE_NAME" required:"true"`
	Environment string     `envconfig:"ENVIRONMENT" required:"true"`
	File        FileConfig `envconfig:"FILE"`
	Loki        LokiConfig `envconfig:"LOKI"`
}

type FileConfig struct {
	Path       string        `envconfig:"PATH" default:"var/log/cms.log"`
	Timezone   string        `envconfig:"TIMEZONE" default:"Local"`
	MaxSize    int64         `envconfig:"MAX_SIZE" default:"104857600"`
	MaxBackups int           `envconfig:"MAX_BACKUPS" default:"14"`
	MaxAge     time.Duration `envconfig:"MAX_AGE" default:"336h"`
	Compress   bool          `envconfig:"COMPRESS" default:"true"`
}

type LokiConfig struct {
	Endpoint         string        `envconfig:"ENDPOINT"`
	ReadinessTimeout time.Duration `envconfig:"READINESS_TIMEOUT" default:"5s"`
	ExportTimeout    time.Duration `envconfig:"EXPORT_TIMEOUT" default:"5s"`
	ShutdownTimeout  time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"5s"`
}

type Factory struct {
	config Config
}

func NewFactory(config Config) Factory {
	return Factory{config: config}
}

func (f Factory) Open(
	ctx context.Context,
) (logging.Connector, error) {
	driver := strings.ToLower(strings.TrimSpace(f.config.Driver))
	switch driver {
	case "file", "loki":
	case "":
		return nil, fmt.Errorf("logger driver is required")
	default:
		return nil, fmt.Errorf(
			"unsupported logger driver %q",
			f.config.Driver,
		)
	}

	level, err := parseLevel(f.config.Level)
	if err != nil {
		return nil, err
	}
	commonService := strings.TrimSpace(f.config.ServiceName)
	commonEnvironment := strings.TrimSpace(f.config.Environment)
	if commonService == "" {
		return nil, fmt.Errorf("logger service name is empty")
	}
	if commonEnvironment == "" {
		return nil, fmt.Errorf("logger environment is empty")
	}

	switch driver {
	case "file":
		location, err := parseLocation(f.config.File.Timezone)
		if err != nil {
			return nil, err
		}
		return filelogger.New(ctx, filelogger.Config{
			Path:        strings.TrimSpace(f.config.File.Path),
			Level:       level,
			ServiceName: commonService,
			Environment: commonEnvironment,
			Location:    location,
			MaxSize:     f.config.File.MaxSize,
			MaxBackups:  f.config.File.MaxBackups,
			MaxAge:      f.config.File.MaxAge,
			Compress:    f.config.File.Compress,
		})
	case "loki":
		return lokilogger.New(ctx, lokilogger.Config{
			Endpoint:         f.config.Loki.Endpoint,
			Level:            level,
			ServiceName:      commonService,
			Environment:      commonEnvironment,
			ReadinessTimeout: f.config.Loki.ReadinessTimeout,
			ExportTimeout:    f.config.Loki.ExportTimeout,
			ShutdownTimeout:  f.config.Loki.ShutdownTimeout,
		})
	}
	panic("validated logger driver was not handled")
}

func parseLevel(value string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.TrimSpace(value))); err != nil {
		return 0, fmt.Errorf("parse logger level %q: %w", value, err)
	}
	return level, nil
}

func parseLocation(value string) (*time.Location, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "local") {
		return time.Local, nil
	}
	location, err := time.LoadLocation(value)
	if err != nil {
		return nil, fmt.Errorf("load logger timezone %q: %w", value, err)
	}
	return location, nil
}

var _ logging.Factory = Factory{}
