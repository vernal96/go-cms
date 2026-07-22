package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type SiteRepository struct {
	connector *connectorpostgres.Connector
}

func NewSiteRepository(
	connector *connectorpostgres.Connector,
) (*SiteRepository, error) {
	if connector == nil {
		return nil, errors.New("postgres connector is nil")
	}

	if connector.Pool() == nil {
		return nil, errors.New("postgres pool is nil")
	}

	return &SiteRepository{connector: connector}, nil
}

func (r *SiteRepository) List(
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
			if err := json.Unmarshal(rawSettings, &item.Settings); err != nil {
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

var _ site.Repository = (*SiteRepository)(nil)
