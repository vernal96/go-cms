package user

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/access"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type testAccess struct{}

func (testAccess) Check(
	context.Context,
	security.Actor,
	permission.Code,
) error {
	return nil
}
func (testAccess) Codes() []permission.Code { return nil }
func (testAccess) IsPrivileged(
	context.Context,
	security.Actor,
) (bool, error) {
	return true, nil
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

type testMedia struct {
	media.Service
}

func (testMedia) Resolve(
	_ context.Context,
	_ security.Actor,
	id media.ID,
) (media.ResolvedMedia, error) {
	if id != 1 {
		return media.ResolvedMedia{}, media.ErrNotFound
	}
	return media.ResolvedMedia{
		Media: media.Media{ID: id, FileID: 7},
		File: file.File{
			ID:       7,
			MIMEType: "image/png",
		},
	}, nil
}

type testHasher struct {
	verifyCalls int
	hashCalls   int
}

func (h *testHasher) Hash(password string) (string, error) {
	h.hashCalls++
	return "hash:" + password, nil
}

func (h *testHasher) Verify(
	password string,
	encoded string,
) (bool, bool, error) {
	h.verifyCalls++
	valid := encoded == "hash:"+password ||
		encoded == "old:"+password
	return valid, strings.HasPrefix(encoded, "old:"), nil
}

func (*testHasher) DummyHash() string { return "hash:dummy-password" }

type memoryRepository struct {
	nextID   ID
	records  map[ID]Record
	lastHash *string
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		nextID:  1,
		records: make(map[ID]Record),
	}
}

func (r *memoryRepository) Create(
	ctx context.Context,
	actorID *security.UserID,
	record Record,
	validate ValidateAvatarMedia,
) (Record, error) {
	for _, existing := range r.records {
		if existing.Login == record.Login ||
			existing.Email == record.Email {
			return Record{}, ErrConflict
		}
	}
	if record.AvatarMediaID != nil {
		if err := validate(ctx, *record.AvatarMediaID); err != nil {
			return Record{}, err
		}
	}
	record.ID = r.nextID
	r.nextID++
	record.CreatedAt = time.Now().UTC()
	record.UpdatedAt = record.CreatedAt
	record.CreatedBy = cloneUserID(actorID)
	record.UpdatedBy = cloneUserID(actorID)
	r.records[record.ID] = cloneRecord(record)
	return cloneRecord(record), nil
}

func (r *memoryRepository) ByID(
	_ context.Context,
	id ID,
) (Record, error) {
	record, exists := r.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	return cloneRecord(record), nil
}

func (r *memoryRepository) ByIdentifier(
	_ context.Context,
	identifier string,
) (Record, error) {
	for _, record := range r.records {
		if record.Login == identifier ||
			record.Email == identifier {
			return cloneRecord(record), nil
		}
	}
	return Record{}, ErrNotFound
}

func (r *memoryRepository) List(context.Context) ([]Record, error) {
	result := make([]Record, 0, len(r.records))
	for _, record := range r.records {
		result = append(result, cloneRecord(record))
	}
	return result, nil
}

func (r *memoryRepository) Update(
	ctx context.Context,
	actorID *security.UserID,
	_ Record,
	record Record,
	validate ValidateAvatarMedia,
) (Record, error) {
	if record.AvatarMediaID != nil {
		if err := validate(ctx, *record.AvatarMediaID); err != nil {
			return Record{}, err
		}
	}
	record.UpdatedBy = cloneUserID(actorID)
	record.UpdatedAt = time.Now().UTC()
	r.records[record.ID] = cloneRecord(record)
	return cloneRecord(record), nil
}

