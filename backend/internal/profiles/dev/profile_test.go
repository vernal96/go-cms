package dev_test

import (
	"testing"

	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/modules/seo"
)

func TestProfileContainsRequiredModulesInOrder(t *testing.T) {
	if len(dev.Profile.Modules) != 3 {
		t.Fatalf("profile module count = %d", len(dev.Profile.Modules))
	}
	if dev.Profile.Modules[0].Module.Code() != core.ModuleCode {
		t.Fatalf(
			"first profile module = %q",
			dev.Profile.Modules[0].Module.Code(),
		)
	}
	if dev.Profile.Modules[1].Module.Code() != seo.ModuleCode {
		t.Fatalf(
			"second profile module = %q",
			dev.Profile.Modules[1].Module.Code(),
		)
	}
	if dev.Profile.Modules[2].Module.Code() != admin.ModuleCode {
		t.Fatalf(
			"third profile module = %q",
			dev.Profile.Modules[2].Module.Code(),
		)
	}
}

func TestProfileExposesDynamicParamsAndTemplateFields(t *testing.T) {
	if len(dev.Profile.Params) != 10 {
		t.Fatalf("profile params = %d", len(dev.Profile.Params))
	}
	wantTypes := map[field.TypeCode]bool{
		field.TypeString: false, field.TypeInteger: false, field.TypeFloat: false,
		field.TypeCheckbox: false, field.TypeRadio: false, field.TypeSelect: false,
		field.TypeTextarea: false, field.TypeEmail: false, field.TypePhone: false,
	}
	for _, definition := range dev.Profile.Params {
		wantTypes[definition.Type] = true
	}
	for code, found := range wantTypes {
		if !found {
			t.Fatalf("field type %q is missing", code)
		}
	}
	if len(dev.Profile.Templates) != 2 ||
		dev.Profile.Templates[0].Code != "page" || len(dev.Profile.Templates[0].Fields) != 4 ||
		dev.Profile.Templates[1].Code != "landing" || len(dev.Profile.Templates[1].Fields) != 5 {
		t.Fatalf("templates = %#v", dev.Profile.Templates)
	}
	page := dev.Profile.Templates[0]
	if len(page.Layout.Body) != 2 || page.Layout.Body[0].Kind != template.ItemWidget ||
		page.Layout.Body[0].Widget != "core_content" || page.Layout.Body[1].Kind != template.ItemResourceSlot ||
		len(page.Layout.Sidebar) != 1 || page.Layout.Sidebar[0].Kind != template.ItemResourceSlot {
		t.Fatalf("page widget layout = %#v", page.Layout)
	}
	if len(dev.Profile.WidgetViews) != 2 || dev.Profile.WidgetViews[0].Widget != widget.Code("core_content") {
		t.Fatalf("widget views = %#v", dev.Profile.WidgetViews)
	}
}
