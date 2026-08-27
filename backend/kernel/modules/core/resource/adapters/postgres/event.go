package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vernal96/go-cms/kernel/domainevent"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

func appendResourceEvent(ctx context.Context, tx pgx.Tx, name string, resourceID resource.ID, siteID site.ID, storage resource.StorageKind, version int64, actorID *security.UserID) error {
	event, err := resource.NewEvent(name, time.Now().UTC(), resource.EventPayload{
		ResourceID: resourceID, SiteID: siteID, StorageKind: storage, Version: version, ActorID: actorID,
	})
	if err != nil {
		return fmt.Errorf("create %s event: %w", name, err)
	}
	message, err := domainevent.Message(event, []byte(strconv.FormatInt(int64(resourceID), 10)))
	if err != nil {
		return fmt.Errorf("encode %s event: %w", name, err)
	}
	headers, err := json.Marshal(message.Headers)
	if err != nil {
		return fmt.Errorf("encode %s event headers: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO core.outbox_messages
    (message_id,topic,message_key,body,headers,created_at,available_at)
VALUES ($1,$2,$3,$4,$5::jsonb,$6,$6);`, event.ID, message.Topic, message.Key, message.Body, string(headers), event.OccurredAt); err != nil {
		return fmt.Errorf("append %s outbox message: %w", name, translateError(err))
	}
	return nil
}

func appendWidgetResourceEvent(ctx context.Context, tx pgx.Tx, resourceID resource.ID, version int64, actorID *security.UserID) error {
	var siteID site.ID
	var storage resource.StorageKind
	if err := tx.QueryRow(ctx, `SELECT site_id,storage_kind FROM core.resource_entities WHERE id=$1;`, resourceID).Scan(&siteID, &storage); err != nil {
		return translateError(err)
	}
	return appendResourceEvent(ctx, tx, resource.EventUpdated, resourceID, siteID, storage, version, actorID)
}
