package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type Repository struct {
	connector *connectorpostgres.Connector
}

func NewRepository(
	connector *connectorpostgres.Connector,
) (*Repository, error) {
	if connector == nil {
		return nil, errors.New("group postgres connector is nil")
	}
	if connector.Pool() == nil {
		return nil, errors.New("group postgres pool is nil")
	}
	return &Repository{connector: connector}, nil
}

func (r *Repository) Create(
	ctx context.Context,
	actorID *security.UserID,
	item group.Group,
) (group.Group, error) {
	if ctx == nil {
		return group.Group{}, errors.New("create group context is nil")
	}
	result, err := scanGroup(r.connector.Pool().QueryRow(ctx, `
INSERT INTO core.groups
(
    code,
    name,
    is_super,
    created_by,
    updated_by
)
VALUES ($1, $2, $3, $4, $4)
RETURNING
    id, code, name, is_super,
    created_at, updated_at, created_by, updated_by;
`, item.Code, item.Name, item.IsSuper, actorID))
	if err != nil {
		return group.Group{}, translateError(err)
	}
	return result, nil
}

func (r *Repository) ByID(
	ctx context.Context,
	id group.ID,
) (group.Group, error) {
	return r.queryOne(ctx, "id = $1", id)
}

func (r *Repository) ByCode(
	ctx context.Context,
	code string,
) (group.Group, error) {
	return r.queryOne(ctx, "code = $1", code)
}

