package mysqlsite

import (
	"errors"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestNewRepositoryRejectsNilDB(t *testing.T) {
	repository, err := NewRepository(nil)
	if err == nil || err.Error() != "mysql site repository db is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository != nil {
		t.Fatal("repository must be nil when db is nil")
	}
}

func TestScanSiteSettings(t *testing.T) {
	site, err := scanSite(testRow{
		values: []any{
			int64(7),
			"main",
			"example.com",
			"ru",
			[]byte(`{"name":"Example"}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := core.Site{
		ID:          7,
		ProfileCode: "main",
		Domain:      "example.com",
		Locale:      "ru",
		Settings: map[string]any{
			"name": "Example",
		},
	}
	if !reflect.DeepEqual(site, expected) {
		t.Fatalf("unexpected site: %#v", site)
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
		case *string:
			*destination = value.(string)
		case *[]byte:
			*destination = append((*destination)[:0], value.([]byte)...)
		default:
			return errors.New("unexpected scan destination")
		}
	}

	return nil
}
