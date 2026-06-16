package core

import (
	"context"
	"time"
)

const (
	CacheStoreDefault = "default"
	CacheStoreRedis   = "redis"
	CacheStoreFile    = "file"
	CacheStoreMemory  = "memory"
	CacheStoreNull    = "null"
)

type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type CacheManager interface {
	Default() CacheStore
	Redis() CacheStore
	File() CacheStore
	Memory() CacheStore
	Null() CacheStore
	Store(name string) (CacheStore, bool)
}
