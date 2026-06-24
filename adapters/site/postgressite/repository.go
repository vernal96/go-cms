package postgressite

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

func Connect(ctx context.Context, dsn string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return NewRepository(pool)
}

func NewRepository(pool *pgxpool.Pool) (*Repository, error) {
	if pool == nil {
		return nil, errors.New("postgres site repository pool is nil")
	}

	return &Repository{
		pool: pool,
	}, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS sites (
	id BIGSERIAL PRIMARY KEY,
	profile_code TEXT NOT NULL,
	domain TEXT NOT NULL UNIQUE,
	locale TEXT NOT NULL DEFAULT 'ru',
	settings JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sites_profile_code ON sites(profile_code);
`)
	if err != nil {
		return fmt.Errorf("migrate sites table: %w", err)
	}

	return nil
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

	_, err = r.pool.Exec(ctx, `
INSERT INTO sites (profile_code, domain, locale, settings)
VALUES ($1, $2, $3, $4::jsonb)
ON CONFLICT (domain) DO UPDATE SET
	profile_code = EXCLUDED.profile_code,
	locale = EXCLUDED.locale,
	settings = EXCLUDED.settings,
	updated_at = now();
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

	var site core.Site
	var settings []byte

	err := r.pool.QueryRow(ctx, `
SELECT id, profile_code, domain, locale, settings
FROM sites
WHERE domain = $1
LIMIT 1;
`, domain).Scan(
		&site.ID,
		&site.ProfileCode,
		&site.Domain,
		&site.Locale,
		&settings,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.Site{}, core.ErrSiteNotFound
	}
	if err != nil {
		return core.Site{}, fmt.Errorf("find site by domain %q: %w", domain, err)
	}

	if len(settings) == 0 {
		site.Settings = map[string]any{}
		return site, nil
	}

	if err := json.Unmarshal(settings, &site.Settings); err != nil {
		return core.Site{}, fmt.Errorf("unmarshal site settings: %w", err)
	}

	return site, nil
}

var _ core.SiteRepository = (*Repository)(nil)
