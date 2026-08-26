package postgres

import (
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
)

func TestCustomNegativeFilterUsesNotExists(t *testing.T) {
	args := []any{}
	add := func(value any) string { args = append(args, value); return "$" + string(rune('0'+len(args))) }
	fragment, err := resourceQueryFilterFor(resource.FilterCondition{
		Field: resource.FieldPath("resource.field.tags"), Operator: resource.FilterNotIn,
		Value: []string{"blocked", "hidden"}, Kind: field.StorageString,
	}, add, "item", libraryItemQueryColumn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(fragment, "NOT EXISTS") || !strings.Contains(fragment, "value.resource_id = item.id") || !strings.Contains(fragment, "= ANY") {
		t.Fatalf("negative filter SQL = %s", fragment)
	}
}

func TestLibraryKeysetUsesMissingLastAndStableID(t *testing.T) {
	args := []any{}
	add := func(value any) string { args = append(args, value); return "$x" }
	predicate := libraryItemKeysetPredicate([]libraryItemSortExpression{{
		sort: resource.Sort{Field: resource.FieldPublishedAt, Direction: resource.SortDescending},
		sql:  "item.published_at",
	}}, resource.LibraryCursor{Values: []any{int64(7)}, ID: 12}, resource.SortAscending, add)
	if !strings.Contains(predicate, "item.published_at<$x OR item.published_at IS NULL") || !strings.Contains(predicate, "item.id>$x") {
		t.Fatalf("keyset SQL = %s", predicate)
	}
	nullPredicate := libraryItemKeysetPredicate([]libraryItemSortExpression{{
		sort: resource.Sort{Field: resource.FieldPublishedAt, Direction: resource.SortAscending},
		sql:  "item.published_at",
	}}, resource.LibraryCursor{Values: []any{nil}, ID: 12}, resource.SortAscending, add)
	if strings.Contains(nullPredicate, "published_at>") || !strings.Contains(nullPredicate, "published_at IS NULL") {
		t.Fatalf("null keyset SQL = %s", nullPredicate)
	}
}
