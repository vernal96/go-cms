package mysqlwidgetinstance

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestNewRepositoryRejectsNilDB(t *testing.T) {
	repository, err := NewRepository(nil)
	if err == nil || err.Error() != "mysql widget instance repository db is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository != nil {
		t.Fatal("repository must be nil when db is nil")
	}
}

func TestScanWidgetInstanceParams(t *testing.T) {
	tests := []struct {
		name   string
		params sql.NullString
		want   core.WidgetParams
	}{
		{
			name: "null column",
			want: core.WidgetParams{},
		},
		{
			name: "empty value",
			params: sql.NullString{
				Valid: true,
			},
			want: core.WidgetParams{},
		},
		{
			name: "json null",
			params: sql.NullString{
				String: "null",
				Valid:  true,
			},
			want: core.WidgetParams{},
		},
		{
			name: "object",
			params: sql.NullString{
				String: `{"text":"Hello"}`,
				Valid:  true,
			},
			want: core.WidgetParams{
				"text": "Hello",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			instance, err := scanWidgetInstance(
				testRow{
					values: []any{
						int64(11),
						core.ResourceID(7),
						core.WidgetCode("core.text"),
						core.WidgetTemplateDefault,
						core.WidgetArea("main"),
						test.params,
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
			if !reflect.DeepEqual(instance.Params, test.want) {
				t.Fatalf("unexpected params: %#v", instance.Params)
			}
		})
	}
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
				sql.NullString{
					String: `{"text":"Hello"}`,
					Valid:  true,
				},
				3,
			},
		},
		core.WidgetInstanceSourceTemplate,
	)
	if err != nil {
		t.Fatal(err)
	}

	if instance.ResourceTemplate != "default" {
		t.Fatalf("unexpected resource template: %q", instance.ResourceTemplate)
	}
	if instance.Source != core.WidgetInstanceSourceTemplate {
		t.Fatalf("unexpected source: %q", instance.Source)
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
		case *sql.NullString:
			*destination = value.(sql.NullString)
		default:
			return errors.New("unexpected scan destination")
		}
	}

	return nil
}
