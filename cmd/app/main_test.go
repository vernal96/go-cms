package main

import (
	"context"
	"testing"
)

func TestBuildDatabaseRepositoriesRejectsUnsupportedDriver(t *testing.T) {
	t.Setenv("GO_CMS_DATABASE_DRIVER", "sqlite")

	repositories, err := buildDatabaseRepositories(context.Background())
	if err == nil || err.Error() != `unsupported database driver "sqlite"` {
		t.Fatalf("unexpected error: %v", err)
	}
	if repositories.close != nil {
		t.Fatal("repositories must be empty for an unsupported driver")
	}
}
