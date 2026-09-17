// Package seeds declares this application's development data.
package seeds

import (
	"embed"
	"github.com/vernal96/go-cms-kernel/seeds"
)

//go:embed dev/*.sql
var files embed.FS

func Dev() seeds.Source {
	return seeds.Source{ID: "sites_dev", Tags: []seeds.Tag{"dev"}, Schema: "core", FS: files, Path: "dev"}
}
