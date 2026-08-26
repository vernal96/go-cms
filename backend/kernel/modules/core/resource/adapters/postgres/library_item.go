package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres/medialock"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/security"
)

const libraryItemColumns = `
    item.id, item.site_id, item.library_id, item.template, item.content_type,
    item.title, item.slug, item.annotation, item.content, item.image_media_id,
    item.is_public, item.is_searchable, item.published_at, item.unpublished_at,
    item.created_at, item.updated_at, item.created_by, item.updated_by,
    item.deleted_at, item.deleted_by`

type libraryRowQueryer interface {
	rowQueryer
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Repository) CreateLibraryItem(ctx context.Context, actorID *security.UserID, item resource.LibraryItem, recordRevision bool) (_ resource.LibraryItem, resultErr error) {
	if ctx == nil {
		return resource.LibraryItem{}, errors.New("create library item context is nil")
	}
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return resource.LibraryItem{}, err
	}
	defer func() {
		if resultErr != nil {
			_ = tx.Rollback(context.Background())
		}
	}()
	if err := ensureLibraryTarget(ctx, tx, item.SiteID, item.LibraryID); err != nil {
		return resource.LibraryItem{}, err
	}
	if item.ImageMediaID != nil {
		if err := medialock.Lock(ctx, tx, *item.ImageMediaID); err != nil {
			return resource.LibraryItem{}, err
		}
		if err := ensureMediaAvailable(ctx, tx, *item.ImageMediaID, 0); err != nil {
			return resource.LibraryItem{}, err
		}
	}
	if err := tx.QueryRow(ctx, `INSERT INTO core.resource_entities (site_id, storage_kind) VALUES ($1, 'library_item') RETURNING id;`, item.SiteID).Scan(&item.ID); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	item.CreatedAt = time.Now().UTC()
	partitionAt := item.CreatedAt
	if item.PublishedAt != nil {
		partitionAt = item.PublishedAt.UTC()
	}
	if _, err := tx.Exec(ctx, `INSERT INTO core.library_item_routes (resource_id, site_id, library_id, slug) VALUES ($1, $2, $3, $4);`, item.ID, item.SiteID, item.LibraryID, item.Slug); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	stored, err := scanLibraryItem(tx.QueryRow(ctx, `
INSERT INTO core.library_items AS item (
    id, site_id, library_id, partition_at, template, content_type, title, slug,
    annotation, content, image_media_id, is_public, is_searchable,
    published_at, unpublished_at, created_at, created_by, updated_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17)
RETURNING `+libraryItemColumns+`;`, item.ID, item.SiteID, item.LibraryID, partitionAt, item.Template, item.ContentType, item.Title, item.Slug, item.Annotation, item.Content, item.ImageMediaID, item.IsPublic, item.IsSearchable, item.PublishedAt, item.UnpublishedAt, item.CreatedAt, actorID))
	if err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	if err := replaceResourceFields(ctx, tx, stored.ID, stored.SiteID, item.FieldValues); err != nil {
		return resource.LibraryItem{}, err
	}
	if err := replaceFileReferences(ctx, tx, stored.ID, item.FileReferences); err != nil {
		return resource.LibraryItem{}, err
	}
	stored.Fields, stored.FieldValues, stored.FileReferences = item.Fields, item.FieldValues, item.FileReferences
	stored.Widgets = widget.CloneBindings(item.Widgets)
	stored.Version = 1
	if recordRevision {
		if err := r.appendLibraryItemRevision(ctx, tx, stored, resource.RevisionCreated, nil, actorID); err != nil {
			return resource.LibraryItem{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	return stored, nil
}

func (r *Repository) LibraryItemByID(ctx context.Context, id resource.ID) (resource.LibraryItem, error) {
	if ctx == nil || id <= 0 {
		return resource.LibraryItem{}, resource.ErrInvalid
	}
	item, err := r.libraryItemByID(ctx, r.connector.Pool(), id, false)
	if err != nil {
		return resource.LibraryItem{}, err
	}
	if err := r.loadLibraryItemFields(ctx, r.connector.Pool(), &item); err != nil {
		return resource.LibraryItem{}, err
	}
	return item, nil
}

func (r *Repository) libraryItemByID(ctx context.Context, queryer libraryRowQueryer, id resource.ID, lock bool) (resource.LibraryItem, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE OF item"
	}
	item, err := scanLibraryItem(queryer.QueryRow(ctx, `
SELECT `+libraryItemColumns+`
FROM core.library_item_routes route
JOIN core.library_items item
  ON item.id = route.resource_id AND item.library_id = route.library_id
WHERE route.resource_id = $1`+suffix+`;`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.LibraryItem{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.LibraryItem{}, fmt.Errorf("query library item %d: %w", id, err)
	}
	if err := queryer.QueryRow(ctx, `SELECT version FROM core.resource_entities WHERE id=$1;`, id).Scan(&item.Version); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	return item, nil
}

func (r *Repository) UpdateLibraryItem(ctx context.Context, actorID *security.UserID, current, item resource.LibraryItem, recordRevision bool) (_ resource.LibraryItem, resultErr error) {
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return resource.LibraryItem{}, err
	}
	defer func() {
		if resultErr != nil {
			_ = tx.Rollback(context.Background())
		}
	}()
	locked, err := r.libraryItemByID(ctx, tx, current.ID, true)
	if err != nil {
		return resource.LibraryItem{}, err
	}
	if current.Version <= 0 || locked.Version != current.Version {
		return resource.LibraryItem{}, resource.ErrConflict
	}
	var nextVersion int64
	if err := tx.QueryRow(ctx, `UPDATE core.resource_entities SET version=version+1 WHERE id=$1 AND version=$2 RETURNING version;`, item.ID, current.Version).Scan(&nextVersion); errors.Is(err, pgx.ErrNoRows) {
		return resource.LibraryItem{}, resource.ErrConflict
	} else if err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	if item.ImageMediaID != nil || locked.ImageMediaID != nil {
		ids := make([]media.ID, 0, 2)
		if item.ImageMediaID != nil {
			ids = append(ids, *item.ImageMediaID)
		}
		if locked.ImageMediaID != nil {
			ids = append(ids, *locked.ImageMediaID)
		}
		if err := medialock.Lock(ctx, tx, ids...); err != nil {
			return resource.LibraryItem{}, err
		}
		if item.ImageMediaID != nil {
			if err := ensureMediaAvailable(ctx, tx, *item.ImageMediaID, item.ID); err != nil {
				return resource.LibraryItem{}, err
			}
		}
	}
	partitionAt := locked.CreatedAt
	if item.PublishedAt != nil {
		partitionAt = item.PublishedAt.UTC()
	}
	if _, err := tx.Exec(ctx, `UPDATE core.library_item_routes SET slug = $2 WHERE resource_id = $1;`, item.ID, item.Slug); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	updated, err := scanLibraryItem(tx.QueryRow(ctx, `
UPDATE core.library_items AS item SET
    partition_at=$2, template=$3, content_type=$4, title=$5, slug=$6,
    annotation=$7, content=$8, image_media_id=$9, is_public=$10,
    is_searchable=$11, published_at=$12, unpublished_at=$13,
    updated_at=now(), updated_by=$14
WHERE id=$1 AND library_id=$15
RETURNING `+libraryItemColumns+`;`, item.ID, partitionAt, item.Template, item.ContentType, item.Title, item.Slug, item.Annotation, item.Content, item.ImageMediaID, item.IsPublic, item.IsSearchable, item.PublishedAt, item.UnpublishedAt, actorID, locked.LibraryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.LibraryItem{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	if err := replaceResourceFields(ctx, tx, updated.ID, updated.SiteID, item.FieldValues); err != nil {
		return resource.LibraryItem{}, err
	}
	if err := replaceFileReferences(ctx, tx, updated.ID, item.FileReferences); err != nil {
		return resource.LibraryItem{}, err
	}
	updated.Fields, updated.FieldValues, updated.FileReferences = item.Fields, item.FieldValues, item.FileReferences
	updated.Widgets = widget.CloneBindings(item.Widgets)
	updated.Version = nextVersion
	if recordRevision {
		if err := r.appendLibraryItemRevision(ctx, tx, updated, resource.RevisionUpdated, nil, actorID); err != nil {
			return resource.LibraryItem{}, err
		}
	}
	if !sameMediaID(locked.ImageMediaID, item.ImageMediaID) && locked.ImageMediaID != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM core.media WHERE id=$1;`, *locked.ImageMediaID); err != nil {
			return resource.LibraryItem{}, translateError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	return updated, nil
}

func (r *Repository) SoftDeleteLibraryItem(ctx context.Context, actorID *security.UserID, id resource.ID) error {
	command, err := r.connector.Pool().Exec(ctx, `UPDATE core.library_items SET deleted_at=coalesce(deleted_at,now()), deleted_by=coalesce(deleted_by,$2), updated_at=now(), updated_by=$2 WHERE id=$1;`, id, actorID)
	if err != nil {
		return translateError(err)
	}
	if command.RowsAffected() == 0 {
		return resource.ErrNotFound
	}
	return nil
}

func (r *Repository) RestoreLibraryItem(ctx context.Context, actorID *security.UserID, id resource.ID) error {
	command, err := r.connector.Pool().Exec(ctx, `UPDATE core.library_items item SET deleted_at=NULL, deleted_by=NULL, updated_at=now(), updated_by=$2 FROM core.resources library WHERE item.id=$1 AND library.id=item.library_id AND library.site_id=item.site_id AND library.type='library' AND library.deleted_at IS NULL;`, id, actorID)
	if err != nil {
		return translateError(err)
	}
	if command.RowsAffected() == 0 {
		return resource.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteLibraryItem(ctx context.Context, id resource.ID) error {
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	item, err := r.libraryItemByID(ctx, tx, id, true)
	if err != nil {
		return err
	}
	if item.ImageMediaID != nil {
		if err := medialock.Lock(ctx, tx, *item.ImageMediaID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.file_field_references WHERE owner_kind='resource' AND owner_id=$1;`, id); err != nil {
		return translateDeleteError(err)
	}
	command, err := tx.Exec(ctx, `DELETE FROM core.resource_entities WHERE id=$1 AND storage_kind='library_item';`, id)
	if err != nil {
		return translateDeleteError(err)
	}
	if command.RowsAffected() == 0 {
		return resource.ErrNotFound
	}
	if item.ImageMediaID != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM core.media WHERE id=$1;`, *item.ImageMediaID); err != nil {
			return translateDeleteError(err)
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) MoveLibraryItem(ctx context.Context, actorID *security.UserID, id, targetLibraryID resource.ID, expectedVersion int64, recordRevision bool) (_ resource.LibraryItem, resultErr error) {
	tx, err := r.connector.Pool().Begin(ctx)
	if err != nil {
		return resource.LibraryItem{}, err
	}
	defer func() {
		if resultErr != nil {
			_ = tx.Rollback(context.Background())
		}
	}()
	item, err := r.libraryItemByID(ctx, tx, id, true)
	if err != nil {
		return resource.LibraryItem{}, err
	}
	if item.Version != expectedVersion {
		return resource.LibraryItem{}, resource.ErrConflict
	}
	if err := tx.QueryRow(ctx, `UPDATE core.resource_entities SET version=version+1 WHERE id=$1 AND version=$2 RETURNING version;`, id, expectedVersion).Scan(&item.Version); errors.Is(err, pgx.ErrNoRows) {
		return resource.LibraryItem{}, resource.ErrConflict
	} else if err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	if err := ensureLibraryTarget(ctx, tx, item.SiteID, targetLibraryID); err != nil {
		return resource.LibraryItem{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE core.library_item_routes SET library_id=$2 WHERE resource_id=$1;`, id, targetLibraryID); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	moved, err := scanLibraryItem(tx.QueryRow(ctx, `UPDATE core.library_items AS item SET library_id=$2, updated_at=now(), updated_by=$3 WHERE id=$1 AND library_id=$4 RETURNING `+libraryItemColumns+`;`, id, targetLibraryID, actorID, item.LibraryID))
	if err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	moved.Version = item.Version
	if recordRevision {
		if err := r.loadLibraryItemFields(ctx, tx, &moved); err != nil {
			return resource.LibraryItem{}, err
		}
		if err := r.appendLibraryItemRevision(ctx, tx, moved, resource.RevisionUpdated, nil, actorID); err != nil {
			return resource.LibraryItem{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return resource.LibraryItem{}, translateError(err)
	}
	if err := r.loadLibraryItemFields(ctx, r.connector.Pool(), &moved); err != nil {
		return resource.LibraryItem{}, err
	}
	return moved, nil
}

func (r *Repository) QueryLibraryItems(ctx context.Context, query resource.LibraryItemQuery) (resource.LibraryItemPage, error) {
	args := []any{query.SiteID, query.LibraryID, query.Limit + 1}
	where := []string{"item.site_id=$1", "item.library_id=$2", "library.type='library'", "library.deleted_at IS NULL"}
	if query.AfterID > 0 {
		args = append(args, query.AfterID)
		where = append(where, fmt.Sprintf("item.id < $%d", len(args)))
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		textArg := len(args)
		clause := fmt.Sprintf("(lower(item.title) LIKE $%d OR lower(item.slug) LIKE $%d)", textArg, textArg)
		if id, parseErr := strconv.ParseInt(search, 10, 64); parseErr == nil && id > 0 {
			args = append(args, resource.ID(id))
			clause = fmt.Sprintf("(item.id=$%d OR lower(item.title) LIKE $%d OR lower(item.slug) LIKE $%d)", len(args), textArg, textArg)
		}
		where = append(where, clause)
	}
	if query.PublicOnly {
		where = append(where, "item.deleted_at IS NULL", "item.is_public", "(item.published_at IS NULL OR item.published_at<=now())", "(item.unpublished_at IS NULL OR now()<item.unpublished_at)")
	}
	if query.Deleted != nil {
		if *query.Deleted {
			where = append(where, "item.deleted_at IS NOT NULL")
		} else {
			where = append(where, "item.deleted_at IS NULL")
		}
	}
	rows, err := r.connector.Pool().Query(ctx, `SELECT `+libraryItemColumns+` FROM core.library_items item JOIN core.resources library ON library.id=item.library_id AND library.site_id=item.site_id WHERE `+strings.Join(where, " AND ")+` ORDER BY item.id DESC LIMIT $3;`, args...)
	if err != nil {
		return resource.LibraryItemPage{}, fmt.Errorf("query library items: %w", err)
	}
	defer rows.Close()
	items := make([]resource.LibraryItem, 0, query.Limit+1)
	for rows.Next() {
		item, err := scanLibraryItem(rows)
		if err != nil {
			return resource.LibraryItemPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return resource.LibraryItemPage{}, err
	}
	rows.Close()
	page := resource.LibraryItemPage{}
	if len(items) > query.Limit {
		page.NextCursor = resource.EncodeLibraryCursor(items[query.Limit-1].ID)
		items = items[:query.Limit]
	}
	if err := r.loadLibraryItemsFields(ctx, r.connector.Pool(), items); err != nil {
		return resource.LibraryItemPage{}, err
	}
	page.Items = items
	return page, nil
}

func (r *Repository) ResolveLibraryItemRoute(ctx context.Context, siteID site.ID, path string) (resource.LibraryItem, resource.Resource, error) {
	rows, err := r.connector.Pool().Query(ctx, `SELECT id, site_id, parent_id, type, template, content_type, title, menu_title, slug, path, annotation, content, image_media_id, target_resource_id, external_url, is_public, is_searchable, in_menu, in_sitemap, sort, published_at, unpublished_at, type_settings, created_at, updated_at, created_by, updated_by, deleted_at, deleted_by FROM core.resources WHERE site_id=$1 AND type='library' AND path IS NOT NULL AND (path='/' OR $2=path OR $2 LIKE path||'/%') ORDER BY length(path) DESC, id LIMIT 16;`, siteID, path)
	if err != nil {
		return resource.LibraryItem{}, resource.Resource{}, err
	}
	defer rows.Close()
	for rows.Next() {
		library, err := scanResource(rows)
		if err != nil {
			return resource.LibraryItem{}, resource.Resource{}, err
		}
		pattern, _ := library.TypeSettings["item_url_pattern"].(string)
		if pattern == "" {
			pattern = resourcetype.DefaultItemURLPattern
		}
		relative := strings.TrimPrefix(path, *library.Path)
		if *library.Path == "/" {
			relative = path
		}
		key, matched := resource.MatchLibraryItemPattern(pattern, relative)
		if !matched {
			continue
		}
		var item resource.LibraryItem
		if key.ID > 0 {
			item, err = r.libraryItemByID(ctx, r.connector.Pool(), key.ID, false)
		} else {
			item, err = scanLibraryItem(r.connector.Pool().QueryRow(ctx, `SELECT `+libraryItemColumns+` FROM core.library_item_routes route JOIN core.library_items item ON item.id=route.resource_id AND item.library_id=route.library_id WHERE route.library_id=$1 AND route.slug=$2;`, library.ID, key.Slug))
		}
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, resource.ErrNotFound) {
			continue
		}
		if err != nil {
			return resource.LibraryItem{}, resource.Resource{}, err
		}
		if item.LibraryID != library.ID {
			continue
		}
		effectiveURL, urlErr := resource.EffectiveLibraryItemURL(library, item)
		if urlErr != nil {
			return resource.LibraryItem{}, resource.Resource{}, urlErr
		}
		if effectiveURL != path {
			continue
		}
		if err := r.loadLibraryItemFields(ctx, r.connector.Pool(), &item); err != nil {
			return resource.LibraryItem{}, resource.Resource{}, err
		}
		return item, library, nil
	}
	return resource.LibraryItem{}, resource.Resource{}, resource.ErrNotFound
}

func ensureLibraryTarget(ctx context.Context, queryer libraryRowQueryer, siteID site.ID, libraryID resource.ID) error {
	var id resource.ID
	err := queryer.QueryRow(ctx, `SELECT id FROM core.resources WHERE id=$1 AND site_id=$2 AND type='library' AND deleted_at IS NULL FOR SHARE;`, libraryID, siteID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.ErrInvalidReference
	}
	if err != nil {
		return translateError(err)
	}
	return nil
}

func scanLibraryItem(scanner rowScanner) (resource.LibraryItem, error) {
	var item resource.LibraryItem
	var templateCode, contentType *string
	var imageMediaID *int64
	err := scanner.Scan(&item.ID, &item.SiteID, &item.LibraryID, &templateCode, &contentType, &item.Title, &item.Slug, &item.Annotation, &item.Content, &imageMediaID, &item.IsPublic, &item.IsSearchable, &item.PublishedAt, &item.UnpublishedAt, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy, &item.UpdatedBy, &item.DeletedAt, &item.DeletedBy)
	if err != nil {
		return resource.LibraryItem{}, err
	}
	if templateCode != nil {
		code := template.Code(*templateCode)
		item.Template = &code
	}
	item.ContentType = contentType
	if imageMediaID != nil {
		id := media.ID(*imageMediaID)
		item.ImageMediaID = &id
	}
	return item, nil
}

func (r *Repository) loadLibraryItemFields(ctx context.Context, queryer rowQueryer, item *resource.LibraryItem) error {
	items := []resource.LibraryItem{*item}
	if err := r.loadLibraryItemsFields(ctx, queryer, items); err != nil {
		return err
	}
	*item = items[0]
	return nil
}
func (r *Repository) loadLibraryItemsFields(ctx context.Context, queryer rowQueryer, items []resource.LibraryItem) error {
	projected := make([]resource.Resource, len(items))
	for i := range items {
		projected[i] = resource.Resource{ID: items[i].ID}
	}
	if err := loadResourceFields(ctx, queryer, projected); err != nil {
		return err
	}
	if err := loadResourceWidgets(ctx, queryer, projected); err != nil {
		return err
	}
	for i := range items {
		items[i].Fields = projected[i].Fields
		items[i].FieldValues = projected[i].FieldValues
		items[i].Widgets = projected[i].Widgets
	}
	return nil
}

var _ resource.LibraryItemRepository = (*Repository)(nil)
