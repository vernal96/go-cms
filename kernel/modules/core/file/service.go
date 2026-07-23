package file

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel/filesystem"
)

type DiskResolver interface {
	Disk(filesystem.Code) (filesystem.Disk, bool)
}

type service struct {
	repository Repository
	disks      DiskResolver
}

func NewService(
	repository Repository,
	disks DiskResolver,
) (Service, error) {
	if repository == nil {
		return nil, errors.New("file repository is nil")
	}
	if disks == nil {
		return nil, errors.New("filesystem disk resolver is nil")
	}
	return &service{repository: repository, disks: disks}, nil
}

func (s *service) CreateFolder(
	ctx context.Context,
	input CreateFolderInput,
) (Folder, error) {
	if err := validateContext(ctx, "create file folder"); err != nil {
		return Folder{}, err
	}
	name, err := normalizeName(input.Name)
	if err != nil {
		return Folder{}, err
	}
	if _, err := s.disk(input.Storage); err != nil {
		return Folder{}, err
	}
	if input.ParentID != nil {
		parent, err := s.repository.FolderByID(ctx, *input.ParentID)
		if err != nil {
			return Folder{}, fmt.Errorf("get parent file folder: %w", err)
		}
		if parent.Storage != input.Storage {
			return Folder{}, ErrStorageMismatch
		}
	}

	result, err := s.repository.CreateFolder(ctx, Folder{
		ParentID: cloneFolderID(input.ParentID),
		Storage:  input.Storage,
		Name:     name,
	})
	if err != nil {
		return Folder{}, fmt.Errorf("create file folder: %w", err)
	}
	return CloneFolder(result), nil
}

func (s *service) GetFolder(
	ctx context.Context,
	id FolderID,
) (Folder, error) {
	if err := validateContext(ctx, "get file folder"); err != nil {
		return Folder{}, err
	}
	if id <= 0 {
		return Folder{}, errors.New("file folder id is invalid")
	}
	result, err := s.repository.FolderByID(ctx, id)
	if err != nil {
		return Folder{}, fmt.Errorf("get file folder %d: %w", id, err)
	}
	return CloneFolder(result), nil
}

func (s *service) ListFolder(
	ctx context.Context,
	storage filesystem.Code,
	folderID *FolderID,
) (Listing, error) {
	if err := validateContext(ctx, "list file folder"); err != nil {
		return Listing{}, err
	}
	if _, err := s.disk(storage); err != nil {
		return Listing{}, err
	}

	var current *Folder
	if folderID != nil {
		item, err := s.repository.FolderByID(ctx, *folderID)
		if err != nil {
			return Listing{}, fmt.Errorf("get listed file folder: %w", err)
		}
		if item.Storage != storage {
			return Listing{}, ErrStorageMismatch
		}
		item = CloneFolder(item)
		current = &item
	}

	folders, err := s.repository.ListFolders(ctx, storage, folderID)
	if err != nil {
		return Listing{}, fmt.Errorf("list file folders: %w", err)
	}
	files, err := s.repository.ListFiles(ctx, storage, folderID)
	if err != nil {
		return Listing{}, fmt.Errorf("list files: %w", err)
	}
	return Listing{Folder: current, Folders: folders, Files: files}, nil
}

func (s *service) Upload(
	ctx context.Context,
	input UploadInput,
) (File, error) {
	if err := validateContext(ctx, "upload file"); err != nil {
		return File{}, err
	}
	if input.Content == nil {
		return File{}, errors.New("file upload content is nil")
	}
	name, err := normalizeName(input.Name)
	if err != nil {
		return File{}, err
	}
	disk, err := s.disk(input.Storage)
	if err != nil {
		return File{}, err
	}
	if input.FolderID != nil {
		folder, err := s.repository.FolderByID(ctx, *input.FolderID)
		if err != nil {
			return File{}, fmt.Errorf("get upload file folder: %w", err)
		}
		if folder.Storage != input.Storage {
			return File{}, ErrStorageMismatch
		}
	}
	if input.ParentID != nil {
		if _, err := s.repository.FileByID(ctx, *input.ParentID); err != nil {
			return File{}, fmt.Errorf("get parent file: %w", err)
		}
	}
	if err := s.repository.NameAvailable(
		ctx,
		input.Storage,
		input.FolderID,
		name,
	); err != nil {
		return File{}, err
	}

	header := make([]byte, 512)
	count, readErr := io.ReadFull(input.Content, header)
	if readErr != nil &&
		!errors.Is(readErr, io.EOF) &&
		!errors.Is(readErr, io.ErrUnexpectedEOF) {
		return File{}, fmt.Errorf("read file header: %w", readErr)
	}
	header = header[:count]
	mimeType := http.DetectContentType(header)
	source := io.MultiReader(bytes.NewReader(header), input.Content)

	key, err := newObjectKey(time.Now().UTC())
	if err != nil {
		return File{}, err
	}
	hash := sha256.New()
	counter := &byteCounter{}
	measured := io.TeeReader(source, io.MultiWriter(hash, counter))
	if err := disk.PutNew(ctx, key, measured, mimeType); err != nil {
		return File{}, fmt.Errorf("store file on disk %q: %w", input.Storage, err)
	}

	item := File{
		FolderID:       cloneFolderID(input.FolderID),
		Storage:        input.Storage,
		Name:           name,
		MIMEType:       mimeType,
		Size:           counter.count,
		ChecksumSHA256: hex.EncodeToString(hash.Sum(nil)),
		Path:           key,
		ParentID:       cloneID(input.ParentID),
	}
	result, createErr := s.repository.CreateFile(ctx, item)
	if createErr != nil {
		cleanupErr := disk.Delete(context.WithoutCancel(ctx), key)
		return File{}, errors.Join(
			fmt.Errorf("register uploaded file: %w", createErr),
			wrapCleanupError(cleanupErr),
		)
	}
	return Clone(result), nil
}

