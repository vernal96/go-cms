package widgetviews

import (
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	corewidgets "github.com/vernal96/go-cms/kernel/modules/core/widgets"
)

var (
	ContentCompact = widget.NewView(corewidgets.Content, "compact", "Компактный")
	ContentArticle = widget.NewView(corewidgets.Content, "article", "Статья")
)
