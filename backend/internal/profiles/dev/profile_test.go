package dev_test

import (
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/internal/profiles/dev/widgetviews"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	corewidgets "github.com/vernal96/go-cms/kernel/modules/core/widgets"
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
	wantProfileTabs := []field.EditorTab{
		{Code: "main", Label: "Основные", Fields: []string{"string_value", "integer_value", "float_value", "checkbox_value"}},
		{Code: "selection", Label: "Выбор", Fields: []string{"radio_value", "select_value", "multi_select_value"}},
		{Code: "text", Label: "Текст", Fields: []string{"textarea_value"}},
		{Code: "contacts", Label: "Контакты", Fields: []string{"email_value", "phone_value"}},
	}
	if !reflect.DeepEqual(dev.Profile.EditorTabs, wantProfileTabs) {
		t.Fatalf("profile editor tabs = %#v", dev.Profile.EditorTabs)
	}
	if len(dev.Profile.Templates) != 2 ||
		dev.Profile.Templates[0].Code != "page" || len(dev.Profile.Templates[0].Fields) != 4 ||
		dev.Profile.Templates[1].Code != "landing" || len(dev.Profile.Templates[1].Fields) != 5 {
		t.Fatalf("templates = %#v", dev.Profile.Templates)
	}
	page := dev.Profile.Templates[0]
	wantPageTabs := []field.EditorTab{
		{Code: "content", Label: "Контент", Fields: []string{"page_title", "page_text", "show_title"}},
		{Code: "layout", Label: "Макет", Fields: []string{"layout"}},
	}
	if !reflect.DeepEqual(page.EditorTabs, wantPageTabs) {
		t.Fatalf("page editor tabs = %#v", page.EditorTabs)
	}
	wantLandingTabs := []field.EditorTab{
		{Code: "content", Label: "Первый экран", Fields: []string{"hero_title", "hero_text"}},
		{Code: "layout", Label: "Макет", Fields: []string{"columns", "content_width"}},
		{Code: "audience", Label: "Аудитория", Fields: []string{"audiences"}},
	}
	if !reflect.DeepEqual(dev.Profile.Templates[1].EditorTabs, wantLandingTabs) {
		t.Fatalf("landing editor tabs = %#v", dev.Profile.Templates[1].EditorTabs)
	}
	if len(page.Layout.Body) != 2 || len(page.Layout.Sidebar) != 1 {
		t.Fatalf("page widget layout = %#v", page.Layout)
	}
	content, contentOK := page.Layout.Body[0].(template.Widget)
	_, bodySlotOK := page.Layout.Body[1].(template.ResourceWidgets)
	_, sidebarSlotOK := page.Layout.Sidebar[0].(template.ResourceWidgets)
	if !contentOK || content.Widget != corewidgets.Content || !content.View.IsZero() ||
		!bodySlotOK || !sidebarSlotOK {
		t.Fatalf("page widget layout = %#v", page.Layout)
	}
	if len(dev.Profile.WidgetViews) != 2 ||
		dev.Profile.WidgetViews[0] != widgetviews.ContentCompact ||
		dev.Profile.WidgetViews[1] != widgetviews.ContentArticle {
		t.Fatalf("widget views = %#v", dev.Profile.WidgetViews)
	}
}
