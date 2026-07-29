package dev_test

import (
	"testing"

	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
)

func TestProfileContainsRequiredModulesInOrder(t *testing.T) {
	if len(dev.Profile.Modules) != 2 {
		t.Fatalf("profile module count = %d", len(dev.Profile.Modules))
	}
	if dev.Profile.Modules[0].Module.Code() != core.ModuleCode {
		t.Fatalf(
			"first profile module = %q",
			dev.Profile.Modules[0].Module.Code(),
		)
	}
	if dev.Profile.Modules[1].Module.Code() != admin.ModuleCode {
		t.Fatalf(
			"second profile module = %q",
			dev.Profile.Modules[1].Module.Code(),
		)
	}
}
