package templates

import (
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	corewidgets "github.com/vernal96/go-cms/kernel/modules/core/widgets"
)

func Page() template.Definition {
	required := true

	return template.Definition{
		Code:  "page",
		Label: "Страница",
		Icon:  "document",
		Layout: template.Layout{
			Body: []template.Item{
				template.Widget{Widget: corewidgets.Content},
				template.ResourceWidgets{},
			},
			Sidebar: []template.Item{
				template.ResourceWidgets{},
			},
		},
		Fields: []field.Definition{
			{
				Key:      "page_title",
				Type:     field.TypeString,
				Label:    "Заголовок страницы",
				Required: &required,
				Rules:    []string{"min=2", "max=120"},
			},
			{
				Key:   "page_text",
				Type:  field.TypeTextarea,
				Label: "Текст страницы",
				Rules: []string{"max=2000"},
			},
			{
				Key:      "show_title",
				Type:     field.TypeCheckbox,
				Label:    "Показывать заголовок",
				Required: &required,
			},
			{
				Key:      "layout",
				Type:     field.TypeRadio,
				Label:    "Макет",
				Required: &required,
				Options: field.RadioOptions{Choices: []field.Choice{
					{Value: "standard", Label: "Стандартный"},
					{Value: "wide", Label: "Широкий"},
				}},
			},
		},
		EditorTabs: []field.EditorTab{
			{Code: "content", Label: "Контент", Fields: []string{"page_title", "page_text", "show_title"}},
			{Code: "layout", Label: "Макет", Fields: []string{"layout"}},
		},
	}
}
