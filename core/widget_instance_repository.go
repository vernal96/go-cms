package core

import "context"

type WidgetInstanceRepository interface {
	FindForResource(
		ctx context.Context,
		resource Resource,
	) ([]WidgetInstance, error)
}
