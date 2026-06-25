package mysqlresourcefield

import (
	"errors"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestNewRepositoryRejectsNilDB(t *testing.T) {
	repository, err := NewRepository(nil)
	if err == nil || err.Error() != "mysql resource field value repository db is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository != nil {
		t.Fatal("repository must be nil when db is nil")
	}
}

func TestScanResourceFieldValue(t *testing.T) {
	value, err := scanResourceFieldValue(testRow{
		resourceID: 12,
		field:      "content",
		value:      []byte(`{"text":"Hello"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := core.ResourceFieldValue{
		ResourceID: 12,
		Field:      "content",
		Value: map[string]any{
			"text": "Hello",
		},
	}
	if !reflect.DeepEqual(value, expected) {
		t.Fatalf("unexpected field value: %#v", value)
	}
}

func TestScanResourceFieldValueNull(t *testing.T) {
	value, err := scanResourceFieldValue(testRow{
		resourceID: 12,
		field:      "content",
		value:      []byte(`null`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Value != nil {
		t.Fatalf("expected nil value, got %#v", value.Value)
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
