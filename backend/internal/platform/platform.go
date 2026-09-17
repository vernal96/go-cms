package platform

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/vernal96/go-cms-kernel/connectors/filelogger"
	"github.com/vernal96/go-cms-kernel/eventbus"
	"github.com/vernal96/go-cms-kernel/logging"
)

type LoggerFactory struct{ Path string }

func (f LoggerFactory) Open(ctx context.Context) (logging.Connector, error) {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o700); err != nil {
		return nil, err
	}
	return filelogger.New(ctx, filelogger.Config{Path: f.Path, Level: slog.LevelInfo, ServiceName: "cms-starter", Environment: "dev"})
}

type DiscardBusFactory struct{}

func (DiscardBusFactory) Open(context.Context) (eventbus.Connector, error) { return &discardBus{}, nil }

type discardBus struct {
	mu     sync.RWMutex
	closed bool
}

func (b *discardBus) Publish(context.Context, eventbus.Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return eventbus.ErrClosed
	}
	return nil
}

func (b *discardBus) Consume(ctx context.Context, _ eventbus.Subscription, _ eventbus.Handler) error {
	<-ctx.Done()
	return ctx.Err()
}

func (b *discardBus) Ping(context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return eventbus.ErrClosed
	}
	return nil
}

func (b *discardBus) Close() error {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	return nil
}
