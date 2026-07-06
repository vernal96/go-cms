package coremodule

import "github.com/vernal96/go-cms/core"

type SiteID int64

type Site struct {
	ID          SiteID
	ProfileCode core.ProfileCode
	Domain      string
	Locale      string
	Settings    map[string]any
}
