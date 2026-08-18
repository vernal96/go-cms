package template

import (
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
)

type testResolver map[field.TypeCode]field.Type

func (r testResolver) FieldType(code field.TypeCode) (field.Type, bool) {
	value, exists := r[code]
	return value, exists
}

func resolver() testResolver {
	result := testResolver{}
	for _, value := range field.StandardTypes() {
		result[value.Code()] = value
	}
	return result
}

func TestCompileRejectsDuplicateResourceSlot(t *testing.T) {
	_, err := Compile([]Definition{{
		Code: "page", Label: "Page",
		Layout: Layout{Body: []LayoutItem{
			{Kind: ItemResourceSlot}, {Kind: ItemResourceSlot},
		}},
	}}, resolver())
	if err == nil || !strings.Contains(err.Error(), "duplicate resource widget slots") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileRejectsPresentationOnResourceSlot(t *testing.T) {
	_, err := Compile([]Definition{{
		Code: "page", Label: "Page",
		Layout: Layout{Body: []LayoutItem{{
			Kind: ItemResourceSlot, Presentation: widget.DefaultPresentation(),
		}}},
	}}, resolver())
	if err == nil || !strings.Contains(err.Error(), "resource slot") {
		t.Fatalf("error = %v", err)
	}
}

func TestComposeInsertsBodyAndSidebarBindingsAtIndependentSlots(t *testing.T) {
	presentation := widget.DefaultPresentation()
	catalog, err := Compile([]Definition{{
		Code: "page", Label: "Page",
		Layout: Layout{
			Body: []LayoutItem{
				{Kind: ItemWidget, Key: "before", Widget: "core_before", Presentation: presentation},
				{Kind: ItemResourceSlot},
				{Kind: ItemWidget, Key: "after", Widget: "core_after", Presentation: presentation},
			},
			Sidebar: []LayoutItem{
				{Kind: ItemWidget, Key: "navigation", Widget: "core_navigation", Presentation: presentation},
				{Kind: ItemResourceSlot},
			},
		},
	}}, resolver())
	if err != nil {
		t.Fatal(err)
	}
	runtime, _ := catalog.Template("page")
	placements, err := Compose(runtime, []widget.Binding{
		{ID: 22, Code: "core_quote", Area: widget.AreaBody, Position: 0, Presentation: presentation},
		{ID: 41, Code: "core_contact", Area: widget.AreaSidebar, Position: 0, Presentation: presentation},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := placementCodes(placements.Body); got != "core_before,core_quote,core_after" {
		t.Fatalf("body = %s", got)
	}
	if got := placementCodes(placements.Sidebar); got != "core_navigation,core_contact" {
		t.Fatalf("sidebar = %s", got)
	}
	if placements.Body[1].Key != "resource-widget-22" || placements.Sidebar[1].Key != "resource-widget-41" {
		t.Fatalf("keys = %#v / %#v", placements.Body, placements.Sidebar)
	}
}

func placementCodes(source []widget.Placement) string {
	values := make([]string, len(source))
	for index, placement := range source {
		values[index] = string(placement.Code)
	}
	return strings.Join(values, ",")
}
