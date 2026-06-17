package fields

type TypeCode string

type FieldType interface {
	Code() TypeCode
}

type OptionableFieldType interface {
	FieldType
	SupportsOptions() bool
}

type MultipleFieldType interface {
	FieldType
	SupportsMultiple() bool
}
