package group

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type ID int64

var (
	ErrNotFound         = errors.New("group not found")
	ErrConflict         = errors.New("group conflict")
	ErrInvalidReference = errors.New("invalid group reference")

	codePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,63}$`)
)

type Group struct {
	ID        ID
	Code      string
	Name      string
	IsSuper   bool
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy *security.UserID
	UpdatedBy *security.UserID
}

type Membership struct {
	UserID    security.UserID
	GroupID   ID
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy *security.UserID
	UpdatedBy *security.UserID
}

type PermissionGrant struct {
	GroupID    ID
	Permission permission.Code
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CreatedBy  *security.UserID
	UpdatedBy  *security.UserID
}

type CreateInput struct {
	Code    string
	Name    string
	IsSuper bool
}

type UpdateInput struct {
	ID      ID
	Name    string
	IsSuper bool
}

type Repository interface {
	Create(context.Context, *security.UserID, Group) (Group, error)
	ByID(context.Context, ID) (Group, error)
	ByCode(context.Context, string) (Group, error)
	List(context.Context) ([]Group, error)
	Update(context.Context, *security.UserID, Group) (Group, error)
	Delete(context.Context, ID) error
	AddUser(
		context.Context,
		*security.UserID,
		ID,
		security.UserID,
	) (Membership, error)
	RemoveUser(context.Context, ID, security.UserID) error
	Members(context.Context, ID) ([]Membership, error)
	GroupsForUser(
		context.Context,
		security.UserID,
	) ([]Group, error)
	GrantPermission(
		context.Context,
		*security.UserID,
		ID,
		permission.Code,
	) (PermissionGrant, error)
	RevokePermission(context.Context, ID, permission.Code) error
	Permissions(context.Context, ID) ([]PermissionGrant, error)
}

type Service interface {
	Create(context.Context, security.Actor, CreateInput) (Group, error)
	Get(context.Context, security.Actor, ID) (Group, error)
	GetByCode(context.Context, security.Actor, string) (Group, error)
	List(context.Context, security.Actor) ([]Group, error)
	Update(context.Context, security.Actor, UpdateInput) (Group, error)
	Delete(context.Context, security.Actor, ID) error
	AddUser(
		context.Context,
		security.Actor,
		ID,
		security.UserID,
	) (Membership, error)
	RemoveUser(
		context.Context,
		security.Actor,
		ID,
		security.UserID,
	) error
	Members(
		context.Context,
		security.Actor,
		ID,
	) ([]Membership, error)
	GroupsForUser(
		context.Context,
		security.Actor,
		security.UserID,
	) ([]Group, error)
	GrantPermission(
		context.Context,
		security.Actor,
		ID,
		permission.Code,
	) (PermissionGrant, error)
	RevokePermission(
		context.Context,
		security.Actor,
		ID,
		permission.Code,
	) error
	Permissions(
		context.Context,
		security.Actor,
		ID,
	) ([]PermissionGrant, error)
}

func Clone(item Group) Group {
	item.CreatedBy = cloneUserID(item.CreatedBy)
	item.UpdatedBy = cloneUserID(item.UpdatedBy)
	return item
}

func cloneUserID(value *security.UserID) *security.UserID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
