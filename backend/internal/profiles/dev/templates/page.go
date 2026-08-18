package templates

import (
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
)

func Page() template.Definition {
	required := true

	return template.Definition{
		Code:  "page",
		Label: "Страница",
		Icon:  "document",
		Layout: template.Layout{
			Body: []template.LayoutItem{
				{
					Kind:         template.ItemWidget,
					Key:          "content",
					Widget:       "core_content",
					Presentation: widget.DefaultPresentation(),
				},
				{Kind: template.ItemResourceSlot},
			},
			Sidebar: []template.LayoutItem{
				{Kind: template.ItemResourceSlot},
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
	}
}
