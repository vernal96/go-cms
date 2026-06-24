package postgresresourcefield

import (
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

func TestScanResourceFieldValue(t *testing.T) {
	row := testRow{
		resourceID: 12,
		field:      "content",
		value:      []byte(`{"blocks":[{"type":"text","value":"Hello"}]}`),
	}

	value, err := scanResourceFieldValue(row)
	if err != nil {
		t.Fatal(err)
	}

	if value.ResourceID != 12 {
		t.Fatalf("unexpected resource id: %d", value.ResourceID)
	}
	if value.Field != "content" {
		t.Fatalf("unexpected field code: %q", value.Field)
	}

	expected := map[string]any{
		"blocks": []any{
			map[string]any{
				"type":  "text",
				"value": "Hello",
			},
		},
	}
	if !reflect.DeepEqual(value.Value, expected) {
		t.Fatalf("unexpected value: %#v", value.Value)
	}
}

func TestScanResourceFieldValueReturnsJSONError(t *testing.T) {
	_, err := scanResourceFieldValue(testRow{
		resourceID: 12,
		field:      "content",
		value:      []byte(`{`),
	})
	if err == nil {
		t.Fatal("expected JSON error")
	}
}

type testRow struct {
	resourceID core.ResourceID
	field      core.ResourceFieldCode
	value      []byte
	err        error
}

func (r testRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 3 {
		return errors.New("unexpected scan destination count")
	}

	resourceID, ok := dest[0].(*core.ResourceID)
	if !ok {
		return errors.New("unexpected resource id destination")
	}
	field, ok := dest[1].(*core.ResourceFieldCode)
	if !ok {
		return errors.New("unexpected field destination")
	}
	value, ok := dest[2].(*[]byte)
	if !ok {
		return errors.New("unexpected value destination")
	}

	*resourceID = r.resourceID
	*field = r.field
	*value = append((*value)[:0], r.value...)

	return nil
}