func (s *service) GetFile(ctx context.Context, id ID) (File, error) {
	if err := validateContext(ctx, "get file"); err != nil {
		return File{}, err
	}
	return s.file(ctx, id)
}

func (s *service) Open(
	ctx context.Context,
	id ID,
) (OpenedFile, error) {
	if err := validateContext(ctx, "open file"); err != nil {
		return OpenedFile{}, err
	}
	item, err := s.file(ctx, id)
	if err != nil {
		return OpenedFile{}, err
	}
	disk, err := s.disk(item.Storage)
	if err != nil {
		return OpenedFile{}, err
	}
	body, err := disk.Open(ctx, item.Path)
	if err != nil {
		return OpenedFile{}, fmt.Errorf("open physical file %d: %w", id, err)
	}
	return OpenedFile{File: item, Body: body}, nil
}

func (s *service) OpenDelivery(
	ctx context.Context,
	id ID,
	authorization DeliveryAuthorization,
) (OpenedFile, error) {
	if err := validateContext(ctx, "open delivered file"); err != nil {
		return OpenedFile{}, err
	}
	item, err := s.file(ctx, id)
	if err != nil {
		return OpenedFile{}, err
	}
	disk, err := s.disk(item.Storage)
	if err != nil {
		return OpenedFile{}, err
	}
	if disk.Visibility() == filesystem.VisibilityPrivate {
		verifier, ok := disk.(filesystem.TemporaryURLVerifier)
		if !ok {
			return OpenedFile{}, ErrUnauthorized
		}
		err := verifier.VerifyTemporaryURL(
			reference(item),
			authorization.ExpiresAt,
			authorization.Signature,
		)
		if err != nil {
			return OpenedFile{}, ErrUnauthorized
		}
	}
	body, err := disk.Open(ctx, item.Path)
	if err != nil {
		return OpenedFile{}, fmt.Errorf("open delivered file %d: %w", id, err)
	}
	return OpenedFile{File: item, Body: body}, nil
}

func (s *service) MoveFile(
	ctx context.Context,
	input MoveFileInput,
) (File, error) {
	if err := validateContext(ctx, "move file"); err != nil {
		return File{}, err
	}
	item, err := s.file(ctx, input.ID)
	if err != nil {
		return File{}, err
	}
	if input.FolderID != nil {
		folder, err := s.repository.FolderByID(ctx, *input.FolderID)
		if err != nil {
			return File{}, fmt.Errorf("get target file folder: %w", err)
		}
		if folder.Storage != item.Storage {
			return File{}, ErrStorageMismatch
		}
	}
	result, err := s.repository.MoveFile(ctx, input.ID, input.FolderID)
	if err != nil {
		return File{}, fmt.Errorf("move file %d: %w", input.ID, err)
	}
	return Clone(result), nil
}

func (s *service) MoveFolder(
	ctx context.Context,
	input MoveFolderInput,
) (Folder, error) {
	if err := validateContext(ctx, "move file folder"); err != nil {
		return Folder{}, err
	}
	if input.ID <= 0 {
		return Folder{}, errors.New("file folder id is invalid")
	}
	item, err := s.repository.FolderByID(ctx, input.ID)
	if err != nil {
		return Folder{}, fmt.Errorf("get moved file folder: %w", err)
	}
	if input.ParentID != nil {
		if *input.ParentID == input.ID {
			return Folder{}, ErrInvalidTree
		}
		parent, err := s.repository.FolderByID(ctx, *input.ParentID)
		if err != nil {
			return Folder{}, fmt.Errorf("get target file folder: %w", err)
		}
		if parent.Storage != item.Storage {
			return Folder{}, ErrStorageMismatch
		}
	}
	result, err := s.repository.MoveFolder(ctx, input.ID, input.ParentID)
	if err != nil {
		return Folder{}, fmt.Errorf("move file folder %d: %w", input.ID, err)
	}
	return CloneFolder(result), nil
}

