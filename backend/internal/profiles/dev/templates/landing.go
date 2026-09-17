package templates

import (
	"github.com/vernal96/go-cms-kernel/modules/core/field"
	"github.com/vernal96/go-cms-kernel/modules/core/template"
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
			{Key: "slides", Type: field.TypeRepeater, Label: "Слайды", Options: field.RepeaterOptions{
				MaxItems: 10,
				Fields: []field.Definition{
					{Key: "title", Type: field.TypeString, Label: "Заголовок", Required: &required, Rules: []string{"max=120"}},
					{Key: "text", Type: field.TypeTextarea, Label: "Текст"},
					{Key: "image", Type: field.TypeMedia, Label: "Изображение", Options: field.MediaOptions{SettingsCode: "image"}},
					{Key: "link", Type: field.TypeString, Label: "Ссылка"},
					{Key: "active", Type: field.TypeCheckbox, Label: "Активен"},
					{Key: "attachment", Type: field.TypeFile, Label: "Файл"},
				},
			}},
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
			{
				Key:     "feature_points",
				Type:    field.TypeString,
				Label:   "Преимущества",
				Rules:   []string{"max=120"},
				Options: field.StringOptions{Multiple: true, MaxItems: 6},
			},
			{
				Key:     "promo_texts",
				Type:    field.TypeTextarea,
				Label:   "Промо-тексты",
				Rules:   []string{"max=500"},
				Options: field.StringOptions{Multiple: true, MaxItems: 3},
			},
			{
				Key:     "metrics",
				Type:    field.TypeInteger,
				Label:   "Метрики",
				Rules:   []string{"min=0", "max=1000000"},
				Options: field.IntegerOptions{Multiple: true, MaxItems: 6},
			},
			{
				Key:     "prices",
				Type:    field.TypeFloat,
				Label:   "Цены",
				Rules:   []string{"min=0", "max=1000000"},
				Options: field.FloatOptions{Multiple: true, MaxItems: 6, Step: &floatStep},
			},
			{Key: "contact_email", Type: field.TypeEmail, Label: "Email для связи", Options: field.StringOptions{Multiple: true, MaxItems: 3}},
			{Key: "contact_phone", Type: field.TypePhone, Label: "Телефоны для связи", Options: field.PhoneOptions{Multiple: true, MaxItems: 3}},
			{Key: "downloads", Type: field.TypeFile, Label: "Файлы для скачивания", Options: field.FileOptions{Multiple: true, MaxItems: 4}},
			{Key: "showcase_media", Type: field.TypeMedia, Label: "Медиа витрины", Options: field.MediaOptions{SettingsCode: "image", Multiple: true, MaxItems: 8}},
		},
		EditorTabs: []field.EditorTab{
			{Code: "slides", Label: "Слайды", Fields: []string{"slides"}},
			{Code: "content", Label: "Первый экран", Fields: []string{"hero_title", "hero_text"}},
			{Code: "layout", Label: "Макет", Fields: []string{"columns", "content_width"}},
			{Code: "audience", Label: "Аудитория", Fields: []string{"audiences"}},
			{Code: "fields", Label: "Поля", Fields: []string{"feature_points", "promo_texts", "metrics", "prices", "contact_email", "contact_phone", "downloads", "showcase_media"}},
		},
	}
}
