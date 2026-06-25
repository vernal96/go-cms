package core

type ResourceData struct {
	Resource Resource
	Fields   []ResourceFieldDefinition
	Values   []ResourceFieldValue
	Widgets  []WidgetInstance
}
