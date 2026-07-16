package memory

import (
	"context"
	"errors"
	"sync"
)

var ErrClosed = errors.New("memory connector is closed")

type Connector struct {
	mu          sync.RWMutex
	closed      bool
	collections map[string]map[string]any
}

func New() *Connector {
	return &Connector{
		collections: make(map[string]map[string]any),
	}
}

func (c *Connector) Ping(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ping context is nil")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClosed
	}

	return nil
}

func (c *Connector) Set(
	collection string,
	key string,
	value any,
) error {
	if collection == "" {
		return errors.New("memory collection is empty")
	}

	if key == "" {
		return errors.New("memory key is empty")
	}

	if value == nil {
		return errors.New("memory value is nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClosed
	}

	if _, exists := c.collections[collection]; !exists {
		c.collections[collection] = make(map[string]any)
	}

	c.collections[collection][key] = value

	return nil
}

func (c *Connector) Get(
	ctx context.Context,
	collection string,
	key string,
) (any, bool, error) {
	if ctx == nil {
		return nil, false, errors.New("get context is nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	if collection == "" {
		return nil, false, errors.New("memory collection is empty")
	}

	if key == "" {
		return nil, false, errors.New("memory key is empty")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, false, ErrClosed
	}

	collectionValues, exists := c.collections[collection]
	if !exists {
		return nil, false, nil
	}

	value, exists := collectionValues[key]
	if !exists {
		return nil, false, nil
	}

	return value, true, nil
}

func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true

	return nil
}
