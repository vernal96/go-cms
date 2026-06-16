package memoryeventbus

import (
	"context"
	"sync"

	"github.com/vernal96/go-cms/core"
)

type Bus struct {
	mu       sync.RWMutex
	handlers map[core.EventName][]core.EventHandler
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[core.EventName][]core.EventHandler),
	}
}

func (b *Bus) Publish(ctx context.Context, event core.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.mu.RLock()
	handlers := append([]core.EventHandler(nil), b.handlers[event.Name]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := handler(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bus) Subscribe(name core.EventName, handler core.EventHandler) error {
	if handler == nil {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[name] = append(b.handlers[name], handler)

	return nil
}

var _ core.EventBus = (*Bus)(nil)
