package cache

import (
	"context"
	"errors"
	"time"

	"github.com/vernal96/go-cms/kernel/filesystem"
)

type Code string
type Alias string
type Tag string

var (
	ErrMiss          = errors.New("cache entry not found")
	ErrStoreNotFound = errors.New("cache store not found")
	ErrInvalidTTL    = errors.New("cache TTL is invalid")
)

type SetOptions struct {
	TTL  time.Duration
	Tags []Tag
}

type Store interface {
	Code() Code
	Ping(context.Context) error
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, SetOptions) error
	Exists(context.Context, string) (bool, error)
	Delete(context.Context, string) error
	InvalidateTag(context.Context, Tag) error
	Close() error
}

type Resolver interface {
	Store(Code) (Store, bool)
}

type FilesystemResolver interface {
	Disk(filesystem.Code) (filesystem.Disk, bool)
}

type Dependencies struct {
	Filesystems FilesystemResolver
}

type Factory interface {
	Code() Code
	Open(context.Context, Dependencies) (Store, error)
}

type Binding struct {
	Alias     Alias
	Code      Code
	Namespace string
}

type ModuleManager interface {
	Store(Alias) (Store, bool)
	Binding(Alias) (Binding, bool)
}
