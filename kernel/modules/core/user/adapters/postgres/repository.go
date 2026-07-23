package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres/medialock"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/security"
)

type Repository struct {
	connector *connectorpostgres.Connector
}

func NewRepository(
	connector *connectorpostgres.Connector,
) (*Repository, error) {
	if connector == nil {
		return nil, errors.New("user postgres connector is nil")
	}
	if connector.Pool() == nil {
		return nil, errors.New("user postgres pool is nil")
	}
	return &Repository{connector: connector}, nil
}

func (r *Repository) Create(
	ctx context.Context,
	actorID *security.UserID,
	record user.Record,
	validate user.ValidateAvatarMedia,
) (_ user.Record, resultErr error) {
	if ctx == nil {
		return user.Record{}, errors.New("create user context is nil")
	}
	transaction, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return user.Record{}, fmt.Errorf("begin user create: %w", err)
	}
	defer rollbackOnError(transaction, &resultErr)

	if record.AvatarMediaID != nil {
		if validate == nil {
			return user.Record{}, errors.New("avatar validator is nil")
		}
		if err := medialock.Lock(
			ctx,
			transaction,
			*record.AvatarMediaID,
		); err != nil {
			return user.Record{}, err
		}
		if err := ensureMediaAvailable(
			ctx,
			transaction,
			*record.AvatarMediaID,
			0,
		); err != nil {
			return user.Record{}, err
		}
		if err := validate(ctx, *record.AvatarMediaID); err != nil {
			return user.Record{}, err
		}
	}

	created, err := scanRecord(transaction.QueryRow(ctx, `
INSERT INTO core.users
(
    login,
    email,
    password_hash,
    name,
    last_name,
    middle_name,
    phone,
    avatar_media_id,
    created_by,
    updated_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
RETURNING
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by;
`,
		record.Login,
		record.Email,
		record.PasswordHash,
		record.Name,
		record.LastName,
		record.MiddleName,
		record.Phone,
		record.AvatarMediaID,
		actorID,
	))
	if err != nil {
		return user.Record{}, translateError(err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return user.Record{}, translateError(err)
	}
	return created, nil
}

func (r *Repository) ByID(
	ctx context.Context,
	id user.ID,
) (user.Record, error) {
	if ctx == nil {
		return user.Record{}, errors.New("get user context is nil")
	}
	record, err := scanRecord(r.connector.Pool().QueryRow(ctx, `
SELECT
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by
FROM core.users
WHERE id = $1;
`, id))
	return translateRecordResult(record, err)
}

func (r *Repository) ByIdentifier(
	ctx context.Context,
	identifier string,
) (user.Record, error) {
	if ctx == nil {
		return user.Record{}, errors.New("get auth user context is nil")
	}
	record, err := scanRecord(r.connector.Pool().QueryRow(ctx, `
SELECT
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by
FROM core.users
WHERE login = $1 OR email = $1
LIMIT 1;
`, identifier))
	return translateRecordResult(record, err)
}

