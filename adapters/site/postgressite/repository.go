package postgressite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vernal96/go-cms/adapters/database/postgresdb"
	"github.com/vernal96/go-cms/core"
)

type Repository struct {
	pool     *pgxpool.Pool
	ownsPool bool
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

	repository, err := NewRepository(pool)
	if err != nil {
		pool.Close()
		return nil, err
	}
	repository.ownsPool = true

	return repository, nil
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
	if r.ownsPool {
		r.pool.Close()
	}
}

func (r *Repository) Migrate(ctx context.Context) error {
	return postgresdb.Migrate(ctx, r.pool)
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
