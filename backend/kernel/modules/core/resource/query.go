package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

// FieldPath is a transport and database-neutral resource field identifier.
// Custom template values retain the existing resource.field.<key> namespace.
type FieldPath string

const (
	FieldID         FieldPath = "resource.id"
	FieldTitle      FieldPath = "resource.title"
	FieldMenuTitle  FieldPath = "resource.menu_title"
	FieldSlug       FieldPath = "resource.slug"
	FieldPathValue  FieldPath = "resource.path"
	FieldAnnotation FieldPath = "resource.annotation"
	FieldType       FieldPath = "resource.type"
	FieldTemplate   FieldPath = "resource.template"
	FieldSort       FieldPath = "resource.sort"
)

type FilterOperator string

const (
	FilterEqual              FilterOperator = "eq"
	FilterNotEqual           FilterOperator = "neq"
	FilterIn                 FilterOperator = "in"
	FilterNotIn              FilterOperator = "not_in"
	FilterGreaterThan        FilterOperator = "gt"
	FilterGreaterThanOrEqual FilterOperator = "gte"
	FilterLessThan           FilterOperator = "lt"
	FilterLessThanOrEqual    FilterOperator = "lte"
)

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

type FilterCondition struct {
	Field    FieldPath
	Operator FilterOperator
	Value    any
	Kind     field.StorageKind
}

type Sort struct {
	Field     FieldPath
	Direction SortDirection
	Kind      field.StorageKind
}

// Query expresses resource selection without leaking adapter/SQL concerns.
// When FilterByParent is true, Parent nil means root resources.
type Query struct {
	SiteID         site.ID
	FilterByParent bool
	Parent         *ID
	IDs            []ID
	ExcludeIDs     []ID
	Types          []resourcetype.Code
	Filters        []FilterCondition
	Sort           []Sort
	Fields         []FieldPath
	Limit          int
	Page           int
	PerPage        int
	PublicOnly     bool
}

type Page struct {
	Items []Resource
	Total int
}

type QueryRepository interface {
	Query(context.Context, Query) (Page, error)
}

type QueryService struct{ repository QueryRepository }

func NewQueryService(repository QueryRepository) (*QueryService, error) {
	if repository == nil {
		return nil, errors.New("resource query repository is nil")
	}
	return &QueryService{repository: repository}, nil
}

func (s *QueryService) Query(ctx context.Context, query Query) (Page, error) {
	if ctx == nil {
		return Page{}, errors.New("resource query context is nil")
	}
	if s == nil || s.repository == nil {
		return Page{}, errors.New("resource query service is nil")
	}
	if err := query.Validate(); err != nil {
		return Page{}, err
	}
	result, err := s.repository.Query(ctx, query)
	if err != nil {
		return Page{}, fmt.Errorf("query resources: %w", err)
	}
	for index := range result.Items {
		result.Items[index] = Clone(result.Items[index])
	}
	return result, nil
}

func (q Query) Validate() error {
	if q.SiteID <= 0 {
		return errors.New("resource query site id is invalid")
	}
	if q.Parent != nil && *q.Parent <= 0 {
		return errors.New("resource query parent id is invalid")
	}
	if q.Limit <= 0 || q.Limit > 100 {
		return errors.New("resource query limit must be between 1 and 100")
	}
	if q.PerPage < 0 || q.Page < 0 {
		return errors.New("resource query pagination is invalid")
	}
	if q.PerPage > 0 && (q.PerPage > q.Limit || q.Page < 1) {
		return errors.New("resource query page is invalid")
	}
	for _, id := range append(append([]ID(nil), q.IDs...), q.ExcludeIDs...) {
		if id <= 0 {
			return errors.New("resource query id is invalid")
		}
	}
	for _, typ := range q.Types {
		if typ == "" || strings.TrimSpace(string(typ)) != string(typ) {
			return errors.New("resource query type is invalid")
		}
	}
	for _, field := range q.Fields {
		if !ValidFieldPath(field) {
			return fmt.Errorf("resource query field %q is invalid", field)
		}
	}
	for _, condition := range q.Filters {
		if err := condition.Validate(); err != nil {
			return err
		}
	}
	for _, item := range q.Sort {
		if !SortableField(item.Field) || (item.Direction != SortAscending && item.Direction != SortDescending) {
			return errors.New("resource query sort is invalid")
		}
		if strings.HasPrefix(string(item.Field), "resource.field.") && !ValidStorageKind(item.Kind) {
			return errors.New("resource query custom sort storage kind is required")
		}
	}
	return nil
}

