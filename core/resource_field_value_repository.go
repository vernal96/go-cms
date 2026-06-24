package core

import (
	"context"
	"errors"
)

var ErrResourceFieldValueNotFound = errors.New("resource field value not found")

type ResourceFieldValueRepository interface {
	FindByResourceID(ctx context.Context, resourceID ResourceID) ([]ResourceFieldValue, error)
	FindByResourceAndField(
		ctx context.Context,
		resourceID ResourceID,
		field ResourceFieldCode,
	) (ResourceFieldValue, error)
}
