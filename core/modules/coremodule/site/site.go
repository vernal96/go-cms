package site

import "github.com/vernal96/go-cms/core"

type ID int64

type Site struct {
	ID          ID
	ProfileCode core.ProfileCode
	Domain      string
	Locale      string
	Settings    map[string]any
}
