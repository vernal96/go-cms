package registry

import (
	"context"
	"errors"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/vernal96/go-cms/adapters/cache/rediscache"
	"github.com/vernal96/go-cms/adapters/eventbus/kafkaeventbus"
	"github.com/vernal96/go-cms/adapters/logger/stdoutlogger"
	"github.com/vernal96/go-cms/adapters/storage/localstorage"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/testmodule"
	"github.com/vernal96/go-cms/internal/testsite"
)

type DevInfrastructureConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	StorageRoot   string
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaGroupID  string
}

type DevInfrastructure struct {
	redisClient *redis.Client
	kafkaBus    *kafkaeventbus.Bus
}

func RegisterDevInfrastructure(
	ctx context.Context,
	r *project.InfrastructureRegistry,
	config DevInfrastructureConfig,
	resources core.ResourceRepository,
) (*DevInfrastructure, error) {
	if r == nil {
		return nil, errors.New("infrastructure registry is nil")
	}
	if resources == nil {
		return nil, errors.New("resource repository is nil")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	cacheStore, err := rediscache.NewStore(redisClient)
	if err != nil {
		_ = redisClient.Close()
		return nil, err
	}
	if err := cacheStore.Ping(ctx); err != nil {
		_ = redisClient.Close()
		return nil, err
	}

	localStorage, err := localstorage.NewStorage(config.StorageRoot)
	if err != nil {
		_ = redisClient.Close()
		return nil, err
	}

	kafkaBus, err := kafkaeventbus.NewBus(kafkaeventbus.Config{
		Brokers: config.KafkaBrokers,
		Topic:   config.KafkaTopic,
		GroupID: config.KafkaGroupID,
	})
	if err != nil {
		_ = redisClient.Close()
		return nil, err
	}

	r.RegisterCacheStore(rediscache.StoreName, cacheStore)
	r.RegisterCacheScope(core.CacheScopeDefault, cacheStore)
	r.RegisterCacheScope(testmodule.CacheScopeDefault, cacheStore)
	r.RegisterFileDisk(testmodule.FileDiskDefault, localStorage)
	r.UseEventBus(kafkaBus)
	r.UseLogger(stdoutlogger.New(slog.Default()))
	r.UseResourceRepository(resources)

	return &DevInfrastructure{
		redisClient: redisClient,
		kafkaBus:    kafkaBus,
	}, nil
}

func (i *DevInfrastructure) Close() error {
	return errors.Join(
		i.kafkaBus.Close(),
		i.redisClient.Close(),
	)
}

func RegisterDevSiteProfiles(r *project.SiteProfileRegistry) {
	r.Register(testsite.New())
}
