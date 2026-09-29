// Package config reads deployment values without assembling the application.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vernal96/go-cms-kernel/security/jwt"
	"github.com/vernal96/go-cms/internal/infrastructure"
)

type Config struct {
	Infrastructure infrastructure.Config
	LoggerPath     string
}

type HTTPConfig struct {
	Server ServerConfig
	JWT    jwt.Config
}

type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Load reads the settings shared by HTTP and console modes. HTTP settings are
// deliberately loaded separately so console commands do not require JWT.
func Load() (Config, error) {
	port, err := envPort("POSTGRES_PORT", 5432)
	if err != nil {
		return Config{}, err
	}
	signingKey := os.Getenv("FILES_PRIVATE_SIGNING_KEY")
	if len(signingKey) < 32 {
		return Config{}, fmt.Errorf("FILES_PRIVATE_SIGNING_KEY must contain at least 32 bytes")
	}
	return Config{
		Infrastructure: infrastructure.Config{
			KafkaBrokers:        strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","),
			PostgresHost:        env("POSTGRES_HOST", "localhost"),
			PostgresPort:        port,
			PostgresDatabase:    env("POSTGRES_DB", "cms"),
			PostgresUser:        env("POSTGRES_USER", "cms"),
			PostgresPassword:    os.Getenv("POSTGRES_PASSWORD"),
			PostgresSSLMode:     env("POSTGRES_SSL_MODE", "disable"),
			PublicFilesRoot:     env("FILES_PUBLIC_ROOT", "var/files/public"),
			PublicFilesBaseURL:  env("FILES_PUBLIC_BASE_URL", "http://localhost:8080"),
			PrivateFilesRoot:    env("FILES_PRIVATE_ROOT", "var/files/private"),
			PrivateFilesBaseURL: env("FILES_PRIVATE_BASE_URL", "http://localhost:8080"),
			PrivateFilesSignKey: signingKey,
			RedisAddress:        env("REDIS_ADDR", "localhost:6379"),
			RedisPassword:       os.Getenv("REDIS_PASSWORD"),
		},
		LoggerPath: env("LOGGER_FILE_PATH", "var/log/cms.log"),
	}, nil
}

// LoadHTTP reads and validates values used only when serving HTTP.
func LoadHTTP() (HTTPConfig, error) {
	port, err := envPort("SERVER_PORT", 8080)
	if err != nil {
		return HTTPConfig{}, err
	}
	accessTTL, err := envDuration("JWT_ACCESS_TTL", 30*time.Minute)
	if err != nil {
		return HTTPConfig{}, err
	}
	clockSkew, err := envDuration("JWT_CLOCK_SKEW", 30*time.Second)
	if err != nil {
		return HTTPConfig{}, err
	}
	signingKey := os.Getenv("JWT_SIGNING_KEY")
	if len(signingKey) < 32 {
		return HTTPConfig{}, fmt.Errorf("JWT_SIGNING_KEY must contain at least 32 bytes")
	}
	if accessTTL <= 0 {
		return HTTPConfig{}, fmt.Errorf("JWT_ACCESS_TTL must be positive")
	}
	if clockSkew < 0 || clockSkew > 5*time.Minute {
		return HTTPConfig{}, fmt.Errorf("JWT_CLOCK_SKEW must be between zero and 5m")
	}
	if clockSkew >= accessTTL {
		return HTTPConfig{}, fmt.Errorf("JWT_CLOCK_SKEW must be shorter than JWT_ACCESS_TTL")
	}
	return HTTPConfig{
		Server: ServerConfig{
			Host: env("SERVER_HOST", "0.0.0.0"), Port: port,
			ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
		JWT: jwt.Config{
			SigningKey: signingKey, Issuer: env("JWT_ISSUER", "go-cms"),
			Audience: env("JWT_AUDIENCE", "go-cms-api"), AccessTTL: accessTTL,
			ClockSkew: clockSkew,
		},
	}, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envPort(key string, fallback int) (int, error) {
	value, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if value < 1 || value > 65535 {
		return 0, fmt.Errorf("%s must be between 1 and 65535", key)
	}
	return value, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	value, err := time.ParseDuration(env(key, fallback.String()))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return value, nil
}
