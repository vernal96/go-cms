package core

import (
	"context"
	"time"
)

type CacheStoreName string

type CacheScope string

const CacheScopeDefault CacheScope = "default"

type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type CacheManager interface {
	Store(name CacheStoreName) (CacheStore, error)
	Scope(scope CacheScope) (CacheStore, error)
}
