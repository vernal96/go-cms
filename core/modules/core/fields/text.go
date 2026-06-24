package fields

import corefields "github.com/vernal96/go-cms/core/fields"

const TextFieldTypeCode corefields.TypeCode = "text"

type TextFieldType struct{}

func NewTextFieldType() *TextFieldType {
	return &TextFieldType{}
}

func (f *TextFieldType) Code() corefields.TypeCode {
	return TextFieldTypeCode
}

var _ corefields.FieldType = (*TextFieldType)(nil)
