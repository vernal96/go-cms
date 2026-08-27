package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/messageid"
	"github.com/vernal96/go-cms/kernel/outbox"
)

type outboxSource struct {
	connector *connectorpostgres.Connector
	name      string
}

func newOutboxSource(connector *connectorpostgres.Connector) *outboxSource {
	return &outboxSource{connector: connector, name: "core:" + string(connector.Code())}
}

func (s *outboxSource) Name() string { return s.name }

func (s *outboxSource) Claim(ctx context.Context, claim outbox.Claim) (_ []outbox.Record, resultErr error) {
	if ctx == nil || strings.TrimSpace(claim.Owner) == "" || claim.Limit < 1 || claim.LeaseDuration <= 0 || claim.Now.IsZero() {
		return nil, errors.New("PostgreSQL outbox claim is invalid")
	}
	tx, err := s.connector.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin outbox claim: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(context.Background()); resultErr != nil && rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			resultErr = errors.Join(resultErr, rollbackErr)
		}
	}()
	rows, err := tx.Query(ctx, `
WITH candidates AS (
    SELECT message_id
    FROM core.outbox_messages
    WHERE published_at IS NULL
      AND available_at <= $1
      AND (lease_until IS NULL OR lease_until <= $1)
    ORDER BY available_at, created_at, message_id
    FOR UPDATE SKIP LOCKED
    LIMIT $2
)
UPDATE core.outbox_messages AS message
SET lease_owner=$3, lease_until=$4
FROM candidates
WHERE message.message_id=candidates.message_id
RETURNING message.message_id,message.topic,message.message_key,message.body,message.headers,
          message.created_at,message.available_at,message.attempt_count,coalesce(message.last_error,''),
          message.lease_owner,message.lease_until,message.published_at;`, claim.Now.UTC(), claim.Limit, claim.Owner, claim.Now.UTC().Add(claim.LeaseDuration))
	if err != nil {
		return nil, fmt.Errorf("claim outbox messages: %w", err)
	}
	defer rows.Close()
	records := make([]outbox.Record, 0, claim.Limit)
	for rows.Next() {
		var record outbox.Record
		var rawHeaders []byte
		if err := rows.Scan(&record.MessageID, &record.Topic, &record.Key, &record.Body, &rawHeaders,
			&record.CreatedAt, &record.AvailableAt, &record.AttemptCount, &record.LastError,
			&record.LeaseOwner, &record.LeaseUntil, &record.PublishedAt); err != nil {
			return nil, fmt.Errorf("scan outbox message: %w", err)
		}
		if err := json.Unmarshal(rawHeaders, &record.Headers); err != nil {
			return nil, fmt.Errorf("decode outbox message %q headers: %w", record.MessageID, err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox messages: %w", err)
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit outbox claim: %w", err)
	}
	return records, nil
}

func (s *outboxSource) MarkPublished(ctx context.Context, id messageid.ID, owner string, at time.Time) error {
	command, err := s.connector.Pool().Exec(ctx, `
UPDATE core.outbox_messages
SET published_at=$3,lease_owner=NULL,lease_until=NULL,last_error=NULL
WHERE message_id=$1 AND lease_owner=$2 AND published_at IS NULL;`, id, owner, at.UTC())
	if err != nil {
		return fmt.Errorf("mark outbox message published: %w", err)
	}
	if command.RowsAffected() != 1 {
		return outbox.ErrLeaseLost
	}
	return nil
}

func (s *outboxSource) MarkFailed(ctx context.Context, id messageid.ID, owner, failure string, availableAt time.Time) error {
	if len(failure) > 4096 {
		failure = failure[:4096]
	}
	command, err := s.connector.Pool().Exec(ctx, `
UPDATE core.outbox_messages
SET attempt_count=attempt_count+1,last_error=$3,available_at=$4,lease_owner=NULL,lease_until=NULL
WHERE message_id=$1 AND lease_owner=$2 AND published_at IS NULL;`, id, owner, failure, availableAt.UTC())
	if err != nil {
		return fmt.Errorf("mark outbox message failed: %w", err)
	}
	if command.RowsAffected() != 1 {
		return outbox.ErrLeaseLost
	}
	return nil
}

func (s *outboxSource) CleanupPublished(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit < 1 {
		return 0, errors.New("outbox cleanup limit is invalid")
	}
	command, err := s.connector.Pool().Exec(ctx, `
WITH candidates AS (
    SELECT message_id FROM core.outbox_messages
    WHERE published_at IS NOT NULL AND published_at <= $1
    ORDER BY published_at,message_id
    LIMIT $2
)
DELETE FROM core.outbox_messages AS message
USING candidates
WHERE message.message_id=candidates.message_id;`, before.UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup published outbox messages: %w", err)
	}
	return command.RowsAffected(), nil
}

var _ outbox.Source = (*outboxSource)(nil)
