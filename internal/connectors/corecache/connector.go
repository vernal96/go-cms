package corecache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vernal96/go-cms/connectors/filesystemcache"
	redisconnector "github.com/vernal96/go-cms/connectors/redis"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/filesystem"
)

const Code cache.Code = "core_cache"

type Config struct {
	Driver     string           `envconfig:"DRIVER" default:"filesystem"`
	Filesystem FilesystemConfig `envconfig:"FILESYSTEM"`
	Redis      RedisConfig      `envconfig:"REDIS"`
}

type FilesystemConfig struct {
	Storage filesystem.Code        `envconfig:"STORAGE" default:"private"`
	Layout  filesystemcache.Layout `envconfig:"LAYOUT" default:"auto"`
	Prefix  string                 `envconfig:"PREFIX" default:"cache/core_cache"`
	MaxSize int64                  `envconfig:"MAX_SIZE" default:"0"`
}

type RedisConfig struct {
	Addrs            []string      `envconfig:"ADDRS" default:"localhost:6379"`
	ClientName       string        `envconfig:"CLIENT_NAME"`
	Username         string        `envconfig:"USERNAME"`
	Password         string        `envconfig:"PASSWORD"`
	DB               int           `envconfig:"DB" default:"0"`
	MasterName       string        `envconfig:"MASTER_NAME"`
	SentinelUsername string        `envconfig:"SENTINEL_USERNAME"`
	SentinelPassword string        `envconfig:"SENTINEL_PASSWORD"`
	Protocol         int           `envconfig:"PROTOCOL" default:"3"`
	MaxRetries       int           `envconfig:"MAX_RETRIES" default:"3"`
	DialTimeout      time.Duration `envconfig:"DIAL_TIMEOUT" default:"5s"`
	ReadTimeout      time.Duration `envconfig:"READ_TIMEOUT" default:"3s"`
	WriteTimeout     time.Duration `envconfig:"WRITE_TIMEOUT" default:"3s"`
	PoolTimeout      time.Duration `envconfig:"POOL_TIMEOUT" default:"4s"`
	PoolSize         int           `envconfig:"POOL_SIZE" default:"0"`
	MinIdleConns     int           `envconfig:"MIN_IDLE_CONNS" default:"0"`
	MaxIdleConns     int           `envconfig:"MAX_IDLE_CONNS" default:"0"`
	ConnMaxIdleTime  time.Duration `envconfig:"CONN_MAX_IDLE_TIME" default:"30m"`
	ConnMaxLifetime  time.Duration `envconfig:"CONN_MAX_LIFETIME" default:"0"`
	TLSEnabled       bool          `envconfig:"TLS_ENABLED" default:"false"`
	TLSServerName    string        `envconfig:"TLS_SERVER_NAME"`
	TLSInsecure      bool          `envconfig:"TLS_INSECURE" default:"false"`
	Prefix           string        `envconfig:"PREFIX" default:"cms:cache:core_cache"`
}

type Factory struct {
	config Config
}

func NewFactory(config Config) Factory {
	return Factory{config: config}
}

func (Factory) Code() cache.Code {
	return Code
}

func (f Factory) Open(
	ctx context.Context,
	dependencies cache.Dependencies,
) (cache.Store, error) {
	switch strings.ToLower(strings.TrimSpace(f.config.Driver)) {
	case "filesystem", "file", "local":
		return filesystemcache.Factory{Config: filesystemcache.Config{
			Code:    Code,
			Disk:    f.config.Filesystem.Storage,
			Layout:  f.config.Filesystem.Layout,
			Prefix:  f.config.Filesystem.Prefix,
			MaxSize: f.config.Filesystem.MaxSize,
		}}.Open(ctx, dependencies)
	case "redis":
		return redisconnector.Factory{Config: redisconnector.Config{
			Code:             Code,
			Addrs:            f.config.Redis.Addrs,
			ClientName:       f.config.Redis.ClientName,
			Username:         f.config.Redis.Username,
			Password:         f.config.Redis.Password,
			DB:               f.config.Redis.DB,
			MasterName:       f.config.Redis.MasterName,
			SentinelUsername: f.config.Redis.SentinelUsername,
			SentinelPassword: f.config.Redis.SentinelPassword,
			Protocol:         f.config.Redis.Protocol,
			MaxRetries:       f.config.Redis.MaxRetries,
			DialTimeout:      f.config.Redis.DialTimeout,
			ReadTimeout:      f.config.Redis.ReadTimeout,
			WriteTimeout:     f.config.Redis.WriteTimeout,
			PoolTimeout:      f.config.Redis.PoolTimeout,
			PoolSize:         f.config.Redis.PoolSize,
			MinIdleConns:     f.config.Redis.MinIdleConns,
			MaxIdleConns:     f.config.Redis.MaxIdleConns,
			ConnMaxIdleTime:  f.config.Redis.ConnMaxIdleTime,
			ConnMaxLifetime:  f.config.Redis.ConnMaxLifetime,
			TLSEnabled:       f.config.Redis.TLSEnabled,
			TLSServerName:    f.config.Redis.TLSServerName,
			TLSInsecure:      f.config.Redis.TLSInsecure,
			Prefix:           f.config.Redis.Prefix,
		}}.Open(ctx, dependencies)
	default:
		return nil, fmt.Errorf(
			"unsupported driver %q for cache store %q",
			f.config.Driver,
			Code,
		)
	}
}

var _ cache.Factory = Factory{}
