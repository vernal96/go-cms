package filesystem

import (
	"context"
	"errors"
	"io"
	"time"
)

type Code string

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

var (
	ErrNotFound          = errors.New("filesystem object not found")
	ErrConflict          = errors.New("filesystem object already exists")
	ErrDiskNotFound      = errors.New("filesystem disk not found")
	ErrInvalidVisibility = errors.New("invalid filesystem visibility")
	ErrUnsupported       = errors.New("filesystem operation is unsupported")
	ErrUnauthorized      = errors.New("filesystem URL is unauthorized")
)

type Reference struct {
	ID   string
	Path string
}

type Disk interface {
	Code() Code
	Visibility() Visibility
	Ping(context.Context) error
	PutNew(context.Context, string, io.Reader, string) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	URL(context.Context, Reference) (string, error)
	TemporaryURL(context.Context, Reference, time.Time) (string, error)
	Close() error
}

// TemporaryURLVerifier is implemented by disks whose temporary URLs are
// delivered by this application (currently localstorage).
type TemporaryURLVerifier interface {
	VerifyTemporaryURL(Reference, time.Time, string) error
}

type Factory interface {
	Code() Code
	Open(context.Context) (Disk, error)
}

func ValidVisibility(value Visibility) bool {
	return value == VisibilityPublic || value == VisibilityPrivate
}
