package site

import "github.com/vernal96/go-cms/kernel"

type ID int64

type Site struct {
	ID          ID
	ProfileCode kernel.ProfileCode
	Domain      string
	Locale      string
	Settings    map[string]any
}
