package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres/medialock"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
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

func (r *Repository) Create(
	ctx context.Context,
	actorID *security.UserID,
	item resource.Resource,
	validate resource.ValidateImageMedia,
) (_ resource.Resource, resultErr error) {
	if ctx == nil {
		return resource.Resource{}, errors.New(
			"create resource context is nil",
		)
	}

	transaction, err := r.connector.Pool().BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"begin resource create: %w",
			err,
		)
	}
	defer func() {
		if resultErr == nil {
			return
		}
		rollbackErr := transaction.Rollback(context.Background())
		if rollbackErr != nil &&
			!errors.Is(rollbackErr, pgx.ErrTxClosed) {
			resultErr = errors.Join(resultErr, rollbackErr)
		}
	}()
	if _, err := transaction.Exec(ctx, `LOCK TABLE core.resources IN SHARE ROW EXCLUSIVE MODE;`); err != nil {
		return resource.Resource{}, fmt.Errorf("lock resources for create: %w", err)
	}
	if err := transaction.QueryRow(ctx, `
SELECT COALESCE(max(sort) + 1, 0)
FROM core.resources
WHERE site_id = $1
  AND parent_id IS NOT DISTINCT FROM $2::bigint;`, item.SiteID, item.ParentID).Scan(&item.Sort); err != nil {
		return resource.Resource{}, fmt.Errorf("resolve resource create position: %w", err)
	}

	if item.ImageMediaID != nil {
		if validate == nil {
			return resource.Resource{}, errors.New(
				"resource image media validator is nil",
			)
		}
		if err := medialock.Lock(
			ctx,
			transaction,
			*item.ImageMediaID,
		); err != nil {
			return resource.Resource{}, err
		}
		if err := ensureMediaAvailable(
			ctx,
			transaction,
			*item.ImageMediaID,
			0,
		); err != nil {
			return resource.Resource{}, err
		}
		if err := validate(ctx, *item.ImageMediaID); err != nil {
			return resource.Resource{}, err
		}
	}

	rawSettings, err := json.Marshal(item.Settings)
	if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"encode resource settings: %w",
			err,
		)
	}

	result, err := scanResource(transaction.QueryRow(ctx, `
INSERT INTO core.resources
(
    site_id,
    parent_id,
    type,
    template,
    content_type,
    title,
    menu_title,
	    slug,
	    path,
	    annotation,
	    content,
    image_media_id,
    target_resource_id,
    external_url,
    is_public,
    is_searchable,
    in_menu,
    in_sitemap,
    sort,
    published_at,
    unpublished_at,
    settings,
    created_by,
    updated_by
)
VALUES
(
	    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
	    $11, $12, $13, $14, $15, $16, $17, $18, $19,
	    $20, $21, $22::jsonb, $23, $23
)
RETURNING
    id, site_id, parent_id, type, template, content_type,
	    title, menu_title, slug, path, annotation, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
	    updated_at, created_by, updated_by, deleted_at, deleted_by;
`,
		item.SiteID,
		item.ParentID,
		item.Type,
		item.Template,
		item.ContentType,
		item.Title,
		item.MenuTitle,
		item.Slug,
		item.Path,
		item.Annotation,
		item.Content,
		item.ImageMediaID,
		item.TargetResourceID,
		item.ExternalURL,
		item.IsPublic,
		item.IsSearchable,
		item.InMenu,
		item.InSitemap,
		item.Sort,
		item.PublishedAt,
		item.UnpublishedAt,
		string(rawSettings),
		actorID,
	))
	if err != nil {
		return resource.Resource{}, translateError(err)
	}
	if err := replaceFileReferences(ctx, transaction, result.ID, item.FileReferences); err != nil {
		return resource.Resource{}, err
	}
	result.FileReferences = cloneFileReferences(item.FileReferences)

	if err := transaction.Commit(ctx); err != nil {
		return resource.Resource{}, translateError(err)
	}
	return result, nil
}

