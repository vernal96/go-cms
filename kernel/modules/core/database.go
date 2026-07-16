package core

import "github.com/vernal96/go-cms/kernel/modules/core/site"

type Database interface {
	Sites() site.Repository
}
