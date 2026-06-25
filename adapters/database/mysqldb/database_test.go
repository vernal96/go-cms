package mysqldb

import (
	"context"
	"testing"
)

func TestConnectRejectsEmptyDSN(t *testing.T) {
	database, err := Connect(context.Background(), "")
	if err == nil || err.Error() != "mysql DSN is empty" {
		t.Fatalf("unexpected error: %v", err)
	}
	if database != nil {
		t.Fatal("database must be nil when DSN is empty")
	}
}

func TestMigrateRejectsNilDB(t *testing.T) {
	err := Migrate(context.Background(), nil)
	if err == nil || err.Error() != "mysql migration db is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}
