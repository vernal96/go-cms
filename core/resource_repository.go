package core

import (
	"context"
	"errors"
)

var ErrResourceNotFound = errors.New("resource not found")

type ResourceRepository interface {
	FindByID(ctx context.Context, id ResourceID) (Resource, error)
	FindByPath(ctx context.Context, siteID int64, path string) (Resource, error)
	FindChildren(ctx context.Context, parentID ResourceID) ([]Resource, error)
}
