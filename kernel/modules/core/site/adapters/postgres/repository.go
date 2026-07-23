package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

type Repository struct {
	connector *connectorpostgres.Connector
}

func NewRepository(
	connector *connectorpostgres.Connector,
) (*Repository, error) {
	if connector == nil {
		return nil, errors.New("postgres connector is nil")
	}
	if connector.Pool() == nil {
		return nil, errors.New("postgres pool is nil")
	}

	return &Repository{connector: connector}, nil
}

func (r *Repository) List(
	ctx context.Context,
) ([]site.Site, error) {
	if ctx == nil {
		return nil, errors.New("list sites context is nil")
	}

	rows, err := r.connector.Pool().Query(ctx, `
SELECT
    id,
    profile_code,
    domain,
    locale,
    settings,
    is_public,
    created_at,
    updated_at,
    created_by,
    updated_by
FROM core.sites
ORDER BY id;
`)
	if err != nil {
		return nil, fmt.Errorf("query core sites: %w", err)
	}
	defer rows.Close()

	sites := make([]site.Site, 0)
	for rows.Next() {
		var item site.Site
		var rawSettings []byte

		if err := rows.Scan(
			&item.ID,
			&item.ProfileCode,
			&item.Domain,
			&item.Locale,
			&rawSettings,
			&item.IsPublic,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.CreatedBy,
			&item.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan core site: %w", err)
		}

		item.Settings = make(map[string]any)
		if len(rawSettings) > 0 {
			decoder := json.NewDecoder(bytes.NewReader(rawSettings))
			decoder.UseNumber()
			if err := decoder.Decode(&item.Settings); err != nil {
				return nil, fmt.Errorf(
					"decode settings for site %d: %w",
					item.ID,
					err,
				)
			}
		}

		sites = append(sites, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate core sites: %w", err)
	}

	return sites, nil
}

func (r *Repository) Update(
	ctx context.Context,
	actorID *security.UserID,
	item site.Site,
) (site.Site, error) {
	if ctx == nil {
		return site.Site{}, errors.New("update site context is nil")
	}
	if item.ID <= 0 {
		return site.Site{}, errors.New("invalid site id")
	}
	if item.Settings == nil {
		item.Settings = map[string]any{}
	}

	rawSettings, err := json.Marshal(item.Settings)
	if err != nil {
		return site.Site{}, fmt.Errorf(
			"encode settings for site %d: %w",
			item.ID,
			err,
		)
	}

	var (
		result      site.Site
		rawReturned []byte
	)
	err = r.connector.Pool().QueryRow(ctx, `
UPDATE core.sites
SET
    domain = $2,
    locale = $3,
    settings = $4::jsonb,
    is_public = $5,
    updated_at = now(),
    updated_by = $6
WHERE id = $1
RETURNING
    id, profile_code, domain, locale, settings, is_public,
    created_at, updated_at, created_by, updated_by;
`,
		item.ID,
		item.Domain,
		item.Locale,
		string(rawSettings),
		item.IsPublic,
		actorID,
	).Scan(
		&result.ID,
		&result.ProfileCode,
		&result.Domain,
		&result.Locale,
		&rawReturned,
		&result.IsPublic,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.CreatedBy,
		&result.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return site.Site{}, site.ErrNotFound
	}
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == pgerrcode.UniqueViolation {
			return site.Site{}, fmt.Errorf("%w: %s", site.ErrConflict, err)
		}
		return site.Site{}, fmt.Errorf(
			"update core site %d: %w",
			item.ID,
			err,
		)
	}

	result.Settings = make(map[string]any)
	decoder := json.NewDecoder(bytes.NewReader(rawReturned))
	decoder.UseNumber()
	if err := decoder.Decode(&result.Settings); err != nil {
		return site.Site{}, fmt.Errorf(
			"decode returned site settings: %w",
			err,
		)
	}
	return result, nil
}

var _ site.Repository = (*Repository)(nil)
