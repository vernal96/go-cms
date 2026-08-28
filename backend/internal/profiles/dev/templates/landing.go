package templates

import (
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
)

func Landing() template.Definition {
	required := true
	integerStep := int64(1)
	floatStep := 0.1

	return template.Definition{
		Code:  "landing",
		Label: "Лендинг",
		Icon:  "promotion",
		Fields: []field.Definition{
			{
				Key:      "hero_title",
				Type:     field.TypeString,
				Label:    "Заголовок первого экрана",
				Required: &required,
				Rules:    []string{"min=2", "max=120"},
			},
			{
				Key:   "hero_text",
				Type:  field.TypeTextarea,
				Label: "Текст первого экрана",
				Rules: []string{"max=2000"},
			},
			{
				Key:      "columns",
				Type:     field.TypeInteger,
				Label:    "Количество колонок",
				Required: &required,
				Rules:    []string{"min=1", "max=4"},
				Options:  field.IntegerOptions{Step: &integerStep},
			},
			{
				Key:      "content_width",
				Type:     field.TypeFloat,
				Label:    "Ширина контента",
				Required: &required,
				Rules:    []string{"min=1", "max=2"},
				Options:  field.FloatOptions{Step: &floatStep},
			},
			{
				Key:   "audiences",
				Type:  field.TypeSelect,
				Label: "Аудитории",
				Options: field.SelectOptions{
					Multiple: true,
					Choices: []field.Choice{
						{Value: "new", Label: "Новая"},
						{Value: "returning", Label: "Вернувшаяся"},
					},
				},
			},
		},
		EditorTabs: []field.EditorTab{
			{Code: "content", Label: "Первый экран", Fields: []string{"hero_title", "hero_text"}},
			{Code: "layout", Label: "Макет", Fields: []string{"columns", "content_width"}},
			{Code: "audience", Label: "Аудитория", Fields: []string{"audiences"}},
		},
	}
}
