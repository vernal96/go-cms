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
	"github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres/medialock"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
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
    $20, $21::jsonb, $22, $22
)
RETURNING
    id, site_id, parent_id, type, template, content_type,
    title, menu_title, slug, path, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
    updated_at, created_by, updated_by;
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
    title, menu_title, slug, path, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
    updated_at, created_by, updated_by
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

	return result, nil
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
    title, menu_title, slug, path, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
    updated_at, created_by, updated_by
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

	return result, nil
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
    title, menu_title, slug, path, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
    updated_at, created_by, updated_by
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

	return result, nil
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
	)
	if err := transaction.QueryRow(ctx, `
SELECT site_id, image_media_id
FROM core.resources
WHERE id = $1
FOR UPDATE;
`, item.ID).Scan(
		&currentSiteID,
		&currentImageMediaID,
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
    content = $10,
    image_media_id = $11,
    target_resource_id = $12,
    external_url = $13,
    is_public = $14,
    is_searchable = $15,
    in_menu = $16,
    in_sitemap = $17,
    sort = $18,
    published_at = $19,
    unpublished_at = $20,
    settings = $21::jsonb,
    updated_at = now(),
    updated_by = $22
WHERE id = $1
RETURNING
    id, site_id, parent_id, type, template, content_type,
    title, menu_title, slug, path, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
    updated_at, created_by, updated_by;
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

type rowScanner interface {
	Scan(...any) error
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
    title, menu_title, slug, path, content, image_media_id,
    target_resource_id,
    external_url, is_public, is_searchable, in_menu, in_sitemap,
    sort, published_at, unpublished_at, settings, created_at,
    updated_at, created_by, updated_by
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

var _ resource.Repository = (*Repository)(nil)
