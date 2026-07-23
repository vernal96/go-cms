package user

import (
	"context"
	"errors"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/security"
)

type ID = security.UserID

const AvatarMediaUsage media.UsageKind = "user.avatar"

var (
	ErrNotFound           = errors.New("user not found")
	ErrConflict           = errors.New("user conflict")
	ErrInvalidReference   = errors.New("invalid user reference")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	ID            ID
	Login         string
	Email         string
	Name          string
	LastName      *string
	MiddleName    *string
	Phone         *string
	AvatarMediaID *media.ID
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
	CreatedBy     *security.UserID
	UpdatedBy     *security.UserID
	DeletedBy     *security.UserID
}

type Record struct {
	User
	PasswordHash string
}

type CreateInput struct {
	Login         string
	Email         string
	Password      string
	Name          string
	LastName      *string
	MiddleName    *string
	Phone         *string
	AvatarMediaID *media.ID
}

type UpdateInput struct {
	ID            ID
	Login         string
	Email         string
	Name          string
	LastName      *string
	MiddleName    *string
	Phone         *string
	AvatarMediaID *media.ID
}

type AuthenticateInput struct {
	Identifier string
	Password   string
}

type ValidateAvatarMedia func(context.Context, media.ID) error

type Repository interface {
	Create(
		context.Context,
		*security.UserID,
		Record,
		ValidateAvatarMedia,
	) (Record, error)
	ByID(context.Context, ID) (Record, error)
	ByIdentifier(context.Context, string) (Record, error)
	List(context.Context) ([]Record, error)
	Update(
		context.Context,
		*security.UserID,
		Record,
		Record,
		ValidateAvatarMedia,
	) (Record, error)
	ChangePassword(
		context.Context,
		*security.UserID,
		ID,
		string,
	) (Record, error)
	RecordLogin(context.Context, ID, *string) (Record, error)
	Delete(context.Context, *security.UserID, ID) (Record, error)
	Restore(context.Context, *security.UserID, ID) (Record, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) (valid bool, needsRehash bool, err error)
	DummyHash() string
}

type MediaService interface {
	Resolve(
		context.Context,
		security.Actor,
		media.ID,
	) (media.ResolvedMedia, error)
}

type Service interface {
	Create(
		context.Context,
		security.Actor,
		CreateInput,
	) (User, error)
	Get(context.Context, security.Actor, ID) (User, error)
	List(context.Context, security.Actor) ([]User, error)
	Update(
		context.Context,
		security.Actor,
		UpdateInput,
	) (User, error)
	ChangePassword(
		context.Context,
		security.Actor,
		ID,
		string,
	) (User, error)
	Delete(context.Context, security.Actor, ID) (User, error)
	Restore(context.Context, security.Actor, ID) (User, error)
	Authenticate(context.Context, AuthenticateInput) (User, error)
}

func ValidateAvatarMediaFile(
	_ context.Context,
	item file.File,
	_ media.Usage,
) error {
	if len(item.MIMEType) < len("image/") ||
		item.MIMEType[:len("image/")] != "image/" {
		return ErrInvalidReference
	}
	return nil
}

func Clone(item User) User {
	item.LastName = cloneString(item.LastName)
	item.MiddleName = cloneString(item.MiddleName)
	item.Phone = cloneString(item.Phone)
	item.AvatarMediaID = cloneMediaID(item.AvatarMediaID)
	item.LastLoginAt = cloneTime(item.LastLoginAt)
	item.DeletedAt = cloneTime(item.DeletedAt)
	item.CreatedBy = cloneUserID(item.CreatedBy)
	item.UpdatedBy = cloneUserID(item.UpdatedBy)
	item.DeletedBy = cloneUserID(item.DeletedBy)
	return item
}

func cloneRecord(record Record) Record {
	record.User = Clone(record.User)
	return record
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneMediaID(value *media.ID) *media.ID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneUserID(value *security.UserID) *security.UserID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
