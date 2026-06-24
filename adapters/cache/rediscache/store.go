package rediscache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vernal96/go-cms/core"
)

const StoreName core.CacheStoreName = "redis"

type Store struct {
	client redis.UniversalClient
}

func NewStore(client redis.UniversalClient) (*Store, error) {
	if client == nil {
		return nil, errors.New("redis cache client is nil")
	}

	return &Store{
		client: client,
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *Store) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

var _ core.CacheStore = (*Store)(nil)
