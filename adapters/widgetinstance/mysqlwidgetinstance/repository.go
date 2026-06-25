package mysqlwidgetinstance

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("mysql widget instance repository db is nil")
	}

	return &Repository{
		db: db,
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

	templateRows, err := r.db.QueryContext(ctx, `
SELECT id, resource_template, widget, template, area, params, sort
FROM template_widgets
WHERE resource_template = ?
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

	resourceRows, err := r.db.QueryContext(ctx, `
SELECT id, resource_id, widget, template, area, params, sort
FROM resource_widgets
WHERE resource_id = ?
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
	var rawParams sql.NullString

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

	if rawParams.Valid && rawParams.String != "" {
		if err := json.Unmarshal([]byte(rawParams.String), &instance.Params); err != nil {
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
