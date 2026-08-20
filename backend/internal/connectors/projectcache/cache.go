package projectcache

import (
	"time"

	"github.com/vernal96/go-cms/connectors/filesystemcache"
	redisconnector "github.com/vernal96/go-cms/connectors/redis"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/filesystem"
)

const (
	FilesystemCode cache.Code = "filesystem_cache"
	RedisCode      cache.Code = "redis_cache"
)

type Config struct {
	Filesystem FilesystemConfig `envconfig:"FILESYSTEM"`
	Redis      RedisConfig      `envconfig:"REDIS"`
}

type FilesystemConfig struct {
	Storage filesystem.Code        `envconfig:"STORAGE" default:"private"`
	Layout  filesystemcache.Layout `envconfig:"LAYOUT" default:"auto"`
	Prefix  string                 `envconfig:"PREFIX" default:"cache/filesystem_cache"`
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
	Prefix           string        `envconfig:"PREFIX" default:"cms:cache:redis_cache"`
}

// Factories declares both application-owned physical cache stores. Modules
// select neither technology; profiles bind module policy aliases to these
// project-level store codes.
func (c Config) Factories() []cache.Factory {
	return []cache.Factory{
		filesystemcache.Factory{Config: filesystemcache.Config{
			Code:    FilesystemCode,
			Disk:    c.Filesystem.Storage,
			Layout:  c.Filesystem.Layout,
			Prefix:  c.Filesystem.Prefix,
			MaxSize: c.Filesystem.MaxSize,
		}},
		redisconnector.Factory{Config: redisconnector.Config{
			Code:             RedisCode,
			Addrs:            c.Redis.Addrs,
			ClientName:       c.Redis.ClientName,
			Username:         c.Redis.Username,
			Password:         c.Redis.Password,
			DB:               c.Redis.DB,
			MasterName:       c.Redis.MasterName,
			SentinelUsername: c.Redis.SentinelUsername,
			SentinelPassword: c.Redis.SentinelPassword,
			Protocol:         c.Redis.Protocol,
			MaxRetries:       c.Redis.MaxRetries,
			DialTimeout:      c.Redis.DialTimeout,
			ReadTimeout:      c.Redis.ReadTimeout,
			WriteTimeout:     c.Redis.WriteTimeout,
			PoolTimeout:      c.Redis.PoolTimeout,
			PoolSize:         c.Redis.PoolSize,
			MinIdleConns:     c.Redis.MinIdleConns,
			MaxIdleConns:     c.Redis.MaxIdleConns,
			ConnMaxIdleTime:  c.Redis.ConnMaxIdleTime,
			ConnMaxLifetime:  c.Redis.ConnMaxLifetime,
			TLSEnabled:       c.Redis.TLSEnabled,
			TLSServerName:    c.Redis.TLSServerName,
			TLSInsecure:      c.Redis.TLSInsecure,
			Prefix:           c.Redis.Prefix,
		}},
	}
}
