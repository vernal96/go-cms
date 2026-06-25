package postgreswidgetinstance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vernal96/go-cms/core"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) (*Repository, error) {
	if pool == nil {
		return nil, errors.New("postgres widget instance repository pool is nil")
	}

	return &Repository{
		pool: pool,
	}, nil
}

func (r *Repository) FindForResource(
	ctx context.Context,
	resource core.Resource,
) ([]core.WidgetInstance, error) {
	if resource.ID <= 0 {
		return nil, errors.New("widget instance resource id must be positive")
	}
	if resource.Template == "" {
		return nil, errors.New("widget instance resource template is empty")
	}

	instances := make([]core.WidgetInstance, 0)

	templateRows, err := r.pool.Query(ctx, `
SELECT id, resource_template, widget, template, area, params, sort
FROM template_widgets
WHERE resource_template = $1
ORDER BY area, sort, id;
`, resource.Template)
	if err != nil {
		return nil, fmt.Errorf(
			"find template widget instances for resource template %q: %w",
			resource.Template,
			err,
		)
	}

	for templateRows.Next() {
		instance, err := scanWidgetInstance(
			templateRows,
			core.WidgetInstanceSourceTemplate,
		)
		if err != nil {
			templateRows.Close()
			return nil, fmt.Errorf(
				"scan template widget instance for resource template %q: %w",
				resource.Template,
				err,
			)
		}

		instances = append(instances, instance)
	}
	if err := templateRows.Err(); err != nil {
		templateRows.Close()
		return nil, fmt.Errorf(
			"iterate template widget instances for resource template %q: %w",
			resource.Template,
			err,
		)
	}
	templateRows.Close()

	resourceRows, err := r.pool.Query(ctx, `
SELECT id, resource_id, widget, template, area, params, sort
FROM resource_widgets
WHERE resource_id = $1
ORDER BY area, sort, id;
`, resource.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"find widget instances for resource %d: %w",
			resource.ID,
			err,
		)
	}
	defer resourceRows.Close()

	for resourceRows.Next() {
		instance, err := scanWidgetInstance(
			resourceRows,
			core.WidgetInstanceSourceResource,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan widget instance for resource %d: %w",
				resource.ID,
				err,
			)
		}

		instances = append(instances, instance)
	}
	if err := resourceRows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate widget instances for resource %d: %w",
			resource.ID,
			err,
		)
	}

	return instances, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanWidgetInstance(
	row rowScanner,
	source core.WidgetInstanceSource,
) (core.WidgetInstance, error) {
	var instance core.WidgetInstance
	var rawParams []byte

	instance.Source = source

	var err error
	switch source {
	case core.WidgetInstanceSourceTemplate:
		err = row.Scan(
			&instance.ID,
			&instance.ResourceTemplate,
			&instance.Widget,
			&instance.Template,
			&instance.Area,
			&rawParams,
			&instance.Sort,
		)
	case core.WidgetInstanceSourceResource:
		err = row.Scan(
			&instance.ID,
			&instance.ResourceID,
			&instance.Widget,
			&instance.Template,
			&instance.Area,
			&rawParams,
			&instance.Sort,
		)
	default:
		return core.WidgetInstance{}, fmt.Errorf(
			"unsupported widget instance source %q",
			source,
		)
	}
	if err != nil {
		return core.WidgetInstance{}, err
	}

	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &instance.Params); err != nil {
			return core.WidgetInstance{}, fmt.Errorf(
				"unmarshal widget instance params: %w",
				err,
			)
		}
	}
	if instance.Params == nil {
		instance.Params = core.WidgetParams{}
	}

	return instance, nil
}

var _ core.WidgetInstanceRepository = (*Repository)(nil)
