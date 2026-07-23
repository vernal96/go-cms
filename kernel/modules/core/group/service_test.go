package group

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/access"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type testAccess struct {
	privileged bool
	codes      []permission.Code
}

func (a testAccess) Check(
	context.Context,
	security.Actor,
	permission.Code,
) error {
	return nil
}
func (a testAccess) Codes() []permission.Code {
	return append([]permission.Code(nil), a.codes...)
}
func (a testAccess) IsPrivileged(
	_ context.Context,
	actor security.Actor,
) (bool, error) {
	return actor.IsSystem() || a.privileged, nil
}
func (testAccess) IsGuestSubject(
	context.Context,
	security.Actor,
) (bool, error) {
	return false, nil
}
func (testAccess) GuestPermissions(
	context.Context,
	security.Actor,
) ([]access.Grant, error) {
	return nil, nil
}
func (testAccess) GrantGuest(
	context.Context,
	security.Actor,
	permission.Code,
) (access.Grant, error) {
	return access.Grant{}, nil
}
func (testAccess) RevokeGuest(
	context.Context,
	security.Actor,
	permission.Code,
) error {
	return nil
}

type memoryRepository struct {
	nextID      ID
	groups      map[ID]Group
	memberships map[ID]map[security.UserID]Membership
	permissions map[ID]map[permission.Code]PermissionGrant
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		nextID:      1,
		groups:      make(map[ID]Group),
		memberships: make(map[ID]map[security.UserID]Membership),
		permissions: make(map[ID]map[permission.Code]PermissionGrant),
	}
}

func (r *memoryRepository) Create(
	_ context.Context,
	actorID *security.UserID,
	item Group,
) (Group, error) {
	for _, existing := range r.groups {
		if existing.Code == item.Code {
			return Group{}, ErrConflict
		}
	}
	item.ID = r.nextID
	r.nextID++
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	item.CreatedBy = cloneUserID(actorID)
	item.UpdatedBy = cloneUserID(actorID)
	r.groups[item.ID] = Clone(item)
	return Clone(item), nil
}

func (r *memoryRepository) ByID(
	_ context.Context,
	id ID,
) (Group, error) {
	item, exists := r.groups[id]
	if !exists {
		return Group{}, ErrNotFound
	}
	return Clone(item), nil
}

func (r *memoryRepository) ByCode(
	_ context.Context,
	code string,
) (Group, error) {
	for _, item := range r.groups {
		if item.Code == code {
			return Clone(item), nil
		}
	}
	return Group{}, ErrNotFound
}

func (r *memoryRepository) List(context.Context) ([]Group, error) {
	result := make([]Group, 0, len(r.groups))
	for _, item := range r.groups {
		result = append(result, Clone(item))
	}
	return result, nil
}

func (r *memoryRepository) Update(
	_ context.Context,
	actorID *security.UserID,
	item Group,
) (Group, error) {
	if _, exists := r.groups[item.ID]; !exists {
		return Group{}, ErrNotFound
	}
	item.UpdatedAt = time.Now().UTC()
	item.UpdatedBy = cloneUserID(actorID)
	r.groups[item.ID] = Clone(item)
	return Clone(item), nil
}

func (r *memoryRepository) Delete(
	_ context.Context,
	id ID,
) error {
	if _, exists := r.groups[id]; !exists {
		return ErrNotFound
	}
	delete(r.groups, id)
	delete(r.memberships, id)
	delete(r.permissions, id)
	return nil
}

func (r *memoryRepository) AddUser(
	_ context.Context,
	actorID *security.UserID,
	groupID ID,
	userID security.UserID,
) (Membership, error) {
	if _, exists := r.groups[groupID]; !exists {
		return Membership{}, ErrNotFound
	}
	if r.memberships[groupID] == nil {
		r.memberships[groupID] = make(map[security.UserID]Membership)
	}
	item := Membership{
		UserID:    userID,
		GroupID:   groupID,
		CreatedAt: time.Now().UTC(),
		CreatedBy: cloneUserID(actorID),
		UpdatedBy: cloneUserID(actorID),
	}
	item.UpdatedAt = item.CreatedAt
	r.memberships[groupID][userID] = item
	return item, nil
}

func (r *memoryRepository) RemoveUser(
	_ context.Context,
	groupID ID,
	userID security.UserID,
) error {
	delete(r.memberships[groupID], userID)
	return nil
}

