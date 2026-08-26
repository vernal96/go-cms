package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type routeQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func lockRouteNamespace(ctx context.Context, tx pgx.Tx, siteID site.ID) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('core.routes:' || $1::bigint::text, 0));`, siteID); err != nil {
		return fmt.Errorf("lock site route namespace: %w", err)
	}
	return nil
}

func routeResourceByID(ctx context.Context, queryer routeQueryer, id resource.ID) (resource.Resource, error) {
	item, err := scanResource(queryer.QueryRow(ctx, `
SELECT id, site_id, parent_id, type, template, content_type, title, menu_title, slug, path,
 annotation, content, image_media_id, target_resource_id, external_url, is_public, is_searchable,
 in_menu, in_sitemap, sort, published_at, unpublished_at, type_settings, created_at, updated_at,
 created_by, updated_by, deleted_at, deleted_by
FROM core.resources WHERE id=$1;`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.Resource{}, resource.ErrNotFound
	}
	return item, err
}

func ensureLibraryItemRouteAvailable(ctx context.Context, queryer routeQueryer, library resource.Resource, item resource.LibraryItem) error {
	path, err := resource.EffectiveLibraryItemURL(library, item)
	if err != nil {
		return err
	}
	var treeConflict bool
	if err := queryer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.resources WHERE site_id=$1 AND path=$2);`, item.SiteID, path).Scan(&treeConflict); err != nil {
		return translateError(err)
	}
	if treeConflict {
		return resource.ErrRouteConflict
	}
	conflict, err := libraryItemRouteExists(ctx, queryer, item.SiteID, path, item.ID, nil)
	if err != nil {
		return err
	}
	if conflict {
		return resource.ErrRouteConflict
	}
	return nil
}

func ensureTreePathsAvailable(ctx context.Context, queryer routeQueryer, siteID site.ID, paths []string, override *resource.Resource) error {
	for _, path := range paths {
		if path == "" {
			continue
		}
		conflict, err := libraryItemRouteExists(ctx, queryer, siteID, path, 0, override)
		if err != nil {
			return err
		}
		if conflict {
			return resource.ErrRouteConflict
		}
	}
	return nil
}

func ensureProspectiveLibraryRoutesAvailable(ctx context.Context, queryer routeQueryer, library resource.Resource) error {
	rows, err := queryer.Query(ctx, `SELECT `+libraryItemColumns+` FROM core.library_items item WHERE item.site_id=$1 AND item.library_id=$2;`, library.SiteID, library.ID)
	if err != nil {
		return translateError(err)
	}
	items := make([]resource.LibraryItem, 0)
	for rows.Next() {
		item, err := scanLibraryItem(rows)
		if err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return translateError(err)
	}
	rows.Close()
	for _, item := range items {
		if err := ensureLibraryItemRouteAvailable(ctx, queryer, library, item); err != nil {
			return err
		}
	}
	return nil
}

func libraryItemRouteExists(ctx context.Context, queryer routeQueryer, siteID site.ID, path string, excludeID resource.ID, override *resource.Resource) (bool, error) {
	rows, err := queryer.Query(ctx, `
SELECT id, site_id, parent_id, type, template, content_type, title, menu_title, slug, path,
 annotation, content, image_media_id, target_resource_id, external_url, is_public, is_searchable,
 in_menu, in_sitemap, sort, published_at, unpublished_at, type_settings, created_at, updated_at,
 created_by, updated_by, deleted_at, deleted_by
FROM core.resources
WHERE site_id=$1 AND type='library' AND path IS NOT NULL
  AND (path='/' OR $2=path OR $2 LIKE path||'/%')
ORDER BY length(path) DESC, id;`, siteID, path)
	if err != nil {
		return false, translateError(err)
	}
	defer rows.Close()
	libraries := make([]resource.Resource, 0, 4)
	for rows.Next() {
		library, err := scanResource(rows)
		if err != nil {
			return false, err
		}
		libraries = append(libraries, library)
	}
	if err := rows.Err(); err != nil {
		return false, translateError(err)
	}
	if override != nil && override.SiteID == siteID && override.Type == resourcetype.Library && override.Path != nil {
		replaced := false
		for index := range libraries {
			if libraries[index].ID == override.ID {
				libraries[index] = resource.Clone(*override)
				replaced = true
				break
			}
		}
		if !replaced && (*override.Path == "/" || path == *override.Path || strings.HasPrefix(path, strings.TrimRight(*override.Path, "/")+"/")) {
			libraries = append(libraries, resource.Clone(*override))
		}
	}
	for _, library := range libraries {
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
		var candidateID resource.ID
		if key.ID > 0 {
			err = queryer.QueryRow(ctx, `SELECT resource_id FROM core.library_item_routes WHERE library_id=$1 AND resource_id=$2;`, library.ID, key.ID).Scan(&candidateID)
		} else {
			err = queryer.QueryRow(ctx, `SELECT resource_id FROM core.library_item_routes WHERE library_id=$1 AND slug=$2;`, library.ID, key.Slug).Scan(&candidateID)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return false, translateError(err)
		}
		if candidateID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func prospectiveTreePaths(ctx context.Context, queryer routeQueryer, rootID resource.ID, rootPath *string) ([]string, error) {
	rows, err := queryer.Query(ctx, `
WITH RECURSIVE tree AS (
 SELECT id, $2::text AS path FROM core.resources WHERE id=$1
 UNION ALL
 SELECT child.id,
  CASE WHEN child.path IS NULL OR tree.path IS NULL THEN NULL
       WHEN tree.path='/' THEN '/'||child.slug ELSE tree.path||'/'||child.slug END
 FROM core.resources child JOIN tree ON child.parent_id=tree.id
)
SELECT path FROM tree WHERE path IS NOT NULL;`, rootID, rootPath)
	if err != nil {
		return nil, translateError(err)
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, translateError(rows.Err())
}
