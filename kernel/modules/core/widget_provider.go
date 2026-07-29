package core

import (
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/modules/core/widgets"
)

func (*Runtime) Widgets() []widget.Widget {
	return widgets.All()
}

var _ widget.Provider = (*Runtime)(nil)
