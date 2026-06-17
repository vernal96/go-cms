package memorycache

import (
	"context"
	"sync"
	"time"

	"github.com/vernal96/go-cms/core"
)

const StoreName core.CacheStoreName = "memory"

type Store struct {
	mu    sync.RWMutex
	items map[string]item
}

type item struct {
	value     []byte
	expiresAt time.Time
	hasTTl    bool
}

func NewStore() *Store {
	return &Store{
		items: make(map[string]item),
	}
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.items[key]
	if !exists {
		return nil, false, nil
	}

	if item.hasTTl && time.Now().After(item.expiresAt) {
		delete(s.items, key)

		return nil, false, nil
	}

	value := make([]byte, len(item.value))
	copy(value, item.value)

	return value, true, nil
}

func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	copiedValue := make([]byte, len(value))
	copy(copiedValue, value)

	cacheItem := item{
		value: copiedValue,
	}

	if ttl > 0 {
		cacheItem.expiresAt = time.Now().Add(ttl)
		cacheItem.hasTTl = true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = cacheItem

	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, key)

	return nil
}