func (r *Repository) ByID(
	ctx context.Context,
	id resource.ID,
) (resource.Resource, error) {
	if ctx == nil {
		return resource.Resource{}, errors.New(
			"get resource context is nil",
		)
	}

	result, err := scanResource(r.connector.Pool().QueryRow(ctx, `
SELECT
    id, site_id, parent_id, type, template, content_type,
	    title, menu_title, slug, path, annotation, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
	    updated_at, created_by, updated_by, deleted_at, deleted_by
FROM core.resources
WHERE id = $1;
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.Resource{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"query core resource %d: %w",
			id,
			err,
		)
	}
	items := []resource.Resource{result}
	if err := loadResourceWidgets(
		ctx,
		r.connector.Pool(),
		items,
	); err != nil {
		return resource.Resource{}, err
	}

	return items[0], nil
}

func (r *Repository) ByPath(
	ctx context.Context,
	siteID site.ID,
	path string,
) (resource.Resource, error) {
	if ctx == nil {
		return resource.Resource{}, errors.New(
			"get resource by path context is nil",
		)
	}

	result, err := scanResource(r.connector.Pool().QueryRow(ctx, `
SELECT
    id, site_id, parent_id, type, template, content_type,
	    title, menu_title, slug, path, annotation, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
	    updated_at, created_by, updated_by, deleted_at, deleted_by
FROM core.resources
WHERE site_id = $1
  AND path = $2;
`, siteID, path))
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.Resource{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"query core resource by path %q: %w",
			path,
			err,
		)
	}
	items := []resource.Resource{result}
	if err := loadResourceWidgets(
		ctx,
		r.connector.Pool(),
		items,
	); err != nil {
		return resource.Resource{}, err
	}

	return items[0], nil
}

func (r *Repository) ListBySite(
	ctx context.Context,
	siteID site.ID,
) ([]resource.Resource, error) {
	if ctx == nil {
		return nil, errors.New("list resources context is nil")
	}

	rows, err := r.connector.Pool().Query(ctx, `
SELECT
    id, site_id, parent_id, type, template, content_type,
	    title, menu_title, slug, path, annotation, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
	    updated_at, created_by, updated_by, deleted_at, deleted_by
FROM core.resources
WHERE site_id = $1
ORDER BY parent_id NULLS FIRST, sort, id;
`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query core resources: %w", err)
	}
	defer rows.Close()

	result := make([]resource.Resource, 0)
	for rows.Next() {
		item, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate core resources: %w", err)
	}
	rows.Close()
	if err := loadResourceWidgets(
		ctx,
		r.connector.Pool(),
		result,
	); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) ListChildren(
	ctx context.Context,
	siteID site.ID,
	parentID *resource.ID,
) ([]resource.Child, error) {
	if ctx == nil {
		return nil, errors.New("list resource children context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, `
WITH valid_parent AS (
    SELECT true AS ok
    WHERE $2::bigint IS NULL OR EXISTS (
        SELECT 1
        FROM core.resources parent
        WHERE parent.site_id = $1
          AND parent.id = $2
    )
), children AS (
SELECT
    current.id,
    current.site_id,
    current.parent_id,
    current.type,
    current.template,
    current.title,
    current.menu_title,
	current.is_public,
	current.published_at,
	current.unpublished_at,
	current.deleted_at,
    EXISTS (
        SELECT 1
        FROM core.resources child
        WHERE child.site_id = current.site_id
          AND child.parent_id = current.id
    ) AS has_children,
    current.sort
FROM core.resources current, valid_parent
WHERE current.site_id = $1
  AND current.parent_id IS NOT DISTINCT FROM $2::bigint
)
SELECT
    EXISTS (SELECT 1 FROM valid_parent) AS parent_exists,
    children.id,
    children.site_id,
    children.parent_id,
    children.type,
    children.template,
    children.title,
    children.menu_title,
	children.is_public,
	children.published_at,
	children.unpublished_at,
	children.deleted_at,
	children.sort,
    children.has_children
FROM (SELECT true) marker
LEFT JOIN children ON true
ORDER BY children.sort, children.id;`, siteID, parentID)
	if err != nil {
		return nil, fmt.Errorf("query resource children: %w", err)
	}
	defer rows.Close()

	items := make([]resource.Child, 0)
	for rows.Next() {
		var (
			parentExists     bool
			rawID            *int64
			rawSiteID        *int64
			rawParent        *int64
			rawType          *string
			rawTemplate      *string
			rawTitle         *string
			rawMenuTitle     *string
			rawIsPublic      *bool
			rawPublishedAt   *time.Time
			rawUnpublishedAt *time.Time
			rawDeletedAt     *time.Time
			rawSort          *int
			rawHasChildren   *bool
		)
		if err := rows.Scan(
			&parentExists,
			&rawID,
			&rawSiteID,
			&rawParent,
			&rawType,
			&rawTemplate,
			&rawTitle,
			&rawMenuTitle,
			&rawIsPublic,
			&rawPublishedAt,
			&rawUnpublishedAt,
			&rawDeletedAt,
			&rawSort,
			&rawHasChildren,
		); err != nil {
			return nil, fmt.Errorf("scan resource child: %w", err)
		}
		if !parentExists {
			return nil, resource.ErrNotFound
		}
		if rawID == nil {
			continue
		}
		item := resource.Child{
			ID:            resource.ID(*rawID),
			SiteID:        site.ID(*rawSiteID),
			Type:          resourcetype.Code(*rawType),
			Title:         *rawTitle,
			MenuTitle:     *rawMenuTitle,
			Sort:          *rawSort,
			IsPublic:      rawIsPublic != nil && *rawIsPublic,
			PublishedAt:   rawPublishedAt,
			UnpublishedAt: rawUnpublishedAt,
			DeletedAt:     rawDeletedAt,
			HasChildren:   *rawHasChildren,
		}
		if rawParent != nil {
			value := resource.ID(*rawParent)
			item.ParentID = &value
		}
		if rawTemplate != nil {
			value := template.Code(*rawTemplate)
			item.Template = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resource children: %w", err)
	}
	return items, nil
}

func (r *Repository) Statistics(
	ctx context.Context,
	query resource.StatisticsQuery,
) (resource.Statistics, error) {
	if ctx == nil {
		return resource.Statistics{}, errors.New("resource statistics context is nil")
	}
	allowed := make([]int64, len(query.Scope.SiteIDs))
	for index, id := range query.Scope.SiteIDs {
		allowed[index] = int64(id)
	}
	breakdown := make([]int64, len(query.SiteIDs))
	for index, id := range query.SiteIDs {
		breakdown[index] = int64(id)
	}

	result := resource.Statistics{BySite: make(map[site.ID]int, len(breakdown))}
	if err := r.connector.Pool().QueryRow(ctx, `
SELECT count(*)
FROM core.resources
WHERE deleted_at IS NULL
  AND ($1 OR site_id = ANY($2::bigint[]));`, query.Scope.All, allowed).Scan(&result.Total); err != nil {
		return resource.Statistics{}, fmt.Errorf("count core resource statistics: %w", err)
	}
	if len(breakdown) == 0 {
		return result, nil
	}

	rows, err := r.connector.Pool().Query(ctx, `
SELECT site_id, count(*)
FROM core.resources
WHERE site_id = ANY($1::bigint[])
  AND deleted_at IS NULL
  AND ($2 OR site_id = ANY($3::bigint[]))
GROUP BY site_id;`, breakdown, query.Scope.All, allowed)
	if err != nil {
		return resource.Statistics{}, fmt.Errorf("query core resource statistics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var siteID site.ID
		var count int
		if err := rows.Scan(&siteID, &count); err != nil {
			return resource.Statistics{}, fmt.Errorf("scan core resource statistics: %w", err)
		}
		result.BySite[siteID] = count
	}
	if err := rows.Err(); err != nil {
		return resource.Statistics{}, fmt.Errorf("iterate core resource statistics: %w", err)
	}
	return result, nil
}

func (r *Repository) ExistsInSite(
	ctx context.Context,
	siteID site.ID,
	id resource.ID,
) (bool, error) {
	if ctx == nil {
		return false, errors.New("check site resource context is nil")
	}
	var exists bool
	if err := r.connector.Pool().QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM core.resources WHERE site_id = $1 AND id = $2
);`, siteID, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check site resource: %w", err)
	}
	return exists, nil
}

func (r *Repository) Update(
	ctx context.Context,
	actorID *security.UserID,
	current resource.Resource,
	item resource.Resource,
	validate resource.ValidateImageMedia,
) (_ resource.Resource, resultErr error) {
	if ctx == nil {
		return resource.Resource{}, errors.New(
			"update resource context is nil",
		)
	}

	transaction, err := r.connector.Pool().BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"begin resource update: %w",
			err,
		)
	}
	defer func() {
		if resultErr == nil {
			return
		}

		rollbackErr := transaction.Rollback(context.Background())
		if rollbackErr != nil &&
			!errors.Is(rollbackErr, pgx.ErrTxClosed) {
			resultErr = errors.Join(resultErr, rollbackErr)
		}
	}()

	if current.ID != item.ID {
		return resource.Resource{}, resource.ErrInvalidReference
	}
	if _, err := transaction.Exec(ctx, `LOCK TABLE core.resources IN SHARE ROW EXCLUSIVE MODE;`); err != nil {
		return resource.Resource{}, fmt.Errorf("lock resources for update: %w", err)
	}

	mediaIDs := make([]media.ID, 0, 2)
	if current.ImageMediaID != nil {
		mediaIDs = append(mediaIDs, *current.ImageMediaID)
	}
	if item.ImageMediaID != nil {
		mediaIDs = append(mediaIDs, *item.ImageMediaID)
	}
	if err := medialock.Lock(ctx, transaction, mediaIDs...); err != nil {
		return resource.Resource{}, err
	}

	var (
		currentSiteID       site.ID
		currentImageMediaID *int64
		currentParentID     *int64
		currentSort         int
		currentDeletedAt    *time.Time
	)
	if err := transaction.QueryRow(ctx, `
SELECT site_id, image_media_id, parent_id, sort, deleted_at
FROM core.resources
WHERE id = $1
FOR UPDATE;
`, item.ID).Scan(
		&currentSiteID,
		&currentImageMediaID,
		&currentParentID,
		&currentSort,
		&currentDeletedAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return resource.Resource{}, resource.ErrNotFound
	} else if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"lock resource %d: %w",
			item.ID,
			err,
		)
	}
	if currentSiteID != item.SiteID {
		return resource.Resource{}, resource.ErrInvalidReference
	}
	if !equalMediaID(current.ImageMediaID, currentImageMediaID) {
		return resource.Resource{}, resource.ErrConflict
	}
	lockedParentID := resourceIDFromInt64(currentParentID)
	if currentDeletedAt != nil && (!sameResourceID(lockedParentID, item.ParentID) || currentSort != item.Sort) {
		return resource.Resource{}, resource.ErrInvalidTree
	}

	if item.ImageMediaID != nil {
		if validate == nil {
			return resource.Resource{}, errors.New(
				"resource image media validator is nil",
			)
		}
		if err := ensureMediaAvailable(
			ctx,
			transaction,
			*item.ImageMediaID,
			item.ID,
		); err != nil {
			return resource.Resource{}, err
		}
		if err := validate(ctx, *item.ImageMediaID); err != nil {
			return resource.Resource{}, err
		}
	}

	var parent *resource.Resource
	if item.ParentID != nil {
		parentItem, err := lockResource(
			ctx,
			transaction,
			*item.ParentID,
		)
		if err != nil {
			return resource.Resource{}, err
		}
		if parentItem.SiteID != item.SiteID {
			return resource.Resource{}, resource.ErrInvalidReference
		}
		if parentItem.DeletedAt != nil && currentDeletedAt == nil {
			return resource.Resource{}, resource.ErrInvalidTree
		}
		parent = &parentItem

		var cycle bool
		if err := transaction.QueryRow(ctx, `
WITH RECURSIVE ancestors AS
(
    SELECT id, parent_id
    FROM core.resources
    WHERE id = $1

    UNION ALL

    SELECT resource.id, resource.parent_id
    FROM core.resources AS resource
    JOIN ancestors
      ON resource.id = ancestors.parent_id
)
SELECT EXISTS
(
    SELECT 1
    FROM ancestors
    WHERE id = $2
);
`, *item.ParentID, item.ID).Scan(&cycle); err != nil {
			return resource.Resource{}, fmt.Errorf(
				"check resource parent cycle: %w",
				err,
			)
		}
		if cycle {
			return resource.Resource{}, resource.ErrInvalidTree
		}
	}

	if item.Path != nil {
		item.Path, err = resource.BuildPath(parent, item.Slug)
		if err != nil {
			return resource.Resource{}, err
		}
	}
	item.Sort, err = reorderSiblings(
		ctx, transaction, item.SiteID, item.ID, lockedParentID, item.ParentID, item.Sort,
	)
	if err != nil {
		return resource.Resource{}, err
	}

	rawSettings, err := json.Marshal(item.Settings)
	if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"encode resource settings: %w",
			err,
		)
	}

	updated, err := scanResource(transaction.QueryRow(ctx, `
UPDATE core.resources
SET
    parent_id = $2,
    type = $3,
    template = $4,
    content_type = $5,
    title = $6,
    menu_title = $7,
	    slug = $8,
	    path = $9,
	    annotation = $10,
	    content = $11,
	    image_media_id = $12,
	    target_resource_id = $13,
	    external_url = $14,
	    is_public = $15,
	    is_searchable = $16,
	    in_menu = $17,
	    in_sitemap = $18,
	    sort = $19,
	    published_at = $20,
	    unpublished_at = $21,
	    settings = $22::jsonb,
	    updated_at = now(),
	    updated_by = $23
WHERE id = $1
RETURNING
    id, site_id, parent_id, type, template, content_type,
	    title, menu_title, slug, path, annotation, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
	    updated_at, created_by, updated_by, deleted_at, deleted_by;
`,
		item.ID,
		item.ParentID,
		item.Type,
		item.Template,
		item.ContentType,
		item.Title,
		item.MenuTitle,
		item.Slug,
		item.Path,
		item.Annotation,
		item.Content,
		item.ImageMediaID,
		item.TargetResourceID,
		item.ExternalURL,
		item.IsPublic,
		item.IsSearchable,
		item.InMenu,
		item.InSitemap,
		item.Sort,
		item.PublishedAt,
		item.UnpublishedAt,
		string(rawSettings),
		actorID,
	))
	if err != nil {
		return resource.Resource{}, translateError(err)
	}

	if _, err := transaction.Exec(ctx, `
WITH RECURSIVE tree AS
(
    SELECT id, path
    FROM core.resources
    WHERE id = $1

    UNION ALL

    SELECT
        child.id,
        CASE
            WHEN child.path IS NULL THEN NULL
            WHEN tree.path IS NULL THEN NULL
            WHEN tree.path = '/' THEN '/' || child.slug
            ELSE tree.path || '/' || child.slug
        END
    FROM core.resources AS child
    JOIN tree
      ON child.parent_id = tree.id
)
UPDATE core.resources AS item
SET
    path = tree.path,
    updated_at = now(),
    updated_by = $2
FROM tree
WHERE item.id = tree.id
  AND item.id <> $1;
`, item.ID, actorID); err != nil {
		return resource.Resource{}, translateError(err)
	}

	updated.Widgets = widget.CloneBindings(item.Widgets)
	if err := replaceFileReferences(ctx, transaction, updated.ID, item.FileReferences); err != nil {
		return resource.Resource{}, err
	}
	updated.FileReferences = cloneFileReferences(item.FileReferences)

	if !sameMediaID(current.ImageMediaID, item.ImageMediaID) &&
		current.ImageMediaID != nil {
		if _, err := transaction.Exec(ctx, `
DELETE FROM core.media
WHERE id = $1;
`, *current.ImageMediaID); err != nil {
			return resource.Resource{}, translateError(err)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return resource.Resource{}, translateError(err)
	}
	return updated, nil
}

