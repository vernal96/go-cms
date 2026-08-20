package widgets

import (
	"context"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/widget"
)

func TestContentWidgetReturnsResourceContent(t *testing.T) {
	declared := All()
	if len(declared) != 1 {
		t.Fatalf("widgets = %#v", declared)
	}

	definition := declared[0].Definition()
	if definition.Reference != Content ||
		definition.Reference.Code() != "content" ||
		definition.Code != "" ||
		definition.Label == "" ||
		definition.Description == "" ||
		len(definition.Fields) != 0 {
		t.Fatalf("definition = %#v", definition)
	}

	instance, err := declared[0].New(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := instance.Render(context.Background(), widget.RenderInput{
		Resource: widget.ResourceSnapshot{
			ID:      42,
			Content: "Resource content",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 || data["content"] != "Resource content" {
		t.Fatalf("data = %#v", data)
	}
}
