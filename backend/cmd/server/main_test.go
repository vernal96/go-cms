package main

import (
	"context"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/internal/bootstrap"
)

func TestExecutableMode(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		mode bootstrap.Mode
	}{
		{"http", nil, bootstrap.HTTP},
		{"console help", []string{"console"}, bootstrap.Console},
		{"user creation", []string{"console", "users", "create", "--group", "admin"}, bootstrap.Console},
		{"schema rollback", []string{"console", "migrations", "down"}, bootstrap.Console},
	} {
		t.Run(test.name, func(t *testing.T) {
			mode, err := executableMode(test.args)
			if err != nil || mode != test.mode {
				t.Fatalf("mode=%v, err=%v", mode, err)
			}
		})
	}
}

func TestUnknownModeFailsBeforeConfigAndInfrastructure(t *testing.T) {
	t.Setenv("POSTGRES_PORT", "invalid")
	for _, args := range [][]string{{"bootstrap-admin"}, {"unknown"}, {"users", "create"}} {
		err := run(context.Background(), args)
		if err == nil || !strings.Contains(err.Error(), "usage: server") {
			t.Fatalf("expected usage before configuration, got %v", err)
		}
	}
}
