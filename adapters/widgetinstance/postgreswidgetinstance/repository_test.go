package postgreswidgetinstance

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestNewRepositoryRejectsNilPool(t *testing.T) {
	repository, err := NewRepository(nil)
	if err == nil {
		t.Fatal("expected nil pool error")
	}
	if repository != nil {
		t.Fatal("repository must be nil when pool is nil")
	}
}

func TestFindForResourceValidatesResource(t *testing.T) {
	repository := &Repository{}

	t.Run("resource id", func(t *testing.T) {
		_, err := repository.FindForResource(context.Background(), core.Resource{
			Template: "default",
		})
		if err == nil || err.Error() != "widget instance resource id must be positive" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resource template", func(t *testing.T) {
		_, err := repository.FindForResource(context.Background(), core.Resource{
			ID: 1,
		})
		if err == nil || err.Error() != "widget instance resource template is empty" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestScanTemplateWidgetInstance(t *testing.T) {
	instance, err := scanWidgetInstance(
		testRow{
			values: []any{
				int64(10),
				core.ResourceTemplateCode("default"),
				core.WidgetCode("core.text"),
				core.WidgetTemplateDefault,
				core.WidgetArea("main"),
				[]byte(`{"text":"Hello"}`),
				3,
			},
		},
		core.WidgetInstanceSourceTemplate,
	)
	if err != nil {
		t.Fatal(err)
	}

	expected := core.WidgetInstance{
		ID:               10,
		Source:           core.WidgetInstanceSourceTemplate,
		ResourceTemplate: "default",
		Widget:           "core.text",
		Template:         core.WidgetTemplateDefault,
		Area:             "main",
		Params: core.WidgetParams{
			"text": "Hello",
		},
		Sort: 3,
	}
	if !reflect.DeepEqual(instance, expected) {
		t.Fatalf("unexpected instance: %#v", instance)
	}
}

func TestScanResourceWidgetInstanceNullParams(t *testing.T) {
	instance, err := scanWidgetInstance(
		testRow{
			values: []any{
				int64(11),
				core.ResourceID(7),
				core.WidgetCode("core.text"),
				core.WidgetTemplateDefault,
				core.WidgetArea("main"),
				[]byte(" \n null \t"),
				4,
			},
		},
		core.WidgetInstanceSourceResource,
	)
	if err != nil {
		t.Fatal(err)
	}

	if instance.Source != core.WidgetInstanceSourceResource {
		t.Fatalf("unexpected source: %q", instance.Source)
	}
	if instance.ResourceID != 7 {
		t.Fatalf("unexpected resource id: %d", instance.ResourceID)
	}
	if instance.Params == nil || len(instance.Params) != 0 {
		t.Fatalf("expected empty params, got %#v", instance.Params)
	}
}

type testRow struct {
	values []any
	err    error
}

func (r testRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("unexpected scan destination count")
	}

	for index, value := range r.values {
		switch destination := dest[index].(type) {
		case *int64:
			*destination = value.(int64)
		case *int:
			*destination = value.(int)
		case *core.ResourceID:
			*destination = value.(core.ResourceID)
		case *core.ResourceTemplateCode:
			*destination = value.(core.ResourceTemplateCode)
		case *core.WidgetCode:
			*destination = value.(core.WidgetCode)
		case *core.WidgetTemplateCode:
			*destination = value.(core.WidgetTemplateCode)
		case *core.WidgetArea:
			*destination = value.(core.WidgetArea)
		case *[]byte:
			*destination = append((*destination)[:0], value.([]byte)...)
		default:
			return errors.New("unexpected scan destination")
		}
	}

	return nil
}