func (r *memoryRepository) Members(
	_ context.Context,
	groupID ID,
) ([]Membership, error) {
	result := make([]Membership, 0, len(r.memberships[groupID]))
	for _, item := range r.memberships[groupID] {
		result = append(result, item)
	}
	return result, nil
}

func (r *memoryRepository) GroupsForUser(
	_ context.Context,
	userID security.UserID,
) ([]Group, error) {
	var result []Group
	for groupID, members := range r.memberships {
		if _, exists := members[userID]; exists {
			result = append(result, Clone(r.groups[groupID]))
		}
	}
	return result, nil
}

func (r *memoryRepository) GrantPermission(
	_ context.Context,
	actorID *security.UserID,
	groupID ID,
	code permission.Code,
) (PermissionGrant, error) {
	if r.permissions[groupID] == nil {
		r.permissions[groupID] = make(
			map[permission.Code]PermissionGrant,
		)
	}
	item := PermissionGrant{
		GroupID:    groupID,
		Permission: code,
		CreatedAt:  time.Now().UTC(),
		CreatedBy:  cloneUserID(actorID),
		UpdatedBy:  cloneUserID(actorID),
	}
	item.UpdatedAt = item.CreatedAt
	r.permissions[groupID][code] = item
	return item, nil
}

func (r *memoryRepository) RevokePermission(
	_ context.Context,
	groupID ID,
	code permission.Code,
) error {
	delete(r.permissions[groupID], code)
	return nil
}

func (r *memoryRepository) Permissions(
	_ context.Context,
	groupID ID,
) ([]PermissionGrant, error) {
	result := make(
		[]PermissionGrant,
		0,
		len(r.permissions[groupID]),
	)
	for _, item := range r.permissions[groupID] {
		result = append(result, item)
	}
	return result, nil
}

func TestSuperGroupChangesAndMembershipRequirePrivilege(t *testing.T) {
	t.Parallel()

	repository := newMemoryRepository()
	service, err := NewService(repository, testAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(
		context.Background(),
		security.User(1),
		CreateInput{
			Code:    "admin",
			Name:    "Administrator",
			IsSuper: true,
		},
	); !errors.Is(err, access.ErrNotPrivileged) {
		t.Fatalf("non-super create error = %v", err)
	}
	admin, err := service.Create(
		context.Background(),
		security.System(),
		CreateInput{
			Code:    "ADMIN",
			Name:    "Administrator",
			IsSuper: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if admin.Code != "admin" {
		t.Fatalf("normalized code = %q", admin.Code)
	}
	if _, err := service.AddUser(
		context.Background(),
		security.User(1),
		admin.ID,
		7,
	); !errors.Is(err, access.ErrNotPrivileged) {
		t.Fatalf("non-super membership error = %v", err)
	}
	membership, err := service.AddUser(
		context.Background(),
		security.System(),
		admin.ID,
		7,
	)
	if err != nil || membership.UserID != 7 {
		t.Fatalf("system membership = %#v, %v", membership, err)
	}
}

func TestPermissionGrantsRequirePrivilegeAndKnownCatalogCode(
	t *testing.T,
) {
	t.Parallel()

	code := permission.MustCode("core", "site", permission.Read)
	repository := newMemoryRepository()
	service, err := NewService(repository, testAccess{
		codes: []permission.Code{code},
	})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := service.Create(
		context.Background(),
		security.System(),
		CreateInput{Code: "manager", Name: "Manager"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GrantPermission(
		context.Background(),
		security.User(1),
		manager.ID,
		code,
	); !errors.Is(err, access.ErrNotPrivileged) {
		t.Fatalf("non-super grant error = %v", err)
	}
	if _, err := service.GrantPermission(
		context.Background(),
		security.System(),
		manager.ID,
		"core.site.publish",
	); !errors.Is(err, permission.ErrUnknown) {
		t.Fatalf("unknown grant error = %v", err)
	}
	grant, err := service.GrantPermission(
		context.Background(),
		security.System(),
		manager.ID,
		code,
	)
	if err != nil || grant.Permission != code {
		t.Fatalf("grant = %#v, %v", grant, err)
	}
}

var _ access.Service = testAccess{}
var _ Repository = (*memoryRepository)(nil)
