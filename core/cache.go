package core

import (
	"context"
	"errors"
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

type LayeredCacheStore struct {
	primary        CacheStore
	secondary      CacheStore
	primaryWarmTTL time.Duration
}

func NewLayeredCacheStore(primary CacheStore, secondary CacheStore, primaryWarmTTL time.Duration) (*LayeredCacheStore, error) {
	if primary == nil {
		return nil, errors.New("primary cache store is nil")
	}

	if secondary == nil {
		return nil, errors.New("secondary cache store is nil")
	}

	return &LayeredCacheStore{
		primary:        primary,
		secondary:      secondary,
		primaryWarmTTL: primaryWarmTTL,
	}, nil
}

func (s *LayeredCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, exists, err := s.primary.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}

	if exists {
		return value, true, nil
	}

	value, exists, err = s.secondary.Get(ctx, key)
	if err != nil || !exists {
		return value, exists, err
	}

	if s.primaryWarmTTL > 0 {
		_ = s.primary.Set(ctx, key, value, s.primaryWarmTTL)
	}

	return value, true, nil
}

func (s *LayeredCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	primaryErr := s.primary.Set(ctx, key, value, ttl)
	secondaryErr := s.secondary.Set(ctx, key, value, ttl)

	return errors.Join(primaryErr, secondaryErr)
}

func (s *LayeredCacheStore) Delete(ctx context.Context, key string) error {
	primaryErr := s.primary.Delete(ctx, key)
	secondaryErr := s.secondary.Delete(ctx, key)

	return errors.Join(primaryErr, secondaryErr)
}

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

var _ CacheStore = (*LayeredCacheStore)(nil)
var _ CacheStore = NullCacheStore{}