func (r *memoryRepository) ChangePassword(
	_ context.Context,
	actorID *security.UserID,
	id ID,
	passwordHash string,
) (Record, error) {
	record, exists := r.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	record.PasswordHash = passwordHash
	record.UpdatedBy = cloneUserID(actorID)
	record.UpdatedAt = time.Now().UTC()
	r.records[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) RecordLogin(
	_ context.Context,
	id ID,
	passwordHash *string,
) (Record, error) {
	record, exists := r.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	now := time.Now().UTC()
	record.LastLoginAt = &now
	if passwordHash != nil {
		record.PasswordHash = *passwordHash
		hash := *passwordHash
		r.lastHash = &hash
	}
	r.records[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) Delete(
	_ context.Context,
	actorID *security.UserID,
	id ID,
) (Record, error) {
	record, exists := r.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	now := time.Now().UTC()
	record.DeletedAt = &now
	record.DeletedBy = cloneUserID(actorID)
	r.records[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) Restore(
	_ context.Context,
	actorID *security.UserID,
	id ID,
) (Record, error) {
	record, exists := r.records[id]
	if !exists {
		return Record{}, ErrNotFound
	}
	record.DeletedAt = nil
	record.DeletedBy = nil
	record.UpdatedBy = cloneUserID(actorID)
	r.records[id] = record
	return cloneRecord(record), nil
}

func newService(
	t *testing.T,
) (*ApplicationService, *memoryRepository, *testHasher) {
	t.Helper()
	repository := newMemoryRepository()
	hasher := &testHasher{}
	service, err := NewService(
		repository,
		hasher,
		testMedia{},
		testAccess{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, hasher
}

func TestCreateNormalizesIdentityAndReservesSoftDeletedValues(
	t *testing.T,
) {
	t.Parallel()

	service, _, _ := newService(t)
	avatar := media.ID(1)
	created, err := service.Create(
		context.Background(),
		security.User(99),
		CreateInput{
			Login:         "  Admin.User ",
			Email:         " ADMIN@EXAMPLE.TEST ",
			Password:      "a-valid-password",
			Name:          " Administrator ",
			AvatarMediaID: &avatar,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.Login != "admin.user" ||
		created.Email != "admin@example.test" ||
		created.Name != "Administrator" ||
		created.CreatedBy == nil ||
		*created.CreatedBy != 99 {
		t.Fatalf("created user = %#v", created)
	}
	if _, err := service.Delete(
		context.Background(),
		security.System(),
		created.ID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(
		context.Background(),
		security.System(),
		CreateInput{
			Login:    "admin.user",
			Email:    "another@example.test",
			Password: "another-password",
			Name:     "Replacement",
		},
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("reserved login error = %v", err)
	}
}

func TestAuthenticateUsesGenericErrorsDummyHashAndRehashes(
	t *testing.T,
) {
	t.Parallel()

	service, repository, hasher := newService(t)
	repository.records[1] = Record{
		User: User{
			ID:    1,
			Login: "active",
			Email: "active@example.test",
			Name:  "Active",
		},
		PasswordHash: "old:a-valid-password",
	}
	deletedAt := time.Now().UTC()
	repository.records[2] = Record{
		User: User{
			ID:        2,
			Login:     "deleted",
			Email:     "deleted@example.test",
			Name:      "Deleted",
			DeletedAt: &deletedAt,
		},
		PasswordHash: "hash:a-valid-password",
	}

	unknownErr := authenticateError(
		t,
		service,
		"missing",
		"a-valid-password",
	)
	wrongErr := authenticateError(
		t,
		service,
		"active",
		"wrong-password",
	)
	deletedErr := authenticateError(
		t,
		service,
		"deleted",
		"a-valid-password",
	)
	for _, err := range []error{unknownErr, wrongErr, deletedErr} {
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("credential error = %v", err)
		}
	}
	if hasher.verifyCalls < 3 {
		t.Fatalf("verify calls = %d", hasher.verifyCalls)
	}

	authenticated, err := service.Authenticate(
		context.Background(),
		AuthenticateInput{
			Identifier: " ACTIVE@EXAMPLE.TEST ",
			Password:   "a-valid-password",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.LastLoginAt == nil ||
		repository.lastHash == nil ||
		*repository.lastHash != "hash:a-valid-password" {
		t.Fatalf(
			"authentication result = %#v, rehash = %#v",
			authenticated,
			repository.lastHash,
		)
	}
}

func authenticateError(
	t *testing.T,
	service *ApplicationService,
	identifier string,
	password string,
) error {
	t.Helper()
	_, err := service.Authenticate(
		context.Background(),
		AuthenticateInput{
			Identifier: identifier,
			Password:   password,
		},
	)
	return err
}

var _ access.Service = testAccess{}
var _ Repository = (*memoryRepository)(nil)