func (r *Repository) queryOne(
	ctx context.Context,
	predicate string,
	value any,
) (group.Group, error) {
	if ctx == nil {
		return group.Group{}, errors.New("get group context is nil")
	}
	result, err := scanGroup(r.connector.Pool().QueryRow(
		ctx,
		`
SELECT
    id, code, name, is_super,
    created_at, updated_at, created_by, updated_by
FROM core.groups
WHERE `+predicate+`;
`,
		value,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return group.Group{}, group.ErrNotFound
	}
	if err != nil {
		return group.Group{}, err
	}
	return result, nil
}

func (r *Repository) List(
	ctx context.Context,
) ([]group.Group, error) {
	if ctx == nil {
		return nil, errors.New("list groups context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, `
SELECT
    id, code, name, is_super,
    created_at, updated_at, created_by, updated_by
FROM core.groups
ORDER BY code;
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]group.Group, 0)
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) Update(
	ctx context.Context,
	actorID *security.UserID,
	item group.Group,
) (group.Group, error) {
	if ctx == nil {
		return group.Group{}, errors.New("update group context is nil")
	}
	result, err := scanGroup(r.connector.Pool().QueryRow(ctx, `
UPDATE core.groups
SET
    name = $2,
    is_super = $3,
    updated_at = now(),
    updated_by = $4
WHERE id = $1
RETURNING
    id, code, name, is_super,
    created_at, updated_at, created_by, updated_by;
`, item.ID, item.Name, item.IsSuper, actorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return group.Group{}, group.ErrNotFound
	}
	if err != nil {
		return group.Group{}, translateError(err)
	}
	return result, nil
}

func (r *Repository) Delete(
	ctx context.Context,
	id group.ID,
) error {
	if ctx == nil {
		return errors.New("delete group context is nil")
	}
	tag, err := r.connector.Pool().Exec(ctx, `
DELETE FROM core.groups
WHERE id = $1;
`, id)
	if err != nil {
		return translateError(err)
	}
	if tag.RowsAffected() == 0 {
		return group.ErrNotFound
	}
	return nil
}

func (r *Repository) AddUser(
	ctx context.Context,
	actorID *security.UserID,
	groupID group.ID,
	userID security.UserID,
) (group.Membership, error) {
	if ctx == nil {
		return group.Membership{}, errors.New("add group user context is nil")
	}
	var item group.Membership
	err := r.connector.Pool().QueryRow(ctx, `
INSERT INTO core.user_groups
(
    user_id,
    group_id,
    created_by,
    updated_by
)
SELECT
    u.id,
    g.id,
    $3,
    $3
FROM core.users u
CROSS JOIN core.groups g
WHERE u.id = $1
  AND u.deleted_at IS NULL
  AND g.id = $2
ON CONFLICT (user_id, group_id) DO UPDATE
SET
    updated_at = now(),
    updated_by = EXCLUDED.updated_by
RETURNING
    user_id, group_id,
    created_at, updated_at, created_by, updated_by;
`, userID, groupID, actorID).Scan(
		&item.UserID,
		&item.GroupID,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CreatedBy,
		&item.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return group.Membership{}, group.ErrInvalidReference
	}
	if err != nil {
		return group.Membership{}, translateError(err)
	}
	return item, nil
}

func (r *Repository) RemoveUser(
	ctx context.Context,
	groupID group.ID,
	userID security.UserID,
) error {
	if ctx == nil {
		return errors.New("remove group user context is nil")
	}
	_, err := r.connector.Pool().Exec(ctx, `
DELETE FROM core.user_groups
WHERE group_id = $1
  AND user_id = $2;
`, groupID, userID)
	return translateError(err)
}

func (r *Repository) Members(
	ctx context.Context,
	groupID group.ID,
) ([]group.Membership, error) {
	if ctx == nil {
		return nil, errors.New("list group members context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, `
SELECT
    user_id, group_id,
    created_at, updated_at, created_by, updated_by
FROM core.user_groups
WHERE group_id = $1
ORDER BY user_id;
`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]group.Membership, 0)
	for rows.Next() {
		var item group.Membership
		if err := rows.Scan(
			&item.UserID,
			&item.GroupID,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.CreatedBy,
			&item.UpdatedBy,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) GroupsForUser(
	ctx context.Context,
	userID security.UserID,
) ([]group.Group, error) {
	if ctx == nil {
		return nil, errors.New("list user groups context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, `
SELECT
    g.id, g.code, g.name, g.is_super,
    g.created_at, g.updated_at, g.created_by, g.updated_by
FROM core.user_groups ug
JOIN core.groups g ON g.id = ug.group_id
WHERE ug.user_id = $1
ORDER BY g.code;
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]group.Group, 0)
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) GrantPermission(
	ctx context.Context,
	actorID *security.UserID,
	groupID group.ID,
	code permission.Code,
) (group.PermissionGrant, error) {
	if ctx == nil {
		return group.PermissionGrant{}, errors.New(
			"grant group permission context is nil",
		)
	}
	var item group.PermissionGrant
	err := r.connector.Pool().QueryRow(ctx, `
INSERT INTO core.group_permissions
(
    group_id,
    permission_code,
    created_by,
    updated_by
)
VALUES ($1, $2, $3, $3)
ON CONFLICT (group_id, permission_code) DO UPDATE
SET
    updated_at = now(),
    updated_by = EXCLUDED.updated_by
RETURNING
    group_id, permission_code,
    created_at, updated_at, created_by, updated_by;
`, groupID, code, actorID).Scan(
		&item.GroupID,
		&item.Permission,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CreatedBy,
		&item.UpdatedBy,
	)
	if err != nil {
		return group.PermissionGrant{}, translateError(err)
	}
	return item, nil
}

func (r *Repository) RevokePermission(
	ctx context.Context,
	groupID group.ID,
	code permission.Code,
) error {
	if ctx == nil {
		return errors.New("revoke group permission context is nil")
	}
	_, err := r.connector.Pool().Exec(ctx, `
DELETE FROM core.group_permissions
WHERE group_id = $1
  AND permission_code = $2;
`, groupID, code)
	return translateError(err)
}

func (r *Repository) Permissions(
	ctx context.Context,
	groupID group.ID,
) ([]group.PermissionGrant, error) {
	if ctx == nil {
		return nil, errors.New("list group permissions context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, `
SELECT
    group_id, permission_code,
    created_at, updated_at, created_by, updated_by
FROM core.group_permissions
WHERE group_id = $1
ORDER BY permission_code;
`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]group.PermissionGrant, 0)
	for rows.Next() {
		var item group.PermissionGrant
		if err := rows.Scan(
			&item.GroupID,
			&item.Permission,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.CreatedBy,
			&item.UpdatedBy,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

type rowScanner interface {
	Scan(...any) error
}

func scanGroup(scanner rowScanner) (group.Group, error) {
	var item group.Group
	err := scanner.Scan(
		&item.ID,
		&item.Code,
		&item.Name,
		&item.IsSuper,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CreatedBy,
		&item.UpdatedBy,
	)
	return item, err
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
		return fmt.Errorf("%w: %s", group.ErrConflict, err)
	case pgerrcode.ForeignKeyViolation, pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %s", group.ErrInvalidReference, err)
	default:
		return err
	}
}

var _ group.Repository = (*Repository)(nil)
