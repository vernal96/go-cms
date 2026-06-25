package core

type WidgetArea string

type WidgetInstanceSource string

const (
	WidgetInstanceSourceTemplate WidgetInstanceSource = "template"
	WidgetInstanceSourceResource WidgetInstanceSource = "resource"
)

type WidgetInstance struct {
	ID               int64
	Source           WidgetInstanceSource
	ResourceID       ResourceID
	ResourceTemplate ResourceTemplateCode
	Widget           WidgetCode
	Template         WidgetTemplateCode
	Area             WidgetArea
	Params           WidgetParams
	Sort             int
}
