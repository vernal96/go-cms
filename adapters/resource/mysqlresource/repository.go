package mysqlresource

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

const resourceColumns = `
	id,
	site_id,
	parent_id,
	type,
	template,
	title,
	alias,
	path,
	sort,
	is_published,
	settings,
	seo
`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("mysql resource repository db is nil")
	}

	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) FindByID(
	ctx context.Context,
	id core.ResourceID,
) (core.Resource, error) {
	resource, err := scanResource(r.db.QueryRowContext(ctx, `
SELECT `+resourceColumns+`
FROM resources
WHERE id = ?
LIMIT 1;
`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Resource{}, core.ErrResourceNotFound
	}
	if err != nil {
		return core.Resource{}, fmt.Errorf("find resource by id %d: %w", id, err)
	}

	return resource, nil
}

func (r *Repository) FindByPath(
	ctx context.Context,
	siteID int64,
	path string,
) (core.Resource, error) {
	if siteID <= 0 {
		return core.Resource{}, errors.New("resource site id must be positive")
	}
	if path == "" {
		return core.Resource{}, errors.New("resource path is empty")
	}

	resource, err := scanResource(r.db.QueryRowContext(ctx, `
SELECT `+resourceColumns+`
FROM resources
WHERE site_id = ? AND path = ?
LIMIT 1;
`, siteID, path))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Resource{}, core.ErrResourceNotFound
	}
	if err != nil {
		return core.Resource{}, fmt.Errorf(
			"find resource by site %d and path %q: %w",
			siteID,
			path,
			err,
		)
	}

	return resource, nil
}

func (r *Repository) FindChildren(
	ctx context.Context,
	parentID core.ResourceID,
) ([]core.Resource, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+resourceColumns+`
FROM resources
WHERE parent_id = ?
ORDER BY sort, id;
`, parentID)
	if err != nil {
		return nil, fmt.Errorf("find resource children for %d: %w", parentID, err)
	}
	defer rows.Close()

	resources := make([]core.Resource, 0)
	for rows.Next() {
		resource, err := scanResource(rows)
		if err != nil {
			return nil, fmt.Errorf("scan resource child for %d: %w", parentID, err)
		}

		resources = append(resources, resource)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resource children for %d: %w", parentID, err)
	}

	return resources, nil
}

func (r *Repository) EnsureResource(
	ctx context.Context,
	resource core.Resource,
) (core.Resource, error) {
	if resource.SiteID <= 0 {
		return core.Resource{}, errors.New("resource site id must be positive")
	}
	if resource.Type == "" {
		return core.Resource{}, errors.New("resource type is empty")
	}
	if resource.Template == "" {
		return core.Resource{}, errors.New("resource template is empty")
	}
	if resource.Title == "" {
		return core.Resource{}, errors.New("resource title is empty")
	}
	if resource.Alias == "" {
		return core.Resource{}, errors.New("resource alias is empty")
	}
	if resource.Path == "" {
		return core.Resource{}, errors.New("resource path is empty")
	}
	if resource.Settings == nil {
		resource.Settings = map[string]any{}
	}
	if resource.SEO == nil {
		resource.SEO = map[string]any{}
	}

	settings, err := json.Marshal(resource.Settings)
	if err != nil {
		return core.Resource{}, fmt.Errorf("marshal resource settings: %w", err)
	}
	seo, err := json.Marshal(resource.SEO)
	if err != nil {
		return core.Resource{}, fmt.Errorf("marshal resource SEO: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO resources (
	site_id,
	parent_id,
	type,
	template,
	title,
	alias,
	path,
	sort,
	is_published,
	settings,
	seo
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	parent_id = VALUES(parent_id),
	type = VALUES(type),
	template = VALUES(template),
	title = VALUES(title),
	alias = VALUES(alias),
	sort = VALUES(sort),
	is_published = VALUES(is_published),
	settings = VALUES(settings),
	seo = VALUES(seo),
	updated_at = CURRENT_TIMESTAMP(6);
`,
		resource.SiteID,
		resourceParentID(resource.ParentID),
		resource.Type,
		resource.Template,
		resource.Title,
		resource.Alias,
		resource.Path,
		resource.Sort,
		resource.IsPublished,
		string(settings),
		string(seo),
	)
	if err != nil {
		return core.Resource{}, fmt.Errorf(
			"ensure resource for site %d and path %q: %w",
			resource.SiteID,
			resource.Path,
			err,
		)
	}

	return r.FindByPath(ctx, resource.SiteID, resource.Path)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanResource(row rowScanner) (core.Resource, error) {
	var resource core.Resource
	var parentID sql.NullInt64
	var settings []byte
	var seo []byte

	if err := row.Scan(
		&resource.ID,
		&resource.SiteID,
		&parentID,
		&resource.Type,
		&resource.Template,
		&resource.Title,
		&resource.Alias,
		&resource.Path,
		&resource.Sort,
		&resource.IsPublished,
		&settings,
		&seo,
	); err != nil {
		return core.Resource{}, err
	}

	if parentID.Valid {
		convertedParentID := core.ResourceID(parentID.Int64)
		resource.ParentID = &convertedParentID
	}

	if len(settings) > 0 {
		if err := json.Unmarshal(settings, &resource.Settings); err != nil {
			return core.Resource{}, fmt.Errorf("unmarshal resource settings: %w", err)
		}
	}
	if resource.Settings == nil {
		resource.Settings = map[string]any{}
	}

	if len(seo) > 0 {
		if err := json.Unmarshal(seo, &resource.SEO); err != nil {
			return core.Resource{}, fmt.Errorf("unmarshal resource SEO: %w", err)
		}
	}
	if resource.SEO == nil {
		resource.SEO = map[string]any{}
	}

	return resource, nil
}

func resourceParentID(parentID *core.ResourceID) any {
	if parentID == nil {
		return nil
	}

	return int64(*parentID)
}

var _ core.ResourceRepository = (*Repository)(nil)
