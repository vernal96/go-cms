package site

import "github.com/vernal96/go-cms/kernal"

type ID int64

type Site struct {
	ID          ID
	ProfileCode kernal.ProfileCode
	Domain      string
	Locale      string
	Settings    map[string]any
}