func (c FilterCondition) Validate() error {
	if !ValidFieldPath(c.Field) {
		return fmt.Errorf("resource query filter field %q is invalid", c.Field)
	}
	switch c.Operator {
	case FilterEqual, FilterNotEqual, FilterIn, FilterNotIn, FilterGreaterThan, FilterGreaterThanOrEqual, FilterLessThan, FilterLessThanOrEqual:
	default:
		return fmt.Errorf("resource query filter operator %q is invalid", c.Operator)
	}
	if (c.Operator == FilterIn || c.Operator == FilterNotIn) && !isSlice(c.Value) {
		return errors.New("resource query set filter value is invalid")
	}
	if (c.Operator == FilterGreaterThan || c.Operator == FilterGreaterThanOrEqual || c.Operator == FilterLessThan || c.Operator == FilterLessThanOrEqual) && !numericField(c.Field) {
		return fmt.Errorf("resource query filter %q does not support ordering", c.Field)
	}
	if strings.HasPrefix(string(c.Field), "resource.field.") && c.Kind != "" &&
		(c.Operator == FilterGreaterThan || c.Operator == FilterGreaterThanOrEqual || c.Operator == FilterLessThan || c.Operator == FilterLessThanOrEqual) &&
		c.Kind != field.StorageString && c.Kind != field.StorageInteger && c.Kind != field.StorageFloat && c.Kind != field.StorageTimestamp {
		return fmt.Errorf("resource query field %q storage kind %q does not support ordering", c.Field, c.Kind)
	}
	if strings.HasPrefix(string(c.Field), "resource.field.") && c.Kind != "" && !ValidStorageKind(c.Kind) {
		return fmt.Errorf("resource query field %q has invalid storage kind %q", c.Field, c.Kind)
	}
	return nil
}

func ValidStorageKind(kind field.StorageKind) bool {
	switch kind {
	case field.StorageString, field.StorageInteger, field.StorageFloat, field.StorageBoolean, field.StorageTimestamp, field.StorageReference, field.StorageJSON:
		return true
	default:
		return false
	}
}

func ValidFieldPath(field FieldPath) bool {
	switch field {
	case FieldID, FieldTitle, FieldMenuTitle, FieldSlug, FieldPathValue, FieldAnnotation, FieldType, FieldTemplate, FieldSort:
		return true
	}
	key, found := strings.CutPrefix(string(field), "resource.field.")
	return found && key != "" && strings.TrimSpace(key) == key && !strings.ContainsAny(key, ". \t\r\n")
}
func SortableField(field FieldPath) bool {
	switch field {
	case FieldID, FieldTitle, FieldMenuTitle, FieldSlug, FieldPathValue, FieldType, FieldTemplate, FieldSort:
		return true
	}
	return strings.HasPrefix(string(field), "resource.field.")
}
func numericField(field FieldPath) bool {
	return field == FieldID || field == FieldSort || strings.HasPrefix(string(field), "resource.field.")
}
func isSlice(value any) bool {
	switch value.(type) {
	case []any, []string, []int64, []ID:
		return true
	}
	return false
}

// Normalized returns an order-insensitive canonical copy suitable for result
// cache identity. Sort order and explicit ID order are intentionally kept.
func (q Query) Normalized() Query {
	q.IDs = append([]ID(nil), q.IDs...)
	sort.Slice(q.IDs, func(i, j int) bool { return q.IDs[i] < q.IDs[j] })
	q.Types = append([]resourcetype.Code(nil), q.Types...)
	sort.Slice(q.Types, func(i, j int) bool { return q.Types[i] < q.Types[j] })
	q.ExcludeIDs = append([]ID(nil), q.ExcludeIDs...)
	sort.Slice(q.ExcludeIDs, func(i, j int) bool { return q.ExcludeIDs[i] < q.ExcludeIDs[j] })
	q.Fields = append([]FieldPath(nil), q.Fields...)
	sort.Slice(q.Fields, func(i, j int) bool { return q.Fields[i] < q.Fields[j] })
	q.Filters = append([]FilterCondition(nil), q.Filters...)
	sort.SliceStable(q.Filters, func(i, j int) bool {
		left, _ := json.Marshal(q.Filters[i])
		right, _ := json.Marshal(q.Filters[j])
		return string(left) < string(right)
	})
	return q
}
