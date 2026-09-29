package config

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vernal96/go-cms/internal/infrastructure"
)

func cleanEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_DB", "POSTGRES_USER",
		"POSTGRES_PASSWORD", "POSTGRES_SSL_MODE", "KAFKA_BROKERS", "REDIS_ADDR", "REDIS_PASSWORD",
		"FILES_PUBLIC_ROOT", "FILES_PUBLIC_BASE_URL", "FILES_PRIVATE_ROOT", "FILES_PRIVATE_BASE_URL",
		"FILES_PRIVATE_SIGNING_KEY", "LOGGER_FILE_PATH", "SERVER_HOST", "SERVER_PORT",
		"JWT_SIGNING_KEY", "JWT_ISSUER", "JWT_AUDIENCE", "JWT_ACCESS_TTL", "JWT_CLOCK_SKEW",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("FILES_PRIVATE_SIGNING_KEY", strings.Repeat("f", 32))
	t.Setenv("JWT_SIGNING_KEY", strings.Repeat("j", 32))
}

func TestDefaults(t *testing.T) {
	cleanEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		Infrastructure: infrastructure.Config{
			KafkaBrokers: []string{"localhost:9092"},
			PostgresHost: "localhost", PostgresPort: 5432, PostgresDatabase: "cms",
			PostgresUser: "cms", PostgresSSLMode: "disable",
			PublicFilesRoot: "var/files/public", PublicFilesBaseURL: "http://localhost:8080",
			PrivateFilesRoot: "var/files/private", PrivateFilesBaseURL: "http://localhost:8080",
			PrivateFilesSignKey: strings.Repeat("f", 32), RedisAddress: "localhost:6379",
		},
		LoggerPath: "var/log/cms.log",
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatal("common defaults differ")
	}
	http, err := LoadHTTP()
	if err != nil {
		t.Fatal(err)
	}
	if http.Server != (ServerConfig{
		Host: "0.0.0.0", Port: 8080, ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second, ShutdownTimeout: 5 * time.Second,
	}) || http.JWT.Issuer != "go-cms" || http.JWT.Audience != "go-cms-api" ||
		http.JWT.AccessTTL != 30*time.Minute || http.JWT.ClockSkew != 30*time.Second {
		t.Fatal("HTTP defaults differ")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	cleanEnv(t)
	for key, value := range map[string]string{
		"POSTGRES_HOST": " db.example ", "POSTGRES_PORT": "15432",
		"POSTGRES_DB": "project", "POSTGRES_USER": "owner", "POSTGRES_PASSWORD": " password ",
		"POSTGRES_SSL_MODE": "require", "KAFKA_BROKERS": "broker1:9092,broker2:9092",
		"REDIS_ADDR": "cache:6379", "REDIS_PASSWORD": " cache-secret ",
		"FILES_PUBLIC_ROOT": "/public", "FILES_PUBLIC_BASE_URL": "https://public.example",
		"FILES_PRIVATE_ROOT": "/private", "FILES_PRIVATE_BASE_URL": "https://private.example",
		"FILES_PRIVATE_SIGNING_KEY": " " + strings.Repeat("f", 32) + " ",
		"LOGGER_FILE_PATH":          "/logs/project.log", "SERVER_HOST": "::1", "SERVER_PORT": "18080",
		"JWT_ISSUER": "project", "JWT_AUDIENCE": "project-api", "JWT_ACCESS_TTL": "1h",
		"JWT_CLOCK_SKEW": "0s", "JWT_SIGNING_KEY": " " + strings.Repeat("j", 32) + " ",
	} {
		t.Setenv(key, value)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		Infrastructure: infrastructure.Config{
			KafkaBrokers: []string{"broker1:9092", "broker2:9092"},
			PostgresHost: "db.example", PostgresPort: 15432, PostgresDatabase: "project",
			PostgresUser: "owner", PostgresPassword: " password ", PostgresSSLMode: "require",
			PublicFilesRoot: "/public", PublicFilesBaseURL: "https://public.example",
			PrivateFilesRoot: "/private", PrivateFilesBaseURL: "https://private.example",
			PrivateFilesSignKey: " " + strings.Repeat("f", 32) + " ",
			RedisAddress:        "cache:6379", RedisPassword: " cache-secret ",
		},
		LoggerPath: "/logs/project.log",
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatal("common overrides or secret whitespace were not preserved")
	}
	http, err := LoadHTTP()
	if err != nil {
		t.Fatal(err)
	}
	if http.Server.Host != "::1" || http.Server.Port != 18080 || http.JWT.Issuer != "project" ||
		http.JWT.Audience != "project-api" || http.JWT.AccessTTL != time.Hour ||
		http.JWT.ClockSkew != 0 || http.JWT.SigningKey != " "+strings.Repeat("j", 32)+" " {
		t.Fatal("HTTP overrides or secret whitespace were not preserved")
	}
}

