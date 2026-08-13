package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

const (
	FileReadPermission   permission.Code = "core.file.read"
	FileCreatePermission permission.Code = "core.file.create"
	FileUpdatePermission permission.Code = "core.file.update"
	FileDeletePermission permission.Code = "core.file.delete"
)

type FilesystemDiskDTO struct {
	Code       filesystem.Code       `json:"code"`
	Visibility filesystem.Visibility `json:"visibility"`
}

type FilesystemDisks struct {
	Items       []FilesystemDiskDTO `json:"items"`
	Permissions PermissionSet       `json:"permissions"`
}

type FilesystemItemDTO struct {
	Kind      file.ItemKind   `json:"kind"`
	ID        int64           `json:"id"`
	ParentID  *file.FolderID  `json:"parent_id"`
	Storage   filesystem.Code `json:"storage"`
	Name      string          `json:"name"`
	MIMEType  *string         `json:"mime_type,omitempty"`
	Size      *int64          `json:"size,omitempty"`
	ItemCount *int            `json:"item_count,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type FilesystemBreadcrumbDTO struct {
	ID   file.FolderID `json:"id"`
	Name string        `json:"name"`
}

type FilesystemListing struct {
	Disk        FilesystemDiskDTO         `json:"disk"`
	Folder      *FilesystemItemDTO        `json:"folder"`
	Breadcrumbs []FilesystemBreadcrumbDTO `json:"breadcrumbs"`
	Items       []FilesystemItemDTO       `json:"items"`
	Permissions PermissionSet             `json:"permissions"`
}

func (m *Management) FilesystemDisks(
	ctx context.Context,
	actor security.Actor,
) (FilesystemDisks, error) {
	items, err := m.files.Disks(ctx, actor)
	if err != nil {
		return FilesystemDisks{}, err
	}
	result := make([]FilesystemDiskDTO, len(items))
	for index, item := range items {
		result[index] = FilesystemDiskDTO{Code: item.Code, Visibility: item.Visibility}
	}
	permissions, err := m.filePermissions(ctx, actor)
	if err != nil {
		return FilesystemDisks{}, err
	}
	return FilesystemDisks{Items: result, Permissions: permissions}, nil
}

func (m *Management) BrowseFilesystem(
	ctx context.Context,
	actor security.Actor,
	storage filesystem.Code,
	folderID *file.FolderID,
) (FilesystemListing, error) {
	listing, err := m.files.Browse(ctx, actor, storage, folderID)
	if err != nil {
		return FilesystemListing{}, err
	}
	items := make([]FilesystemItemDTO, 0, len(listing.Folders)+len(listing.Files))
	for _, entry := range listing.Folders {
		count := entry.ItemCount
		items = append(items, folderItemDTO(entry.Folder, &count))
	}
	for _, item := range listing.Files {
		items = append(items, fileItemDTO(item))
	}
	breadcrumbs := make([]FilesystemBreadcrumbDTO, len(listing.Breadcrumbs))
	for index, item := range listing.Breadcrumbs {
		breadcrumbs[index] = FilesystemBreadcrumbDTO{ID: item.ID, Name: item.Name}
	}
	var folder *FilesystemItemDTO
	if listing.Folder != nil {
		value := folderItemDTO(*listing.Folder, nil)
		folder = &value
	}
	permissions, err := m.filePermissions(ctx, actor)
	if err != nil {
		return FilesystemListing{}, err
	}
	return FilesystemListing{
		Disk:   FilesystemDiskDTO{Code: listing.Storage, Visibility: listing.Visibility},
		Folder: folder, Breadcrumbs: breadcrumbs, Items: items, Permissions: permissions,
	}, nil
}

func (m *Management) CreateFilesystemFolder(
	ctx context.Context,
	actor security.Actor,
	input file.CreateFolderInput,
) (FilesystemItemDTO, error) {
	item, err := m.files.CreateAvailableFolder(ctx, actor, input)
	if err != nil {
		return FilesystemItemDTO{}, fileValidationError(err)
	}
	return folderItemDTO(item, intPointer(0)), nil
}

func (m *Management) UploadFilesystemFile(
	ctx context.Context,
	actor security.Actor,
	input file.UploadInput,
) (FilesystemItemDTO, error) {
	item, err := m.files.UploadAvailable(ctx, actor, input)
	if err != nil {
		return FilesystemItemDTO{}, fileValidationError(err)
	}
	return fileItemDTO(item), nil
}

func (m *Management) FilesystemFile(
	ctx context.Context,
	actor security.Actor,
	id file.ID,
) (FilesystemItemDTO, error) {
	item, err := m.files.GetFile(ctx, actor, id)
	if err != nil {
		return FilesystemItemDTO{}, err
	}
	return fileItemDTO(item), nil
}

func (m *Management) RenameFilesystemFile(
	ctx context.Context,
	actor security.Actor,
	input file.RenameFileInput,
) (FilesystemItemDTO, error) {
	item, err := m.files.RenameFile(ctx, actor, input)
	if err != nil {
		return FilesystemItemDTO{}, fileValidationError(err)
	}
	return fileItemDTO(item), nil
}

func (m *Management) RenameFilesystemFolder(
	ctx context.Context,
	actor security.Actor,
	input file.RenameFolderInput,
) (FilesystemItemDTO, error) {
	item, err := m.files.RenameFolder(ctx, actor, input)
	if err != nil {
		return FilesystemItemDTO{}, fileValidationError(err)
	}
	return folderItemDTO(item, nil), nil
}

func (m *Management) MoveFilesystemItems(
	ctx context.Context,
	actor security.Actor,
	input file.MoveItemsInput,
) error {
	_, _, err := m.files.MoveItems(ctx, actor, input)
	if err != nil {
		return fileValidationError(err)
	}
	return nil
}

func (m *Management) DeleteFilesystemItems(
	ctx context.Context,
	actor security.Actor,
	input file.DeleteItemsInput,
) error {
	if err := m.files.DeleteItems(ctx, actor, input); err != nil {
		return fileValidationError(err)
	}
	return nil
}

func (m *Management) OpenFilesystemFile(
	ctx context.Context,
	actor security.Actor,
	id file.ID,
) (file.OpenedFile, error) {
	return m.files.Open(ctx, actor, id)
}

func (m *Management) filePermissions(ctx context.Context, actor security.Actor) (PermissionSet, error) {
	codes := []permission.Code{FileReadPermission, FileCreatePermission, FileUpdatePermission, FileDeletePermission}
	values := make([]bool, len(codes))
	for index, code := range codes {
		err := m.authorizer.Check(ctx, actor, code)
		switch {
		case err == nil:
			values[index] = true
		case errors.Is(err, security.ErrForbidden):
		default:
			return PermissionSet{}, fmt.Errorf("check file permission %q: %w", code, err)
		}
	}
	return PermissionSet{Read: values[0], Create: values[1], Update: values[2], Delete: values[3]}, nil
}

func folderItemDTO(item file.Folder, count *int) FilesystemItemDTO {
	return FilesystemItemDTO{Kind: file.ItemFolder, ID: int64(item.ID), ParentID: item.ParentID,
		Storage: item.Storage, Name: item.Name, ItemCount: count,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func fileItemDTO(item file.File) FilesystemItemDTO {
	mimeType, size := item.MIMEType, item.Size
	return FilesystemItemDTO{Kind: file.ItemFile, ID: int64(item.ID), ParentID: item.FolderID,
		Storage: item.Storage, Name: item.Name, MIMEType: &mimeType, Size: &size,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func intPointer(value int) *int { return &value }

func fileValidationError(err error) error {
	switch {
	case errors.Is(err, security.ErrUnauthenticated),
		errors.Is(err, security.ErrForbidden),
		errors.Is(err, file.ErrNotFound),
		errors.Is(err, file.ErrFolderNotFound),
		errors.Is(err, file.ErrStorageNotFound),
		errors.Is(err, file.ErrConflict),
		errors.Is(err, file.ErrStorageMismatch),
		errors.Is(err, file.ErrInvalidTree),
		errors.Is(err, file.ErrInUse):
		return err
	default:
		return fmt.Errorf("%w: request data is invalid", ErrValidation)
	}
}
