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
	"github.com/vernal96/go-cms/kernel/modules/forms"
	"github.com/vernal96/go-cms/kernel/modules/mail"
	"github.com/vernal96/go-cms/kernel/modules/search"
	"github.com/vernal96/go-cms/kernel/modules/seo"
)

var profile = dev.Profile(mail.Config{}, forms.Config{}, "private")

func TestProfileContainsRequiredModulesInOrder(t *testing.T) {
	if len(profile.Modules) != 6 {
		t.Fatalf("profile module count = %d", len(profile.Modules))
	}
	if profile.Modules[0].Module.Code() != core.ModuleCode {
		t.Fatalf(
			"first profile module = %q",
			profile.Modules[0].Module.Code(),
		)
	}
	if profile.Modules[1].Module.Code() != seo.ModuleCode {
		t.Fatalf(
			"second profile module = %q",
			profile.Modules[1].Module.Code(),
		)
	}
	if profile.Modules[4].Module.Code() != search.ModuleCode {
		t.Fatalf("search module = %q", profile.Modules[4].Module.Code())
	}
	if profile.Modules[5].Module.Code() != admin.ModuleCode {
		t.Fatalf(
			"last profile module = %q",
			profile.Modules[5].Module.Code(),
		)
	}
}

func TestProfileExposesDynamicParamsAndTemplateFields(t *testing.T) {
	if len(profile.Params) != 10 {
		t.Fatalf("profile params = %d", len(profile.Params))
	}
	wantTypes := map[field.TypeCode]bool{
		field.TypeString: false, field.TypeInteger: false, field.TypeFloat: false,
		field.TypeCheckbox: false, field.TypeRadio: false, field.TypeSelect: false,
		field.TypeTextarea: false, field.TypeEmail: false, field.TypePhone: false,
	}
	for _, definition := range profile.Params {
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
	if !reflect.DeepEqual(profile.EditorTabs, wantProfileTabs) {
		t.Fatalf("profile editor tabs = %#v", profile.EditorTabs)
	}
	if len(profile.Templates) != 2 ||
		profile.Templates[0].Code != "page" || len(profile.Templates[0].Fields) != 8 ||
		profile.Templates[1].Code != "landing" || len(profile.Templates[1].Fields) != 6 {
		t.Fatalf("templates = %#v", profile.Templates)
	}
	page := profile.Templates[0]
	wantPageTabs := []field.EditorTab{
		{Code: "multiple", Label: "Множественные поля", Fields: []string{"gallery", "tags", "scores"}},
		{Code: "content", Label: "Контент", Fields: []string{"page_title", "page_text", "page_media", "show_title"}},
		{Code: "layout", Label: "Макет", Fields: []string{"layout"}},
	}
	if !reflect.DeepEqual(page.EditorTabs, wantPageTabs) {
		t.Fatalf("page editor tabs = %#v", page.EditorTabs)
	}
	wantLandingTabs := []field.EditorTab{
		{Code: "slides", Label: "Слайды", Fields: []string{"slides"}},
		{Code: "content", Label: "Первый экран", Fields: []string{"hero_title", "hero_text"}},
		{Code: "layout", Label: "Макет", Fields: []string{"columns", "content_width"}},
		{Code: "audience", Label: "Аудитория", Fields: []string{"audiences"}},
	}
	if !reflect.DeepEqual(profile.Templates[1].EditorTabs, wantLandingTabs) {
		t.Fatalf("landing editor tabs = %#v", profile.Templates[1].EditorTabs)
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
	if len(profile.WidgetViews) != 2 ||
		profile.WidgetViews[0] != widgetviews.ContentCompact ||
		profile.WidgetViews[1] != widgetviews.ContentArticle {
		t.Fatalf("widget views = %#v", profile.WidgetViews)
	}
}
