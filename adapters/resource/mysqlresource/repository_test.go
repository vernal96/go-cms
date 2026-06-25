package mysqlresource

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestNewRepositoryRejectsNilDB(t *testing.T) {
	repository, err := NewRepository(nil)
	if err == nil || err.Error() != "mysql resource repository db is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository != nil {
		t.Fatal("repository must be nil when db is nil")
	}
}

func TestScanResource(t *testing.T) {
	resource, err := scanResource(testRow{
		values: []any{
			core.ResourceID(10),
			int64(7),
			sql.NullInt64{Int64: 3, Valid: true},
			core.ResourceType("page"),
			"default",
			"Home",
			"home",
			"/",
			2,
			true,
			[]byte(`{"layout":"wide"}`),
			[]byte(`{"title":"Home"}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	parentID := core.ResourceID(3)
	expected := core.Resource{
		ID:          10,
		SiteID:      7,
		ParentID:    &parentID,
		Type:        "page",
		Template:    "default",
		Title:       "Home",
		Alias:       "home",
		Path:        "/",
		Sort:        2,
		IsPublished: true,
		Settings: map[string]any{
			"layout": "wide",
		},
		SEO: map[string]any{
			"title": "Home",
		},
	}
	if !reflect.DeepEqual(resource, expected) {
		t.Fatalf("unexpected resource: %#v", resource)
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
		case *core.ResourceID:
			*destination = value.(core.ResourceID)
		case *int64:
			*destination = value.(int64)
		case *sql.NullInt64:
			*destination = value.(sql.NullInt64)
		case *core.ResourceType:
			*destination = value.(core.ResourceType)
		case *string:
			*destination = value.(string)
		case *int:
			*destination = value.(int)
		case *bool:
			*destination = value.(bool)
		case *[]byte:
			*destination = append((*destination)[:0], value.([]byte)...)
		default:
			return errors.New("unexpected scan destination")
		}
	}

	return nil
}