func (s *service) DeleteFile(ctx context.Context, id ID) error {
	if err := validateContext(ctx, "delete file"); err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("file id is invalid")
	}
	if err := s.repository.DeleteFile(ctx, id, s.deletePhysical); err != nil {
		return fmt.Errorf("delete file %d: %w", id, err)
	}
	return nil
}

func (s *service) DeleteFolder(
	ctx context.Context,
	id FolderID,
) error {
	if err := validateContext(ctx, "delete file folder"); err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("file folder id is invalid")
	}
	if err := s.repository.DeleteFolder(ctx, id, s.deletePhysical); err != nil {
		return fmt.Errorf("delete file folder %d: %w", id, err)
	}
	return nil
}

func (s *service) URL(ctx context.Context, id ID) (string, error) {
	if err := validateContext(ctx, "create file URL"); err != nil {
		return "", err
	}
	item, err := s.file(ctx, id)
	if err != nil {
		return "", err
	}
	disk, err := s.disk(item.Storage)
	if err != nil {
		return "", err
	}
	if disk.Visibility() != filesystem.VisibilityPublic {
		return "", filesystem.ErrInvalidVisibility
	}
	return disk.URL(ctx, reference(item))
}

func (s *service) TemporaryURL(
	ctx context.Context,
	id ID,
	expiresAt time.Time,
) (string, error) {
	if err := validateContext(ctx, "create temporary file URL"); err != nil {
		return "", err
	}
	if !expiresAt.After(time.Now()) {
		return "", errors.New("temporary URL expiration must be in the future")
	}
	item, err := s.file(ctx, id)
	if err != nil {
		return "", err
	}
	disk, err := s.disk(item.Storage)
	if err != nil {
		return "", err
	}
	if disk.Visibility() != filesystem.VisibilityPrivate {
		return "", filesystem.ErrInvalidVisibility
	}
	return disk.TemporaryURL(ctx, reference(item), expiresAt)
}

func (s *service) deletePhysical(
	ctx context.Context,
	items []File,
) error {
	var deleteErrors []error
	for _, item := range items {
		disk, err := s.disk(item.Storage)
		if err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf(
				"resolve disk for file %d: %w",
				item.ID,
				err,
			))
			continue
		}
		if err := disk.Delete(ctx, item.Path); err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf(
				"delete physical file %d: %w",
				item.ID,
				err,
			))
		}
	}
	return errors.Join(deleteErrors...)
}

func (s *service) file(ctx context.Context, id ID) (File, error) {
	if id <= 0 {
		return File{}, errors.New("file id is invalid")
	}
	item, err := s.repository.FileByID(ctx, id)
	if err != nil {
		return File{}, fmt.Errorf("get file %d: %w", id, err)
	}
	return Clone(item), nil
}

func (s *service) disk(code filesystem.Code) (filesystem.Disk, error) {
	if code == "" {
		return nil, errors.New("file storage is empty")
	}
	disk, exists := s.disks.Disk(code)
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrStorageNotFound, code)
	}
	return disk, nil
}

func reference(item File) filesystem.Reference {
	return filesystem.Reference{
		ID:   strconv.FormatInt(int64(item.ID), 10),
		Path: item.Path,
	}
}

func validateContext(ctx context.Context, operation string) error {
	if ctx == nil {
		return fmt.Errorf("%s context is nil", operation)
	}
	return ctx.Err()
}

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return "", errors.New("file name is empty")
	case value == ".", value == "..":
		return "", errors.New("file name is reserved")
	case strings.Contains(value, "/"):
		return "", errors.New("file name contains a path separator")
	case strings.ContainsRune(value, '\x00'):
		return "", errors.New("file name contains NUL")
	default:
		return value, nil
	}
}

func newObjectKey(now time.Time) (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate file object key: %w", err)
	}
	return path.Join(
		"objects",
		now.Format("2006"),
		now.Format("01"),
		hex.EncodeToString(random),
	), nil
}

func wrapCleanupError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("clean up unregistered physical file: %w", err)
}

type byteCounter struct {
	count int64
}

func (c *byteCounter) Write(source []byte) (int, error) {
	c.count += int64(len(source))
	return len(source), nil
}

var _ Service = (*service)(nil)