func (r *Repository) SoftDelete(
	ctx context.Context,
	actorID *security.UserID,
	id resource.ID,
) (resultErr error) {
	if ctx == nil {
		return errors.New("soft delete resource context is nil")
	}
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin resource soft delete: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err := tx.Exec(ctx, `LOCK TABLE core.resources IN SHARE ROW EXCLUSIVE MODE;`); err != nil {
		return fmt.Errorf("lock resources for soft delete: %w", err)
	}
	command, err := tx.Exec(ctx, `
WITH RECURSIVE tree AS (
    SELECT id FROM core.resources WHERE id = $1
    UNION ALL
    SELECT child.id
    FROM core.resources child
    JOIN tree parent ON child.parent_id = parent.id
)
UPDATE core.resources item
SET deleted_at = now(), deleted_by = $2, updated_at = now(), updated_by = $2
WHERE item.id IN (SELECT id FROM tree);`, id, actorID)
	if err != nil {
		return translateError(err)
	}
	if command.RowsAffected() == 0 {
		return resource.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return translateError(err)
	}
	return nil
}

func (r *Repository) Restore(
	ctx context.Context,
	actorID *security.UserID,
	id resource.ID,
	withDescendants bool,
) (resultErr error) {
	if ctx == nil {
		return errors.New("restore resource context is nil")
	}
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin resource restore: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err := tx.Exec(ctx, `LOCK TABLE core.resources IN SHARE ROW EXCLUSIVE MODE;`); err != nil {
		return fmt.Errorf("lock resources for restore: %w", err)
	}
	var parentDeleted bool
	if err := tx.QueryRow(ctx, `
SELECT COALESCE(parent.deleted_at IS NOT NULL, false)
FROM core.resources item
LEFT JOIN core.resources parent ON parent.id = item.parent_id
WHERE item.id = $1;`, id).Scan(&parentDeleted); errors.Is(err, pgx.ErrNoRows) {
		return resource.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("check resource restore parent: %w", err)
	}
	if parentDeleted {
		return resource.ErrInvalidTree
	}
	if withDescendants {
		if _, err := tx.Exec(ctx, `
WITH RECURSIVE tree AS (
    SELECT id FROM core.resources WHERE id = $1
    UNION ALL
    SELECT child.id FROM core.resources child
    JOIN tree parent ON child.parent_id = parent.id
)
UPDATE core.resources item
SET deleted_at = NULL, deleted_by = NULL, updated_at = now(), updated_by = $2
WHERE item.id IN (SELECT id FROM tree);`, id, actorID); err != nil {
			return translateError(err)
		}
	} else if command, err := tx.Exec(ctx, `
UPDATE core.resources
SET deleted_at = NULL, deleted_by = NULL, updated_at = now(), updated_by = $2
WHERE id = $1;`, id, actorID); err != nil {
		return translateError(err)
	} else if command.RowsAffected() == 0 {
		return resource.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return translateError(err)
	}
	return nil
}

