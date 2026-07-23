package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
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

func (r *Repository) NameAvailable(
	ctx context.Context,
	storage filesystem.Code,
	folderID *file.FolderID,
	name string,
) error {
	if ctx == nil {
		return errors.New("file namespace context is nil")
	}
	var exists bool
	if err := r.connector.Pool().QueryRow(ctx, namespaceQuery,
		storage,
		folderID,
		name,
		nil,
		nil,
	).Scan(&exists); err != nil {
		return fmt.Errorf("query file namespace: %w", err)
	}
	if exists {
		return file.ErrConflict
	}
	return nil
}

func (r *Repository) CreateFolder(
	ctx context.Context,
	item file.Folder,
) (file.Folder, error) {
	if ctx == nil {
		return file.Folder{}, errors.New("create file folder context is nil")
	}
	tx, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return file.Folder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMutation(ctx, tx); err != nil {
		return file.Folder{}, err
	}
	if err := lockNamespace(ctx, tx, item.Storage, item.ParentID, item.Name); err != nil {
		return file.Folder{}, err
	}
	if err := ensureNamespaceAvailable(
		ctx, tx, item.Storage, item.ParentID, item.Name, nil, nil,
	); err != nil {
		return file.Folder{}, err
	}

	result, err := scanFolder(tx.QueryRow(ctx, `
INSERT INTO core.file_folders (parent_id, storage, name)
VALUES ($1, $2, $3)
RETURNING id, parent_id, storage, name, created_at, updated_at;
`, item.ParentID, item.Storage, item.Name))
	if err != nil {
		return file.Folder{}, translateError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return file.Folder{}, translateError(err)
	}
	return result, nil
}

