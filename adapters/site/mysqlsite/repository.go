package mysqlsite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/adapters/database/mysqldb"
	"github.com/vernal96/go-cms/core"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("mysql site repository db is nil")
	}

	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) Migrate(ctx context.Context) error {
	return mysqldb.Migrate(ctx, r.db)
}

func (r *Repository) EnsureSite(ctx context.Context, site core.Site) error {
	if site.ProfileCode == "" {
		return errors.New("site profile code is empty")
	}
	if site.Domain == "" {
		return errors.New("site domain is empty")
	}
	if site.Locale == "" {
		site.Locale = "ru"
	}
	if site.Settings == nil {
		site.Settings = map[string]any{}
	}

	settings, err := json.Marshal(site.Settings)
	if err != nil {
		return fmt.Errorf("marshal site settings: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO sites (profile_code, domain, locale, settings)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	profile_code = VALUES(profile_code),
	locale = VALUES(locale),
	settings = VALUES(settings),
	updated_at = CURRENT_TIMESTAMP(6);
`, site.ProfileCode, site.Domain, site.Locale, string(settings))
	if err != nil {
		return fmt.Errorf("ensure site %q: %w", site.Domain, err)
	}

	return nil
}

func (r *Repository) FindByDomain(ctx context.Context, domain string) (core.Site, error) {
	if domain == "" {
		return core.Site{}, errors.New("site domain is empty")
	}

	site, err := scanSite(r.db.QueryRowContext(ctx, `
SELECT id, profile_code, domain, locale, settings
FROM sites
WHERE domain = ?
LIMIT 1;
`, domain))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Site{}, core.ErrSiteNotFound
	}
	if err != nil {
		return core.Site{}, fmt.Errorf("find site by domain %q: %w", domain, err)
	}

	return site, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSite(row rowScanner) (core.Site, error) {
	var site core.Site
	var settings []byte

	if err := row.Scan(
		&site.ID,
		&site.ProfileCode,
		&site.Domain,
		&site.Locale,
		&settings,
	); err != nil {
		return core.Site{}, err
	}

	if len(settings) > 0 {
		if err := json.Unmarshal(settings, &site.Settings); err != nil {
			return core.Site{}, fmt.Errorf("unmarshal site settings: %w", err)
		}
	}
	if site.Settings == nil {
		site.Settings = map[string]any{}
	}

	return site, nil
}

var _ core.SiteRepository = (*Repository)(nil)