func TestInvalidEnvironment(t *testing.T) {
	for _, test := range []struct {
		key, value string
		http       bool
	}{
		{"POSTGRES_PORT", "not-a-number", false},
		{"POSTGRES_PORT", "0", false},
		{"POSTGRES_PORT", "-1", false},
		{"POSTGRES_PORT", "65536", false},
		{"FILES_PRIVATE_SIGNING_KEY", "", false},
		{"FILES_PRIVATE_SIGNING_KEY", "short-secret", false},
		{"SERVER_PORT", "not-a-number", true},
		{"SERVER_PORT", "0", true},
		{"SERVER_PORT", "65536", true},
		{"JWT_ACCESS_TTL", "not-a-duration", true},
		{"JWT_ACCESS_TTL", "0s", true},
		{"JWT_ACCESS_TTL", "-1s", true},
		{"JWT_ACCESS_TTL", "30s", true},
		{"JWT_CLOCK_SKEW", "not-a-duration", true},
		{"JWT_CLOCK_SKEW", "-1s", true},
		{"JWT_CLOCK_SKEW", "6m", true},
		{"JWT_SIGNING_KEY", "", true},
		{"JWT_SIGNING_KEY", "short-secret", true},
	} {
		t.Run(test.key+"/"+test.value, func(t *testing.T) {
			cleanEnv(t)
			t.Setenv(test.key, test.value)
			var err error
			if test.http {
				_, err = LoadHTTP()
			} else {
				_, err = Load()
			}
			if err == nil || !strings.Contains(err.Error(), test.key) {
				t.Fatalf("expected an error naming %s, got %v", test.key, err)
			}
			if strings.HasSuffix(test.key, "SIGNING_KEY") && test.value != "" && strings.Contains(err.Error(), test.value) {
				t.Fatal("error exposes a secret")
			}
		})
	}
}

func TestCommonConfigIgnoresHTTPSettings(t *testing.T) {
	cleanEnv(t)
	for _, key := range []string{"SERVER_PORT", "JWT_ACCESS_TTL", "JWT_CLOCK_SKEW"} {
		t.Setenv(key, "invalid")
	}
	t.Setenv("JWT_SIGNING_KEY", "")
	if _, err := Load(); err != nil {
		t.Fatalf("console config must not validate HTTP settings: %v", err)
	}
}

func TestWhitespaceUsesDefaults(t *testing.T) {
	cleanEnv(t)
	for _, key := range []string{"POSTGRES_PORT", "SERVER_PORT", "JWT_ACCESS_TTL", "JWT_CLOCK_SKEW"} {
		t.Setenv(key, " \t ")
	}
	cfg, err := Load()
	if err != nil || cfg.Infrastructure.PostgresPort != 5432 {
		t.Fatalf("common whitespace defaults: %v", err)
	}
	http, err := LoadHTTP()
	if err != nil || http.Server.Port != 8080 || http.JWT.AccessTTL != 30*time.Minute || http.JWT.ClockSkew != 30*time.Second {
		t.Fatalf("HTTP whitespace defaults: %v", err)
	}
}
