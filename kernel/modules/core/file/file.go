package file

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/vernal96/go-cms/kernel/filesystem"
)

type ID int64
type FolderID int64

var (
	ErrNotFound         = errors.New("file not found")
	ErrFolderNotFound   = errors.New("file folder not found")
	ErrConflict         = errors.New("file namespace conflict")
	ErrInvalidTree      = errors.New("invalid file folder tree")
	ErrInvalidReference = errors.New("invalid file reference")
	ErrStorageNotFound  = errors.New("file storage not found")
	ErrStorageMismatch  = errors.New("file storage mismatch")
	ErrUnauthorized     = errors.New("file delivery is unauthorized")
)

type Folder struct {
	ID        FolderID
	ParentID  *FolderID
	Storage   filesystem.Code
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type File struct {
	ID             ID
	FolderID       *FolderID
	Storage        filesystem.Code
	Name           string
	MIMEType       string
	Size           int64
	ChecksumSHA256 string
	Path           string
	ParentID       *ID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateFolderInput struct {
	ParentID *FolderID
	Storage  filesystem.Code
	Name     string
}

type UploadInput struct {
	FolderID *FolderID
	Storage  filesystem.Code
	Name     string
	ParentID *ID
	Content  io.Reader
}

type MoveFileInput struct {
	ID       ID
	FolderID *FolderID
}

type MoveFolderInput struct {
	ID       FolderID
	ParentID *FolderID
}

type Listing struct {
	Folder  *Folder
	Folders []Folder
	Files   []File
}

type OpenedFile struct {
	File File
	Body io.ReadCloser
}

type DeliveryAuthorization struct {
	ExpiresAt time.Time
	Signature string
}

type DeletePhysical func(context.Context, []File) error

type Repository interface {
	NameAvailable(
		context.Context,
		filesystem.Code,
		*FolderID,
		string,
	) error
	CreateFolder(context.Context, Folder) (Folder, error)
	FolderByID(context.Context, FolderID) (Folder, error)
	ListFolders(
		context.Context,
		filesystem.Code,
		*FolderID,
	) ([]Folder, error)
	CreateFile(context.Context, File) (File, error)
	FileByID(context.Context, ID) (File, error)
	ListFiles(
		context.Context,
		filesystem.Code,
		*FolderID,
	) ([]File, error)
	MoveFile(context.Context, ID, *FolderID) (File, error)
	MoveFolder(context.Context, FolderID, *FolderID) (Folder, error)
	DeleteFile(context.Context, ID, DeletePhysical) error
	DeleteFolder(context.Context, FolderID, DeletePhysical) error
}

type Service interface {
	CreateFolder(context.Context, CreateFolderInput) (Folder, error)
	GetFolder(context.Context, FolderID) (Folder, error)
	ListFolder(
		context.Context,
		filesystem.Code,
		*FolderID,
	) (Listing, error)
	Upload(context.Context, UploadInput) (File, error)
	GetFile(context.Context, ID) (File, error)
	Open(context.Context, ID) (OpenedFile, error)
	OpenDelivery(
		context.Context,
		ID,
		DeliveryAuthorization,
	) (OpenedFile, error)
	MoveFile(context.Context, MoveFileInput) (File, error)
	MoveFolder(context.Context, MoveFolderInput) (Folder, error)
	DeleteFile(context.Context, ID) error
	DeleteFolder(context.Context, FolderID) error
	URL(context.Context, ID) (string, error)
	TemporaryURL(context.Context, ID, time.Time) (string, error)
}

func CloneFolder(item Folder) Folder {
	item.ParentID = cloneFolderID(item.ParentID)
	return item
}

func Clone(item File) File {
	item.FolderID = cloneFolderID(item.FolderID)
	item.ParentID = cloneID(item.ParentID)
	return item
}

func cloneID(value *ID) *ID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneFolderID(value *FolderID) *FolderID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