func (r *Repository) List(
	ctx context.Context,
) ([]user.Record, error) {
	if ctx == nil {
		return nil, errors.New("list users context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, `
SELECT
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by
FROM core.users
ORDER BY id;
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]user.Record, 0)
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (r *Repository) Update(
	ctx context.Context,
	actorID *security.UserID,
	_ user.Record,
	next user.Record,
	validate user.ValidateAvatarMedia,
) (_ user.Record, resultErr error) {
	if ctx == nil {
		return user.Record{}, errors.New("update user context is nil")
	}
	transaction, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return user.Record{}, fmt.Errorf("begin user update: %w", err)
	}
	defer rollbackOnError(transaction, &resultErr)

	locked, err := scanRecord(transaction.QueryRow(ctx, `
SELECT
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by
FROM core.users
WHERE id = $1
FOR UPDATE;
`, next.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return user.Record{}, user.ErrNotFound
	}
	if err != nil {
		return user.Record{}, err
	}

	avatarChanged := !sameMediaID(
		locked.AvatarMediaID,
		next.AvatarMediaID,
	)
	if avatarChanged {
		if validate == nil && next.AvatarMediaID != nil {
			return user.Record{}, errors.New("avatar validator is nil")
		}
		ids := make([]media.ID, 0, 2)
		if locked.AvatarMediaID != nil {
			ids = append(ids, *locked.AvatarMediaID)
		}
		if next.AvatarMediaID != nil {
			ids = append(ids, *next.AvatarMediaID)
		}
		if err := medialock.Lock(ctx, transaction, ids...); err != nil {
			return user.Record{}, err
		}
		if next.AvatarMediaID != nil {
			if err := ensureMediaAvailable(
				ctx,
				transaction,
				*next.AvatarMediaID,
				next.ID,
			); err != nil {
				return user.Record{}, err
			}
			if err := validate(ctx, *next.AvatarMediaID); err != nil {
				return user.Record{}, err
			}
		}
	}

	updated, err := scanRecord(transaction.QueryRow(ctx, `
UPDATE core.users
SET
    login = $2,
    email = $3,
    name = $4,
    last_name = $5,
    middle_name = $6,
    phone = $7,
    avatar_media_id = $8,
    updated_at = now(),
    updated_by = $9
WHERE id = $1
RETURNING
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by;
`,
		next.ID,
		next.Login,
		next.Email,
		next.Name,
		next.LastName,
		next.MiddleName,
		next.Phone,
		next.AvatarMediaID,
		actorID,
	))
	if err != nil {
		return user.Record{}, translateError(err)
	}

	if avatarChanged && locked.AvatarMediaID != nil {
		if _, err := transaction.Exec(ctx, `
DELETE FROM core.media
WHERE id = $1;
`, *locked.AvatarMediaID); err != nil {
			return user.Record{}, translateError(err)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return user.Record{}, translateError(err)
	}
	return updated, nil
}

func (r *Repository) ChangePassword(
	ctx context.Context,
	actorID *security.UserID,
	id user.ID,
	passwordHash string,
) (user.Record, error) {
	if ctx == nil {
		return user.Record{}, errors.New("change password context is nil")
	}
	record, err := scanRecord(r.connector.Pool().QueryRow(ctx, `
UPDATE core.users
SET
    password_hash = $2,
    updated_at = now(),
    updated_by = $3
WHERE id = $1
RETURNING
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by;
`, id, passwordHash, actorID))
	return translateRecordResult(record, translateError(err))
}

func (r *Repository) RecordLogin(
	ctx context.Context,
	id user.ID,
	passwordHash *string,
) (user.Record, error) {
	if ctx == nil {
		return user.Record{}, errors.New("record login context is nil")
	}
	record, err := scanRecord(r.connector.Pool().QueryRow(ctx, `
UPDATE core.users
SET
    password_hash = COALESCE($2, password_hash),
    last_login_at = now(),
    updated_at = now(),
    updated_by = id
WHERE id = $1
  AND deleted_at IS NULL
RETURNING
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by;
`, id, passwordHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return user.Record{}, user.ErrInvalidCredentials
	}
	return record, translateError(err)
}

func (r *Repository) Delete(
	ctx context.Context,
	actorID *security.UserID,
	id user.ID,
) (user.Record, error) {
	if ctx == nil {
		return user.Record{}, errors.New("delete user context is nil")
	}
	record, err := scanRecord(r.connector.Pool().QueryRow(ctx, `
UPDATE core.users
SET
    deleted_at = COALESCE(deleted_at, now()),
    deleted_by = COALESCE(deleted_by, $2),
    updated_at = now(),
    updated_by = $2
WHERE id = $1
RETURNING
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by;
`, id, actorID))
	return translateRecordResult(record, translateError(err))
}

func (r *Repository) Restore(
	ctx context.Context,
	actorID *security.UserID,
	id user.ID,
) (user.Record, error) {
	if ctx == nil {
		return user.Record{}, errors.New("restore user context is nil")
	}
	record, err := scanRecord(r.connector.Pool().QueryRow(ctx, `
UPDATE core.users
SET
    deleted_at = NULL,
    deleted_by = NULL,
    updated_at = now(),
    updated_by = $2
WHERE id = $1
RETURNING
    id, login, email, password_hash, name,
    last_name, middle_name, phone, avatar_media_id,
    last_login_at, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by;
`, id, actorID))
	return translateRecordResult(record, translateError(err))
}

func ensureMediaAvailable(
	ctx context.Context,
	transaction pgx.Tx,
	id media.ID,
	excludeUserID user.ID,
) error {
	var (
		exists       bool
		resourceUsed bool
		userUsed     bool
	)
	err := transaction.QueryRow(ctx, `
SELECT
    EXISTS (
        SELECT 1 FROM core.media WHERE id = $1
    ),
    EXISTS (
        SELECT 1
        FROM core.resources
        WHERE image_media_id = $1
    ),
    EXISTS (
        SELECT 1
        FROM core.users
        WHERE avatar_media_id = $1
          AND id <> $2
    );
`, id, excludeUserID).Scan(
		&exists,
		&resourceUsed,
		&userUsed,
	)
	if err != nil {
		return fmt.Errorf("query avatar media availability: %w", err)
	}
	if !exists {
		return user.ErrInvalidReference
	}
	if resourceUsed || userUsed {
		return media.ErrAlreadyAttached
	}
	return nil
}

func sameMediaID(left, right *media.ID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

type rowScanner interface {
	Scan(...any) error
}

func scanRecord(scanner rowScanner) (user.Record, error) {
	var record user.Record
	err := scanner.Scan(
		&record.ID,
		&record.Login,
		&record.Email,
		&record.PasswordHash,
		&record.Name,
		&record.LastName,
		&record.MiddleName,
		&record.Phone,
		&record.AvatarMediaID,
		&record.LastLoginAt,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
		&record.CreatedBy,
		&record.UpdatedBy,
		&record.DeletedBy,
	)
	return record, err
}

func translateRecordResult(
	record user.Record,
	err error,
) (user.Record, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return user.Record{}, user.ErrNotFound
	}
	if err != nil {
		return user.Record{}, err
	}
	return record, nil
}

func rollbackOnError(
	transaction pgx.Tx,
	resultErr *error,
) func() {
	return func() {
		if *resultErr == nil {
			return
		}
		err := transaction.Rollback(context.Background())
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			*resultErr = errors.Join(*resultErr, err)
		}
	}
}

func translateError(err error) error {
	if err == nil {
		return nil
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return err
	}
	switch postgresError.Code {
	case pgerrcode.UniqueViolation:
		return fmt.Errorf("%w: %s", user.ErrConflict, err)
	case pgerrcode.ForeignKeyViolation, pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %s", user.ErrInvalidReference, err)
	default:
		return err
	}
}

var _ user.Repository = (*Repository)(nil)
