package templates

import (
	"github.com/vernal96/go-cms-kernel/modules/core/field"
	"github.com/vernal96/go-cms-kernel/modules/core/template"
	corewidgets "github.com/vernal96/go-cms-kernel/modules/core/widgets"
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
			{Key: "gallery", Type: field.TypeMedia, Label: "Галерея", Options: field.MediaOptions{SettingsCode: "image", Multiple: true, MaxItems: 10}},
			{Key: "tags", Type: field.TypeString, Label: "Теги", Rules: []string{"max=80"}, Options: field.StringOptions{Multiple: true, MaxItems: 10}},
			{Key: "scores", Type: field.TypeInteger, Label: "Числовые значения", Rules: []string{"min=0", "max=100"}, Options: field.IntegerOptions{Multiple: true, MaxItems: 10}},
			{Key: "page_media", Type: field.TypeMedia, Label: "Медиа", Options: field.MediaOptions{SettingsCode: "image"}},
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
			{Code: "multiple", Label: "Множественные поля", Fields: []string{"gallery", "tags", "scores"}},
			{Code: "content", Label: "Контент", Fields: []string{"page_title", "page_text", "page_media", "show_title"}},
			{Code: "layout", Label: "Макет", Fields: []string{"layout"}},
		},
	}
}
