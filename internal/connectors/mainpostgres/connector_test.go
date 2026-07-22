package mainpostgres_test

import (
	"testing"
	"time"

	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
)

func TestFactoryMapsInstanceConfigToReusableConnector(t *testing.T) {
	factory := mainpostgres.Factory(mainpostgres.Config{
		Host:            "postgres.example",
		Port:            5432,
		Database:        "cms",
		User:            "cms",
		Password:        "secret",
		SSLMode:         "require",
		MaxOpenConns:    20,
		MinConns:        2,
		ConnMaxLifetime: time.Hour,
		ConnectTimeout:  time.Second,
	})

	if factory.Code() != mainpostgres.ConnectionCode {
		t.Fatalf("factory code = %q", factory.Code())
	}
	if factory.Config.Host != "postgres.example" || factory.Config.Database != "cms" {
		t.Fatalf("factory config = %#v", factory.Config)
	}
	if factory.Config.MaxConns != 20 || factory.Config.MinConns != 2 {
		t.Fatalf("pool config = %#v", factory.Config)
	}
}
