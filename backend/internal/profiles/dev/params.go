package dev

import "github.com/vernal96/go-cms/kernel/modules/core/field"

func Params() []field.Definition {
	required := true
	integerStep := int64(1)
	floatStep := 0.1

	return []field.Definition{
		{
			Key:      "string_value",
			Type:     field.TypeString,
			Label:    "Строка",
			Required: &required,
			Rules:    []string{"min=2", "max=120"},
		},
		{
			Key:      "integer_value",
			Type:     field.TypeInteger,
			Label:    "Целое число",
			Required: &required,
			Rules:    []string{"min=0", "max=1000"},
			Options:  field.IntegerOptions{Step: &integerStep},
		},
		{
			Key:      "float_value",
			Type:     field.TypeFloat,
			Label:    "Дробное число",
			Required: &required,
			Rules:    []string{"min=0", "max=100"},
			Options:  field.FloatOptions{Step: &floatStep},
		},
		{
			Key:      "checkbox_value",
			Type:     field.TypeCheckbox,
			Label:    "Флаг",
			Required: &required,
		},
		{
			Key:      "radio_value",
			Type:     field.TypeRadio,
			Label:    "Переключатель",
			Required: &required,
			Options: field.RadioOptions{Choices: []field.Choice{
				{Value: "first", Label: "Первый"},
				{Value: "second", Label: "Второй"},
			}},
		},
		{
			Key:      "select_value",
			Type:     field.TypeSelect,
			Label:    "Одиночный список",
			Required: &required,
			Options:  field.SelectOptions{Choices: commonChoices()},
		},
		{
			Key:   "multi_select_value",
			Type:  field.TypeSelect,
			Label: "Множественный список",
			Options: field.SelectOptions{
				Choices:  commonChoices(),
				Multiple: true,
			},
		},
		{
			Key:   "textarea_value",
			Type:  field.TypeTextarea,
			Label: "Многострочный текст",
			Rules: []string{"max=1000"},
		},
		{
			Key:   "email_value",
			Type:  field.TypeEmail,
			Label: "Электронная почта",
		},
		{
			Key:     "phone_value",
			Type:    field.TypePhone,
			Label:   "Телефон",
			Options: field.PhoneOptions{},
		},
	}
}

func commonChoices() []field.Choice {
	return []field.Choice{
		{Value: "alpha", Label: "Альфа"},
		{Value: "beta", Label: "Бета"},
		{Value: "gamma", Label: "Гамма"},
	}
}
