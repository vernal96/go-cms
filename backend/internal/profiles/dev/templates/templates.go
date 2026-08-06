package templates

import "github.com/vernal96/go-cms/kernel/modules/core/template"

func All() []template.Definition {
	return []template.Definition{Page(), Landing()}
}
