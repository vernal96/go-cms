package postgresresourcefield

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vernal96/go-cms/core"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) (*Repository, error) {
	if pool == nil {
		return nil, errors.New("postgres resource field value repository pool is nil")
	}

	return &Repository{
		pool: pool,
	}, nil
}

func (r *Repository) FindByResourceID(
	ctx context.Context,
	resourceID core.ResourceID,
) ([]core.ResourceFieldValue, error) {
	if resourceID <= 0 {
		return nil, errors.New("resource field value resource id must be positive")
	}

	rows, err := r.pool.Query(ctx, `
SELECT resource_id, field, value
FROM resource_field_values
WHERE resource_id = $1
ORDER BY field;
`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("find field values for resource %d: %w", resourceID, err)
	}
	defer rows.Close()

	values := make([]core.ResourceFieldValue, 0)
	for rows.Next() {
		value, err := scanResourceFieldValue(rows)
		if err != nil {
			return nil, fmt.Errorf("scan field value for resource %d: %w", resourceID, err)
		}

		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate field values for resource %d: %w", resourceID, err)
	}

	return values, nil
}

func (r *Repository) FindByResourceAndField(
	ctx context.Context,
	resourceID core.ResourceID,
	field core.ResourceFieldCode,
) (core.ResourceFieldValue, error) {
	if resourceID <= 0 {
		return core.ResourceFieldValue{}, errors.New(
			"resource field value resource id must be positive",
		)
	}
	if field == "" {
		return core.ResourceFieldValue{}, errors.New("resource field value field is empty")
	}

	value, err := scanResourceFieldValue(r.pool.QueryRow(ctx, `
SELECT resource_id, field, value
FROM resource_field_values
WHERE resource_id = $1 AND field = $2
LIMIT 1;
`, resourceID, field))
	if errors.Is(err, pgx.ErrNoRows) {
		return core.ResourceFieldValue{}, core.ErrResourceFieldValueNotFound
	}
	if err != nil {
		return core.ResourceFieldValue{}, fmt.Errorf(
			"find field value %q for resource %d: %w",
			field,
			resourceID,
			err,
		)
	}

	return value, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanResourceFieldValue(row rowScanner) (core.ResourceFieldValue, error) {
	var value core.ResourceFieldValue
	var rawValue []byte

	if err := row.Scan(
		&value.ResourceID,
		&value.Field,
		&rawValue,
	); err != nil {
		return core.ResourceFieldValue{}, err
	}

	if err := json.Unmarshal(rawValue, &value.Value); err != nil {
		return core.ResourceFieldValue{}, fmt.Errorf("unmarshal resource field value: %w", err)
	}

	return value, nil
}

var _ core.ResourceFieldValueRepository = (*Repository)(nil)
