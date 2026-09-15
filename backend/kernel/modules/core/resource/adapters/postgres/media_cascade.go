package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/security"
)

// ClearMediaReferences participates in the filesystem transaction. Every owner
// goes through its hooks, history policy and outbox before physical deletion.
func (r *Repository) ClearMediaReferences(ctx context.Context, tx pgx.Tx, mediaIDs []int64, actorID *security.UserID) error {
	rows, err := tx.Query(ctx, `SELECT id FROM core.resources WHERE image_media_id=ANY($1::bigint[])
 UNION SELECT id FROM core.library_items WHERE image_media_id=ANY($1::bigint[])
 UNION SELECT resource_id FROM core.resource_media_references WHERE media_id=ANY($1::bigint[]) ORDER BY id`, mediaIDs)
	if err != nil {
		return err
	}
	var ids []resource.ID
	for rows.Next() {
		var id resource.ID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		before, err := r.eventState(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := clearStructuredMediaReferences(ctx, tx, id, mediaIDs); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM core.resource_field_values fv USING core.resource_media_references mr
   WHERE fv.resource_id=$1 AND fv.resource_id=mr.resource_id AND fv.field_key=mr.field_key AND fv.position=mr.position AND cardinality(mr.value_path)=0 AND mr.media_id=ANY($2::bigint[])`, id, mediaIDs); err != nil {
			return err
		}
		query := `UPDATE core.resources SET image_media_id=CASE WHEN image_media_id=ANY($2::bigint[]) THEN NULL ELSE image_media_id END, updated_at=clock_timestamp(), updated_by=$3 WHERE id=$1`
		if before.Data.StorageKind == resource.StorageLibraryItem {
			query = `UPDATE core.library_items SET image_media_id=CASE WHEN image_media_id=ANY($2::bigint[]) THEN NULL ELSE image_media_id END, updated_at=clock_timestamp(), updated_by=$3 WHERE id=$1`
		}
		if _, err := tx.Exec(ctx, query, id, mediaIDs, actorID); err != nil {
			return err
		}
		after, err := r.eventState(ctx, tx, id)
		if err != nil {
			return err
		}
		if _, err := resource.PrepareMutation(ctx, &before, after); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `UPDATE core.resource_entities SET version=version+1 WHERE id=$1 RETURNING version`, id).Scan(&after.Version); err != nil {
			return err
		}
		if resource.MediaCascadeRecordsRevision(ctx, after) {
			if err := r.appendCurrentRevision(ctx, tx, id, after.Version, actorID); err != nil {
				return err
			}
		}
		if err := r.appendStateEvent(ctx, tx, resource.EventUpdated, after, actorID); err != nil {
			return err
		}
	}
	return nil
}

// JSON paths come from the compiled field collector, never from a field-type
// switch. Removing an object member preserves the containing rows and order.
func clearStructuredMediaReferences(ctx context.Context, tx pgx.Tx, id resource.ID, mediaIDs []int64) error {
	rows, err := tx.Query(ctx, `SELECT field_key,position,value_path FROM core.resource_media_references WHERE resource_id=$1 AND media_id=ANY($2::bigint[]) AND cardinality(value_path)>0 ORDER BY field_key,position,value_path`, id, mediaIDs)
	if err != nil {
		return err
	}
	type location struct {
		key      string
		position int
		path     []string
	}
	var locations []location
	for rows.Next() {
		var item location
		if err := rows.Scan(&item.key, &item.position, &item.path); err != nil {
			rows.Close()
			return err
		}
		locations = append(locations, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range locations {
		if _, err := tx.Exec(ctx, `UPDATE core.resource_field_values SET value_json=value_json #- $4::text[] WHERE resource_id=$1 AND field_key=$2 AND position=$3 AND value_kind='json'`, id, item.key, item.position, item.path); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `DELETE FROM core.resource_media_references WHERE resource_id=$1 AND media_id=ANY($2::bigint[]) AND cardinality(value_path)>0`, id, mediaIDs)
	return err
}
