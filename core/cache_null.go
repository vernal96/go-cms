package core

import (
	"context"
	"time"
)

type NullCacheStore struct{}

func (s NullCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return nil, false, nil
}

func (s NullCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (s NullCacheStore) Delete(ctx context.Context, key string) error {
	return nil
}

var _ CacheStore = NullCacheStore{}
