package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
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
    settings
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

func (r *Repository) UpdateSettings(
	ctx context.Context,
	id site.ID,
	settings map[string]any,
) error {
	if ctx == nil {
		return errors.New("update site settings context is nil")
	}
	if id <= 0 {
		return errors.New("invalid site id")
	}
	if settings == nil {
		settings = map[string]any{}
	}

	rawSettings, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encode settings for site %d: %w", id, err)
	}

	commandTag, err := r.connector.Pool().Exec(ctx, `
UPDATE core.sites
SET
    settings = $2::jsonb,
    updated_at = now()
WHERE id = $1;
`, id, string(rawSettings))
	if err != nil {
		return fmt.Errorf("update settings for core site %d: %w", id, err)
	}
	if commandTag.RowsAffected() == 0 {
		return site.ErrNotFound
	}

	return nil
}

var _ site.Repository = (*Repository)(nil)
