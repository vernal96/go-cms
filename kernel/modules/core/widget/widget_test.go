package widget

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
)

type testResolver map[field.TypeCode]field.Type

func (r testResolver) FieldType(
	code field.TypeCode,
) (field.Type, bool) {
	fieldType, exists := r[code]
	return fieldType, exists
}

func standardResolver() testResolver {
	result := make(testResolver)
	for _, fieldType := range field.StandardTypes() {
		result[fieldType.Code()] = fieldType
	}
	return result
}

type testWidget struct {
	definition Definition
	new        func(map[string]any) (Instance, error)
}

func (w *testWidget) Definition() Definition {
	return CloneDefinition(w.definition)
}

func (w *testWidget) New(values map[string]any) (Instance, error) {
	return w.new(values)
}

type testInstance struct {
	data map[string]any
}

func (i testInstance) Render(
	context.Context,
	RenderInput,
) (map[string]any, error) {
	return i.data, nil
}

func boolPointer(value bool) *bool {
	return &value
}

func TestCatalogQualifiesCompilesAndClonesWidgets(t *testing.T) {
	var received map[string]any
	catalog, err := Compile(
		[]Source{{
			Module: "content",
			Widgets: []Widget{&testWidget{
				definition: Definition{
					Code:        "summary",
					Label:       "Summary",
					Description: "Article summary",
					Fields: []field.Definition{{
						Key:      "count",
						Type:     field.TypeInteger,
						Label:    "Count",
						Required: boolPointer(true),
					}},
				},
				new: func(values map[string]any) (Instance, error) {
					received = values
					return testInstance{data: values}, nil
				},
			}},
		}},
		standardResolver(),
	)
	if err != nil {
		t.Fatal(err)
	}

	runtime, exists := catalog.Widget("content_summary")
	if !exists {
		t.Fatal("qualified widget is unavailable")
	}
	instance, err := runtime.New(map[string]any{
		"count": json.Number("3"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if received["count"] != int64(3) {
		t.Fatalf("normalized params = %#v", received)
	}
	data, err := instance.Render(context.Background(), RenderInput{})
	if err != nil || !reflect.DeepEqual(data, received) {
		t.Fatalf("render data = %#v, %v", data, err)
	}

	definitions := catalog.Definitions()
	if len(definitions) != 1 ||
		definitions[0].Code != "content_summary" {
		t.Fatalf("definitions = %#v", definitions)
	}
	definitions[0].Fields[0].Label = "Changed"
	if runtime.Definition().Fields[0].Label != "Count" {
		t.Fatal("catalog definition shares caller memory")
	}
}

func TestRuntimeClassifiesParamsAndInstanceFailures(t *testing.T) {
	catalog, err := Compile(
		[]Source{{
			Module: "content",
			Widgets: []Widget{&testWidget{
				definition: Definition{
					Code:        "summary",
					Label:       "Summary",
					Description: "Article summary",
					Fields: []field.Definition{{
						Key:      "title",
						Type:     field.TypeString,
						Label:    "Title",
						Required: boolPointer(true),
					}},
				},
				new: func(map[string]any) (Instance, error) {
					return nil, errors.New("constructor failed")
				},
			}},
		}},
		standardResolver(),
	)
	if err != nil {
		t.Fatal(err)
	}
	runtime, _ := catalog.Widget("content_summary")

	if _, err := runtime.New(map[string]any{}); !errors.Is(
		err,
		ErrInvalidParams,
	) {
		t.Fatalf("invalid params error = %v", err)
	}
	if _, err := runtime.New(map[string]any{
		"title": "Article",
	}); !errors.Is(err, ErrInstanceFailed) {
		t.Fatalf("instance error = %v", err)
	}
}

func TestCatalogRejectsInvalidAndDuplicateWidgets(t *testing.T) {
	valid := func(code Code) Widget {
		return &testWidget{
			definition: Definition{
				Code:        code,
				Label:       "Widget",
				Description: "Description",
			},
			new: func(map[string]any) (Instance, error) {
				return testInstance{}, nil
			},
		}
	}

	testCases := []struct {
		name    string
		sources []Source
		match   string
	}{
		{
			name: "typed nil",
			sources: []Source{{
				Module:  "content",
				Widgets: []Widget{(*testWidget)(nil)},
			}},
			match: "is nil",
		},
		{
			name: "duplicate local",
			sources: []Source{{
				Module: "content",
				Widgets: []Widget{
					valid("summary"),
					valid("summary"),
				},
			}},
			match: "duplicate widget code",
		},
		{
			name: "duplicate global",
			sources: []Source{
				{Module: "content_news", Widgets: []Widget{valid("top")}},
				{Module: "content", Widgets: []Widget{valid("news_top")}},
			},
			match: "duplicate global widget code",
		},
		{
			name: "empty description",
			sources: []Source{{
				Module: "content",
				Widgets: []Widget{&testWidget{
					definition: Definition{
						Code:  "summary",
						Label: "Summary",
					},
				}},
			}},
			match: "invalid description",
		},
		{
			name: "unknown field type",
			sources: []Source{{
				Module: "content",
				Widgets: []Widget{&testWidget{
					definition: Definition{
						Code:        "summary",
						Label:       "Summary",
						Description: "Description",
						Fields: []field.Definition{{
							Key:   "value",
							Type:  "unknown",
							Label: "Value",
						}},
					},
				}},
			}},
			match: "unknown type",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			_, err := Compile(test.sources, standardResolver())
			if err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
