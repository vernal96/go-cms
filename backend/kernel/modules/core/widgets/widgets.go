package widgets

import "github.com/vernal96/go-cms/kernel/modules/core/widget"

// All returns every widget declared by the core module.
func All() []widget.Widget {
	return []widget.Widget{
		contentWidget{},
	}
}
