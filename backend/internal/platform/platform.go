package platform

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/vernal96/go-cms-kernel/connectors/filelogger"
	"github.com/vernal96/go-cms-kernel/logging"
)

type LoggerFactory struct{ Path string }

func (f LoggerFactory) Open(ctx context.Context) (logging.Connector, error) {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o700); err != nil {
		return nil, err
	}
	return filelogger.New(ctx, filelogger.Config{Path: f.Path, Level: slog.LevelInfo, ServiceName: "cms-starter", Environment: "dev"})
}
