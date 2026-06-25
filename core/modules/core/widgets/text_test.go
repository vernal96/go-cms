package widgets

import (
	"context"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestTextWidgetRender(t *testing.T) {
	result, err := NewTextWidget().Render(context.Background(), core.WidgetParams{
		"text": "<strong>Hello</strong>",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.HTML != "&lt;strong&gt;Hello&lt;/strong&gt;" {
		t.Fatalf("unexpected HTML: %q", result.HTML)
	}
}