func (r *Repository) FolderByID(
	ctx context.Context,
	id file.FolderID,
) (file.Folder, error) {
	if ctx == nil {
		return file.Folder{}, errors.New("get file folder context is nil")
	}
	result, err := scanFolder(r.connector.Pool().QueryRow(ctx, `
SELECT id, parent_id, storage, name, created_at, updated_at
FROM core.file_folders
WHERE id = $1;
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return file.Folder{}, file.ErrFolderNotFound
	}
	if err != nil {
		return file.Folder{}, fmt.Errorf("query file folder %d: %w", id, err)
	}
	return result, nil
}

func (r *Repository) ListFolders(
	ctx context.Context,
	storage filesystem.Code,
	parentID *file.FolderID,
) ([]file.Folder, error) {
	if ctx == nil {
		return nil, errors.New("list file folders context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, `
SELECT id, parent_id, storage, name, created_at, updated_at
FROM core.file_folders
WHERE storage = $1
  AND parent_id IS NOT DISTINCT FROM $2
ORDER BY name, id;
`, storage, parentID)
	if err != nil {
		return nil, fmt.Errorf("query file folders: %w", err)
	}
	defer rows.Close()

	var result []file.Folder
	for rows.Next() {
		item, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) CreateFile(
	ctx context.Context,
	item file.File,
) (file.File, error) {
	if ctx == nil {
		return file.File{}, errors.New("create file context is nil")
	}
	checksum, err := hex.DecodeString(item.ChecksumSHA256)
	if err != nil || len(checksum) != sha256Size {
		return file.File{}, errors.New("file checksum SHA-256 is invalid")
	}

	tx, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return file.File{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMutation(ctx, tx); err != nil {
		return file.File{}, err
	}
	if err := lockNamespace(ctx, tx, item.Storage, item.FolderID, item.Name); err != nil {
		return file.File{}, err
	}
	if err := ensureNamespaceAvailable(
		ctx, tx, item.Storage, item.FolderID, item.Name, nil, nil,
	); err != nil {
		return file.File{}, err
	}

	result, err := scanFile(tx.QueryRow(ctx, `
INSERT INTO core.files
(
    folder_id, storage, name, mime_type, size,
    checksum_sha256, path, parent_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING
    id, folder_id, storage, name, mime_type, size,
    checksum_sha256, path, parent_id, created_at, updated_at;
`,
		item.FolderID,
		item.Storage,
		item.Name,
		item.MIMEType,
		item.Size,
		checksum,
		item.Path,
		item.ParentID,
	))
	if err != nil {
		return file.File{}, translateError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return file.File{}, translateError(err)
	}
	return result, nil
}

func (r *Repository) FileByID(
	ctx context.Context,
	id file.ID,
) (file.File, error) {
	if ctx == nil {
		return file.File{}, errors.New("get file context is nil")
	}
	result, err := scanFile(r.connector.Pool().QueryRow(ctx, fileSelect+`
WHERE id = $1;
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return file.File{}, file.ErrNotFound
	}
	if err != nil {
		return file.File{}, fmt.Errorf("query file %d: %w", id, err)
	}
	return result, nil
}

func (r *Repository) ListFiles(
	ctx context.Context,
	storage filesystem.Code,
	folderID *file.FolderID,
) ([]file.File, error) {
	if ctx == nil {
		return nil, errors.New("list files context is nil")
	}
	rows, err := r.connector.Pool().Query(ctx, fileSelect+`
WHERE storage = $1
  AND folder_id IS NOT DISTINCT FROM $2
ORDER BY name, id;
`, storage, folderID)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()
	return scanFiles(rows)
}

func (r *Repository) MoveFile(
	ctx context.Context,
	id file.ID,
	folderID *file.FolderID,
) (file.File, error) {
	if ctx == nil {
		return file.File{}, errors.New("move file context is nil")
	}
	tx, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return file.File{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMutation(ctx, tx); err != nil {
		return file.File{}, err
	}
	current, err := scanFile(tx.QueryRow(ctx, fileSelect+`
WHERE id = $1
FOR UPDATE;
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return file.File{}, file.ErrNotFound
	}
	if err != nil {
		return file.File{}, err
	}
	if err := lockNamespace(ctx, tx, current.Storage, folderID, current.Name); err != nil {
		return file.File{}, err
	}
	if err := ensureNamespaceAvailable(
		ctx, tx, current.Storage, folderID, current.Name, nil, &id,
	); err != nil {
		return file.File{}, err
	}

	result, err := scanFile(tx.QueryRow(ctx, `
UPDATE core.files
SET folder_id = $2, updated_at = now()
WHERE id = $1
RETURNING
    id, folder_id, storage, name, mime_type, size,
    checksum_sha256, path, parent_id, created_at, updated_at;
`, id, folderID))
	if err != nil {
		return file.File{}, translateError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return file.File{}, translateError(err)
	}
	return result, nil
}

func (r *Repository) MoveFolder(
	ctx context.Context,
	id file.FolderID,
	parentID *file.FolderID,
) (file.Folder, error) {
	if ctx == nil {
		return file.Folder{}, errors.New("move file folder context is nil")
	}
	tx, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return file.Folder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMutation(ctx, tx); err != nil {
		return file.Folder{}, err
	}
	current, err := scanFolder(tx.QueryRow(ctx, `
SELECT id, parent_id, storage, name, created_at, updated_at
FROM core.file_folders
WHERE id = $1
FOR UPDATE;
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return file.Folder{}, file.ErrFolderNotFound
	}
	if err != nil {
		return file.Folder{}, err
	}
	if parentID != nil {
		var createsCycle bool
		if err := tx.QueryRow(ctx, `
WITH RECURSIVE ancestors AS
(
    SELECT id, parent_id
    FROM core.file_folders
    WHERE id = $2
    UNION ALL
    SELECT parent.id, parent.parent_id
    FROM core.file_folders AS parent
    JOIN ancestors AS child ON child.parent_id = parent.id
)
SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $1);
`, id, parentID).Scan(&createsCycle); err != nil {
			return file.Folder{}, err
		}
		if createsCycle {
			return file.Folder{}, file.ErrInvalidTree
		}
	}
	if err := lockNamespace(
		ctx, tx, current.Storage, parentID, current.Name,
	); err != nil {
		return file.Folder{}, err
	}
	if err := ensureNamespaceAvailable(
		ctx, tx, current.Storage, parentID, current.Name, &id, nil,
	); err != nil {
		return file.Folder{}, err
	}

	result, err := scanFolder(tx.QueryRow(ctx, `
UPDATE core.file_folders
SET parent_id = $2, updated_at = now()
WHERE id = $1
RETURNING id, parent_id, storage, name, created_at, updated_at;
`, id, parentID))
	if err != nil {
		return file.Folder{}, translateError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return file.Folder{}, translateError(err)
	}
	return result, nil
}

func (r *Repository) DeleteFile(
	ctx context.Context,
	id file.ID,
	deletePhysical file.DeletePhysical,
) error {
	if ctx == nil {
		return errors.New("delete file context is nil")
	}
	if deletePhysical == nil {
		return errors.New("physical file deleter is nil")
	}
	tx, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMutation(ctx, tx); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
WITH RECURSIVE tree AS
(
    SELECT id
    FROM core.files
    WHERE id = $1
    UNION ALL
    SELECT child.id
    FROM core.files AS child
    JOIN tree AS parent ON child.parent_id = parent.id
)
SELECT
    item.id, item.folder_id, item.storage, item.name,
    item.mime_type, item.size, item.checksum_sha256,
    item.path, item.parent_id, item.created_at, item.updated_at
FROM core.files AS item
JOIN tree ON tree.id = item.id
ORDER BY item.id
FOR UPDATE OF item;
`, id)
	if err != nil {
		return err
	}
	items, err := scanFiles(rows)
	rows.Close()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return file.ErrNotFound
	}
	if err := deletePhysical(ctx, items); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.files WHERE id = $1;`, id); err != nil {
		return translateError(err)
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteFolder(
	ctx context.Context,
	id file.FolderID,
	deletePhysical file.DeletePhysical,
) error {
	if ctx == nil {
		return errors.New("delete file folder context is nil")
	}
	if deletePhysical == nil {
		return errors.New("physical file deleter is nil")
	}
	tx, err := r.connector.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockMutation(ctx, tx); err != nil {
		return err
	}
	folderRows, err := tx.Query(ctx, `
WITH RECURSIVE tree AS
(
    SELECT id
    FROM core.file_folders
    WHERE id = $1
    UNION ALL
    SELECT child.id
    FROM core.file_folders AS child
    JOIN tree AS parent ON child.parent_id = parent.id
)
SELECT item.id
FROM core.file_folders AS item
JOIN tree ON tree.id = item.id
ORDER BY item.id
FOR UPDATE OF item;
`, id)
	if err != nil {
		return err
	}
	folderCount := 0
	for folderRows.Next() {
		var lockedID file.FolderID
		if err := folderRows.Scan(&lockedID); err != nil {
			folderRows.Close()
			return err
		}
		folderCount++
	}
	err = folderRows.Err()
	folderRows.Close()
	if err != nil {
		return err
	}
	if folderCount == 0 {
		return file.ErrFolderNotFound
	}

	rows, err := tx.Query(ctx, `
WITH RECURSIVE folder_tree AS
(
    SELECT id
    FROM core.file_folders
    WHERE id = $1
    UNION ALL
    SELECT child.id
    FROM core.file_folders AS child
    JOIN folder_tree AS parent ON child.parent_id = parent.id
),
file_tree AS
(
    SELECT item.id
    FROM core.files AS item
    WHERE item.folder_id IN (SELECT id FROM folder_tree)
    UNION
    SELECT child.id
    FROM core.files AS child
    JOIN file_tree AS parent ON child.parent_id = parent.id
)
SELECT
    item.id, item.folder_id, item.storage, item.name,
    item.mime_type, item.size, item.checksum_sha256,
    item.path, item.parent_id, item.created_at, item.updated_at
FROM core.files AS item
JOIN file_tree ON file_tree.id = item.id
ORDER BY item.id
FOR UPDATE OF item;
`, id)
	if err != nil {
		return err
	}
	items, err := scanFiles(rows)
	rows.Close()
	if err != nil {
		return err
	}
	if err := deletePhysical(ctx, items); err != nil {
		return err
	}
	if _, err := tx.Exec(
		ctx,
		`DELETE FROM core.file_folders WHERE id = $1;`,
		id,
	); err != nil {
		return translateError(err)
	}
	return tx.Commit(ctx)
}

const (
	sha256Size = 32
	fileSelect = `
SELECT
    id, folder_id, storage, name, mime_type, size,
    checksum_sha256, path, parent_id, created_at, updated_at
FROM core.files
`
	namespaceQuery = `
SELECT EXISTS
(
    SELECT 1
    FROM core.file_folders
    WHERE storage = $1
      AND parent_id IS NOT DISTINCT FROM $2
      AND name = $3
      AND ($4::bigint IS NULL OR id <> $4)
    UNION ALL
    SELECT 1
    FROM core.files
    WHERE storage = $1
      AND folder_id IS NOT DISTINCT FROM $2
      AND name = $3
      AND ($5::bigint IS NULL OR id <> $5)
);
`
)

type rowScanner interface {
	Scan(...any) error
}

func scanFolder(scanner rowScanner) (file.Folder, error) {
	var (
		result   file.Folder
		parentID *int64
	)
	if err := scanner.Scan(
		&result.ID,
		&parentID,
		&result.Storage,
		&result.Name,
		&result.CreatedAt,
		&result.UpdatedAt,
	); err != nil {
		return file.Folder{}, err
	}
	if parentID != nil {
		value := file.FolderID(*parentID)
		result.ParentID = &value
	}
	return result, nil
}

func scanFile(scanner rowScanner) (file.File, error) {
	var (
		result   file.File
		folderID *int64
		parentID *int64
		checksum []byte
	)
	if err := scanner.Scan(
		&result.ID,
		&folderID,
		&result.Storage,
		&result.Name,
		&result.MIMEType,
		&result.Size,
		&checksum,
		&result.Path,
		&parentID,
		&result.CreatedAt,
		&result.UpdatedAt,
	); err != nil {
		return file.File{}, err
	}
	if len(checksum) != sha256Size {
		return file.File{}, errors.New("stored file checksum SHA-256 is invalid")
	}
	result.ChecksumSHA256 = hex.EncodeToString(checksum)
	if folderID != nil {
		value := file.FolderID(*folderID)
		result.FolderID = &value
	}
	if parentID != nil {
		value := file.ID(*parentID)
		result.ParentID = &value
	}
	return result, nil
}

func scanFiles(rows pgx.Rows) ([]file.File, error) {
	var result []file.File
	for rows.Next() {
		item, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func lockNamespace(
	ctx context.Context,
	tx pgx.Tx,
	storage filesystem.Code,
	parentID *file.FolderID,
	name string,
) error {
	parent := "root"
	if parentID != nil {
		parent = strconv.FormatInt(int64(*parentID), 10)
	}
	key := string(storage) + "\x1f" + parent + "\x1f" + name
	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0));`,
		key,
	); err != nil {
		return fmt.Errorf("lock file namespace: %w", err)
	}
	return nil
}

func lockMutation(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('core.filesystem.mutation', 0));`,
	); err != nil {
		return fmt.Errorf("lock filesystem mutation: %w", err)
	}
	return nil
}

func ensureNamespaceAvailable(
	ctx context.Context,
	tx pgx.Tx,
	storage filesystem.Code,
	parentID *file.FolderID,
	name string,
	excludeFolder *file.FolderID,
	excludeFile *file.ID,
) error {
	var exists bool
	if err := tx.QueryRow(
		ctx,
		namespaceQuery,
		storage,
		parentID,
		name,
		excludeFolder,
		excludeFile,
	).Scan(&exists); err != nil {
		return fmt.Errorf("query file namespace: %w", err)
	}
	if exists {
		return file.ErrConflict
	}
	return nil
}

func translateError(err error) error {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return err
	}
	switch postgresError.Code {
	case pgerrcode.UniqueViolation:
		return file.ErrConflict
	case pgerrcode.ForeignKeyViolation:
		return file.ErrInvalidReference
	default:
		return err
	}
}

var _ file.Repository = (*Repository)(nil)
