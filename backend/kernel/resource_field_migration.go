package kernel

import (
	"context"
	"io"
)

type ResourceFieldMigrationReport struct {
	Resources int      `json:"resources"`
	Rows      int      `json:"rows"`
	Issues    []string `json:"issues,omitempty"`
}

type ResourceFieldMigrator interface {
	PrepareResourceFields(context.Context, map[ProfileCode]*ProfileBlueprint) (ResourceFieldMigrationReport, error)
	AuditResourceFields(context.Context, map[ProfileCode]*ProfileBlueprint) (ResourceFieldMigrationReport, error)
	RepairResourceFields(context.Context, map[ProfileCode]*ProfileBlueprint, io.Reader) (ResourceFieldMigrationReport, error)
}