func (r *Repository) Delete(
	ctx context.Context,
	id resource.ID,
) (_ error) {
	if ctx == nil {
		return errors.New("delete resource context is nil")
	}

	transaction, err := r.connector.Pool().BeginTx(
		ctx,
		pgx.TxOptions{},
	)
	if err != nil {
		return fmt.Errorf("begin resource delete: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = transaction.Rollback(context.Background())
	}()
	if _, err := transaction.Exec(ctx, `LOCK TABLE core.resources IN SHARE ROW EXCLUSIVE MODE;`); err != nil {
		return fmt.Errorf("lock resources for permanent delete: %w", err)
	}
	var deletedSiteID site.ID
	var deletedParentID *int64
	if err := transaction.QueryRow(ctx, `SELECT site_id, parent_id FROM core.resources WHERE id = $1;`, id).Scan(
		&deletedSiteID, &deletedParentID,
	); errors.Is(err, pgx.ErrNoRows) {
		return resource.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("read resource delete position: %w", err)
	}

	observedMediaIDs, exists, err := treeMediaIDs(
		ctx,
		transaction,
		id,
		false,
	)
	if err != nil {
		return err
	}
	if !exists {
		return resource.ErrNotFound
	}
	if err := medialock.Lock(
		ctx,
		transaction,
		observedMediaIDs...,
	); err != nil {
		return err
	}

	actualMediaIDs, exists, err := treeMediaIDs(
		ctx,
		transaction,
		id,
		true,
	)
	if err != nil {
		return err
	}
	if !exists {
		return resource.ErrNotFound
	}
	if !mediaIDsContained(actualMediaIDs, observedMediaIDs) {
		return resource.ErrConflict
	}
	if _, err := transaction.Exec(ctx, `
WITH RECURSIVE tree AS
(
    SELECT id FROM core.resources WHERE id = $1
    UNION ALL
    SELECT child.id FROM core.resources AS child
    JOIN tree AS parent ON child.parent_id = parent.id
)
DELETE FROM core.file_field_references
WHERE owner_kind = 'resource' AND owner_id IN (SELECT id FROM tree);
`, id); err != nil {
		return translateDeleteError(err)
	}

	commandTag, err := transaction.Exec(ctx, `
DELETE FROM core.resources
WHERE id = $1;
`, id)
	if err != nil {
		return translateDeleteError(err)
	}
	if commandTag.RowsAffected() == 0 {
		return resource.ErrNotFound
	}
	if _, err := transaction.Exec(ctx, `
WITH ordered AS (
    SELECT id, row_number() OVER (ORDER BY sort, id) - 1 AS new_sort
    FROM core.resources
    WHERE site_id = $1 AND parent_id IS NOT DISTINCT FROM $2::bigint
)
UPDATE core.resources item
SET sort = ordered.new_sort
FROM ordered
WHERE item.id = ordered.id AND item.sort <> ordered.new_sort;`, deletedSiteID, deletedParentID); err != nil {
		return translateDeleteError(err)
	}

	for _, mediaID := range actualMediaIDs {
		if _, err := transaction.Exec(ctx, `
DELETE FROM core.media
WHERE id = $1;
`, mediaID); err != nil {
			return translateDeleteError(err)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return translateDeleteError(err)
	}
	committed = true
	return nil
}

func reorderSiblings(
	ctx context.Context,
	tx pgx.Tx,
	siteID site.ID,
	movingID resource.ID,
	sourceParentID *resource.ID,
	targetParentID *resource.ID,
	position int,
) (int, error) {
	if position < 0 {
		return 0, resource.ErrInvalidTree
	}
	target, err := siblingIDs(ctx, tx, siteID, targetParentID, movingID)
	if err != nil {
		return 0, err
	}
	if position > len(target) {
		position = len(target)
	}
	if sameResourceID(sourceParentID, targetParentID) {
		ordered := append(target, 0)
		copy(ordered[position+1:], ordered[position:])
		ordered[position] = movingID
		if err := updateSiblingPositions(ctx, tx, ordered, movingID); err != nil {
			return 0, err
		}
		return position, nil
	}
	source, err := siblingIDs(ctx, tx, siteID, sourceParentID, movingID)
	if err != nil {
		return 0, err
	}
	if err := updateSiblingPositions(ctx, tx, source, 0); err != nil {
		return 0, err
	}
	ordered := append(target, 0)
	copy(ordered[position+1:], ordered[position:])
	ordered[position] = movingID
	if err := updateSiblingPositions(ctx, tx, ordered, movingID); err != nil {
		return 0, err
	}
	return position, nil
}

func siblingIDs(
	ctx context.Context,
	tx pgx.Tx,
	siteID site.ID,
	parentID *resource.ID,
	exclude resource.ID,
) ([]resource.ID, error) {
	rows, err := tx.Query(ctx, `
SELECT id
FROM core.resources
WHERE site_id = $1
  AND parent_id IS NOT DISTINCT FROM $2::bigint
  AND id <> $3
ORDER BY sort, id;`, siteID, parentID, exclude)
	if err != nil {
		return nil, fmt.Errorf("list resource siblings: %w", err)
	}
	defer rows.Close()
	result := make([]resource.ID, 0)
	for rows.Next() {
		var id resource.ID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan resource sibling: %w", err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resource siblings: %w", err)
	}
	return result, nil
}

func updateSiblingPositions(ctx context.Context, tx pgx.Tx, ids []resource.ID, skip resource.ID) error {
	for position, id := range ids {
		if id == skip {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE core.resources SET sort = $2 WHERE id = $1;`, id, position); err != nil {
			return fmt.Errorf("update resource sibling position: %w", err)
		}
	}
	return nil
}

func resourceIDFromInt64(value *int64) *resource.ID {
	if value == nil {
		return nil
	}
	result := resource.ID(*value)
	return &result
}

func sameResourceID(left, right *resource.ID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

type rowScanner interface {
	Scan(...any) error
}

type rowQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (r *Repository) CreateWidget(
	ctx context.Context,
	resourceID resource.ID,
	binding widget.Binding,
) (widget.Binding, error) {
	if ctx == nil || resourceID <= 0 {
		return widget.Binding{}, errors.New("resource widget create input is invalid")
	}
	rawParams, err := encodeWidgetParams(binding)
	if err != nil {
		return widget.Binding{}, err
	}
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return widget.Binding{}, translateError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := lockWidgetResource(ctx, tx, resourceID); err != nil {
		return widget.Binding{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM core.resource_widgets WHERE resource_id = $1 AND area = $2;`, resourceID, binding.Area).Scan(&count); err != nil {
		return widget.Binding{}, translateError(err)
	}
	if binding.Position != count {
		return widget.Binding{}, fmt.Errorf("resource %d widget position %d does not append to %q at %d", resourceID, binding.Position, binding.Area, count)
	}
	created, err := scanWidget(tx.QueryRow(ctx, `
INSERT INTO core.resource_widgets
    (resource_id, widget_code, area, position, view, columns, margin_top, margin_bottom, enabled, params)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
RETURNING id, widget_code, area, position, view, columns, margin_top, margin_bottom, enabled, params;
`, resourceID, binding.Code, binding.Area, binding.Position, binding.Presentation.View,
		binding.Presentation.Columns, binding.Presentation.MarginTop, binding.Presentation.MarginBottom,
		binding.Presentation.Enabled, string(rawParams)))
	if err != nil {
		return widget.Binding{}, translateError(err)
	}
	if err := touchWidgetResource(ctx, tx, resourceID); err != nil {
		return widget.Binding{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return widget.Binding{}, translateError(err)
	}
	return created, nil
}

func (r *Repository) UpdateWidget(
	ctx context.Context,
	resourceID resource.ID,
	binding widget.Binding,
) (widget.Binding, error) {
	if ctx == nil || resourceID <= 0 || binding.ID <= 0 {
		return widget.Binding{}, errors.New("resource widget update input is invalid")
	}
	rawParams, err := encodeWidgetParams(binding)
	if err != nil {
		return widget.Binding{}, err
	}
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return widget.Binding{}, translateError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := lockWidgetResource(ctx, tx, resourceID); err != nil {
		return widget.Binding{}, err
	}
	updated, err := scanWidget(tx.QueryRow(ctx, `
UPDATE core.resource_widgets
SET widget_code = $3, area = $4, position = $5, view = $6, columns = $7,
    margin_top = $8, margin_bottom = $9, enabled = $10, params = $11::jsonb
WHERE resource_id = $1 AND id = $2
RETURNING id, widget_code, area, position, view, columns, margin_top, margin_bottom, enabled, params;
`, resourceID, binding.ID, binding.Code, binding.Area, binding.Position,
		binding.Presentation.View, binding.Presentation.Columns, binding.Presentation.MarginTop,
		binding.Presentation.MarginBottom, binding.Presentation.Enabled, string(rawParams)))
	if err != nil {
		return widget.Binding{}, translateError(err)
	}
	if err := touchWidgetResource(ctx, tx, resourceID); err != nil {
		return widget.Binding{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return widget.Binding{}, translateError(err)
	}
	return updated, nil
}

func (r *Repository) DeleteWidget(
	ctx context.Context,
	resourceID resource.ID,
	bindingID widget.BindingID,
) error {
	if ctx == nil || resourceID <= 0 || bindingID <= 0 {
		return errors.New("resource widget delete input is invalid")
	}
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return translateError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := lockWidgetResource(ctx, tx, resourceID); err != nil {
		return err
	}
	var area widget.AreaCode
	var position int
	if err := tx.QueryRow(ctx, `SELECT area, position FROM core.resource_widgets WHERE resource_id = $1 AND id = $2 FOR UPDATE;`, resourceID, bindingID).Scan(&area, &position); err != nil {
		return translateError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.resource_widgets WHERE resource_id = $1 AND id = $2;`, resourceID, bindingID); err != nil {
		return translateError(err)
	}
	var offset int
	if err := tx.QueryRow(ctx, `SELECT coalesce(max(position), -1) + 1 FROM core.resource_widgets WHERE resource_id = $1 AND area = $2;`, resourceID, area).Scan(&offset); err != nil {
		return translateError(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE core.resource_widgets SET position = position + $4 WHERE resource_id = $1 AND area = $2 AND position > $3;`, resourceID, area, position, offset); err != nil {
		return translateError(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE core.resource_widgets SET position = position - $4::integer - 1 WHERE resource_id = $1 AND area = $2 AND position > $3::integer + $4::integer;`, resourceID, area, position, offset); err != nil {
		return translateError(err)
	}
	if err := touchWidgetResource(ctx, tx, resourceID); err != nil {
		return err
	}
	return translateError(tx.Commit(ctx))
}

func (r *Repository) ReorderWidgets(
	ctx context.Context,
	resourceID resource.ID,
	order []widget.Order,
) ([]widget.Binding, error) {
	if ctx == nil || resourceID <= 0 {
		return nil, errors.New("resource widget reorder input is invalid")
	}
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return nil, translateError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := lockWidgetResource(ctx, tx, resourceID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id FROM core.resource_widgets WHERE resource_id = $1 FOR UPDATE;`, resourceID)
	if err != nil {
		return nil, translateError(err)
	}
	known := make(map[widget.BindingID]struct{})
	for rows.Next() {
		var id widget.BindingID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, translateError(err)
		}
		known[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, translateError(err)
	}
	if len(known) != len(order) {
		return nil, errors.New("resource widget order is incomplete")
	}
	positions := map[widget.AreaCode]int{
		widget.AreaBody:    0,
		widget.AreaSidebar: 0,
	}
	seen := make(map[widget.BindingID]struct{}, len(order))
	for _, item := range order {
		if _, exists := known[item.ID]; !exists {
			return nil, fmt.Errorf("resource widget %d is unavailable", item.ID)
		}
		if _, exists := seen[item.ID]; exists {
			return nil, fmt.Errorf("resource widget %d is duplicated in order", item.ID)
		}
		seen[item.ID] = struct{}{}
		if !widget.ValidArea(item.Area) {
			return nil, fmt.Errorf("resource widget %d has invalid area %q", item.ID, item.Area)
		}
		if item.Position != positions[item.Area] {
			return nil, fmt.Errorf(
				"resource widget %d in %q has position %d instead of %d",
				item.ID,
				item.Area,
				item.Position,
				positions[item.Area],
			)
		}
		positions[item.Area]++
	}
	var offset int
	if err := tx.QueryRow(ctx, `SELECT coalesce(max(position), -1) + 1 FROM core.resource_widgets WHERE resource_id = $1;`, resourceID).Scan(&offset); err != nil {
		return nil, translateError(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE core.resource_widgets SET position = position + $2 WHERE resource_id = $1;`, resourceID, offset); err != nil {
		return nil, translateError(err)
	}
	for _, item := range order {
		command, err := tx.Exec(ctx, `UPDATE core.resource_widgets SET area = $3, position = $4 WHERE resource_id = $1 AND id = $2;`, resourceID, item.ID, item.Area, item.Position)
		if err != nil {
			return nil, translateError(err)
		}
		if command.RowsAffected() != 1 {
			return nil, resource.ErrNotFound
		}
	}
	loaded := []resource.Resource{{ID: resourceID}}
	if err := loadResourceWidgets(ctx, tx, loaded); err != nil {
		return nil, err
	}
	if err := touchWidgetResource(ctx, tx, resourceID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, translateError(err)
	}
	return loaded[0].Widgets, nil
}

func lockWidgetResource(ctx context.Context, tx pgx.Tx, resourceID resource.ID) error {
	var id resource.ID
	if err := tx.QueryRow(ctx, `SELECT id FROM core.resources WHERE id = $1 FOR UPDATE;`, resourceID).Scan(&id); err != nil {
		return translateError(err)
	}
	return nil
}

func touchWidgetResource(ctx context.Context, tx pgx.Tx, resourceID resource.ID) error {
	if _, err := tx.Exec(ctx, `UPDATE core.resources SET updated_at = now() WHERE id = $1;`, resourceID); err != nil {
		return translateError(err)
	}
	return nil
}

func encodeWidgetParams(binding widget.Binding) ([]byte, error) {
	params := binding.Params
	if params == nil {
		params = map[string]any{}
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode params for widget %q: %w", binding.Code, err)
	}
	return raw, nil
}

func scanWidget(scanner rowScanner) (widget.Binding, error) {
	var binding widget.Binding
	var rawParams []byte
	if err := scanner.Scan(
		&binding.ID, &binding.Code, &binding.Area, &binding.Position,
		&binding.Presentation.View, &binding.Presentation.Columns,
		&binding.Presentation.MarginTop, &binding.Presentation.MarginBottom,
		&binding.Presentation.Enabled, &rawParams,
	); err != nil {
		return widget.Binding{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(rawParams))
	decoder.UseNumber()
	if err := decoder.Decode(&binding.Params); err != nil {
		return widget.Binding{}, fmt.Errorf("decode params for widget %d: %w", binding.ID, err)
	}
	if binding.Params == nil {
		binding.Params = map[string]any{}
	}
	return binding, nil
}

func loadResourceWidgets(
	ctx context.Context,
	queryer rowQueryer,
	items []resource.Resource,
) error {
	if len(items) == 0 {
		return nil
	}

	indexes := make(map[resource.ID]int, len(items))
	ids := make([]int64, len(items))
	for index := range items {
		indexes[items[index].ID] = index
		ids[index] = int64(items[index].ID)
		items[index].Widgets = nil
	}

	rows, err := queryer.Query(ctx, `
SELECT resource_id, id, widget_code, area, position, view, columns,
       margin_top, margin_bottom, enabled, params
FROM core.resource_widgets
WHERE resource_id = ANY($1::bigint[])
ORDER BY resource_id, area, position, id;
`, ids)
	if err != nil {
		return fmt.Errorf("query resource widgets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			resourceID resource.ID
			binding    widget.Binding
			rawParams  []byte
		)
		if err := rows.Scan(
			&resourceID,
			&binding.ID,
			&binding.Code,
			&binding.Area,
			&binding.Position,
			&binding.Presentation.View,
			&binding.Presentation.Columns,
			&binding.Presentation.MarginTop,
			&binding.Presentation.MarginBottom,
			&binding.Presentation.Enabled,
			&rawParams,
		); err != nil {
			return fmt.Errorf("scan resource widget: %w", err)
		}

		index, exists := indexes[resourceID]
		if !exists {
			return fmt.Errorf(
				"resource widget references unexpected resource %d",
				resourceID,
			)
		}
		params := make(map[string]any)
		decoder := json.NewDecoder(bytes.NewReader(rawParams))
		decoder.UseNumber()
		if err := decoder.Decode(&params); err != nil {
			return fmt.Errorf(
				"decode params for resource %d widget %q: %w",
				resourceID,
				binding.Code,
				err,
			)
		}

		binding.Params = params
		items[index].Widgets = append(items[index].Widgets, binding)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate resource widgets: %w", err)
	}

	return nil
}

func scanResource(scanner rowScanner) (resource.Resource, error) {
	var (
		item             resource.Resource
		parentID         *int64
		templateCode     *string
		contentType      *string
		path             *string
		imageMediaID     *int64
		targetResourceID *int64
		externalURL      *string
		rawSettings      []byte
	)

	if err := scanner.Scan(
		&item.ID,
		&item.SiteID,
		&parentID,
		&item.Type,
		&templateCode,
		&contentType,
		&item.Title,
		&item.MenuTitle,
		&item.Slug,
		&path,
		&item.Annotation,
		&item.Content,
		&imageMediaID,
		&targetResourceID,
		&externalURL,
		&item.IsPublic,
		&item.IsSearchable,
		&item.InMenu,
		&item.InSitemap,
		&item.Sort,
		&item.PublishedAt,
		&item.UnpublishedAt,
		&rawSettings,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CreatedBy,
		&item.UpdatedBy,
		&item.DeletedAt,
		&item.DeletedBy,
	); err != nil {
		return resource.Resource{}, err
	}

	if parentID != nil {
		value := resource.ID(*parentID)
		item.ParentID = &value
	}
	if templateCode != nil {
		value := template.Code(*templateCode)
		item.Template = &value
	}
	item.ContentType = contentType
	item.Path = path
	if imageMediaID != nil {
		value := media.ID(*imageMediaID)
		item.ImageMediaID = &value
	}
	if targetResourceID != nil {
		value := resource.ID(*targetResourceID)
		item.TargetResourceID = &value
	}
	item.ExternalURL = externalURL

	item.Settings = make(map[string]any)
	if len(rawSettings) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(rawSettings))
		decoder.UseNumber()
		if err := decoder.Decode(&item.Settings); err != nil {
			return resource.Resource{}, fmt.Errorf(
				"decode settings for resource %d: %w",
				item.ID,
				err,
			)
		}
	}

	return item, nil
}

func ensureMediaAvailable(
	ctx context.Context,
	transaction pgx.Tx,
	id media.ID,
	exclude resource.ID,
) error {
	var attached bool
	if err := transaction.QueryRow(ctx, `
SELECT EXISTS
(
    SELECT 1
    FROM core.resources
    WHERE image_media_id = $1
      AND ($2 = 0 OR id <> $2)

    UNION ALL

    SELECT 1
    FROM core.users
    WHERE avatar_media_id = $1
);
`, id, exclude).Scan(&attached); err != nil {
		return fmt.Errorf(
			"check media %d attachment: %w",
			id,
			err,
		)
	}
	if attached {
		return media.ErrAlreadyAttached
	}
	return nil
}

func treeMediaIDs(
	ctx context.Context,
	transaction pgx.Tx,
	id resource.ID,
	lock bool,
) ([]media.ID, bool, error) {
	query := `
WITH RECURSIVE tree AS
(
    SELECT id
    FROM core.resources
    WHERE id = $1

    UNION ALL

    SELECT child.id
    FROM core.resources AS child
    JOIN tree
      ON child.parent_id = tree.id
)
SELECT item.id, item.image_media_id
FROM core.resources AS item
JOIN tree
  ON tree.id = item.id
ORDER BY item.id`
	if lock {
		query += `
FOR UPDATE OF item`
	}
	query += ";"

	rows, err := transaction.Query(ctx, query, id)
	if err != nil {
		return nil, false, fmt.Errorf(
			"query resource %d delete tree: %w",
			id,
			err,
		)
	}
	defer rows.Close()

	seen := make(map[media.ID]struct{})
	result := make([]media.ID, 0)
	exists := false
	for rows.Next() {
		exists = true
		var (
			resourceID   resource.ID
			imageMediaID *int64
		)
		if err := rows.Scan(&resourceID, &imageMediaID); err != nil {
			return nil, false, fmt.Errorf(
				"scan resource delete tree: %w",
				err,
			)
		}
		if imageMediaID == nil {
			continue
		}
		mediaID := media.ID(*imageMediaID)
		if _, duplicate := seen[mediaID]; duplicate {
			continue
		}
		seen[mediaID] = struct{}{}
		result = append(result, mediaID)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf(
			"iterate resource delete tree: %w",
			err,
		)
	}
	return result, exists, nil
}

func mediaIDsContained(
	actual []media.ID,
	locked []media.ID,
) bool {
	lockedSet := make(map[media.ID]struct{}, len(locked))
	for _, id := range locked {
		lockedSet[id] = struct{}{}
	}
	for _, id := range actual {
		if _, exists := lockedSet[id]; !exists {
			return false
		}
	}
	return true
}

func equalMediaID(
	expected *media.ID,
	actual *int64,
) bool {
	if expected == nil || actual == nil {
		return expected == nil && actual == nil
	}
	return int64(*expected) == *actual
}

func sameMediaID(
	left *media.ID,
	right *media.ID,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func lockResource(
	ctx context.Context,
	transaction pgx.Tx,
	id resource.ID,
) (resource.Resource, error) {
	item, err := scanResource(transaction.QueryRow(ctx, `
SELECT
    id, site_id, parent_id, type, template, content_type,
	    title, menu_title, slug, path, annotation, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
	    updated_at, created_by, updated_by, deleted_at, deleted_by
FROM core.resources
WHERE id = $1
FOR UPDATE;
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.Resource{}, resource.ErrInvalidReference
	}
	if err != nil {
		return resource.Resource{}, fmt.Errorf(
			"lock resource %d: %w",
			id,
			err,
		)
	}
	return item, nil
}

func translateError(err error) error {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return err
	}

	switch postgresError.Code {
	case pgerrcode.UniqueViolation:
		return fmt.Errorf("%w: %s", resource.ErrConflict, err)
	case pgerrcode.ForeignKeyViolation:
		return fmt.Errorf("%w: %s", resource.ErrInvalidReference, err)
	default:
		return err
	}
}

func translateDeleteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) &&
		postgresError.Code == pgerrcode.ForeignKeyViolation {
		return fmt.Errorf("%w: %s", resource.ErrReferenced, err)
	}
	return translateError(err)
}

func replaceFileReferences(
	ctx context.Context,
	tx pgx.Tx,
	ownerID resource.ID,
	references map[string]file.ID,
) error {
	if _, err := tx.Exec(ctx, `DELETE FROM core.file_field_references WHERE owner_kind = 'resource' AND owner_id = $1;`, ownerID); err != nil {
		return fmt.Errorf("delete resource file references: %w", err)
	}
	for key, id := range references {
		if _, err := tx.Exec(ctx, `
INSERT INTO core.file_field_references (owner_kind, owner_id, field_key, file_id)
VALUES ('resource', $1, $2, $3);`, ownerID, key, id); err != nil {
			return fmt.Errorf("insert resource file reference: %w", err)
		}
	}
	return nil
}

func cloneFileReferences(source map[string]file.ID) map[string]file.ID {
	if source == nil {
		return nil
	}
	result := make(map[string]file.ID, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

var _ resource.Repository = (*Repository)(nil)
var _ resource.WidgetRepository = (*Repository)(nil)
var _ resource.ManagementRepository = (*Repository)(nil)
var _ resource.StatisticsRepository = (*Repository)(nil)
