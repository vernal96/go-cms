package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/vernal96/go-cms/kernel/modules/core/access"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

const (
	minPasswordBytes = 12
	maxPasswordBytes = 1024
)

var (
	loginPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,63}$`)

	readPermission   = permission.MustCode("core", "user", permission.Read)
	createPermission = permission.MustCode("core", "user", permission.Create)
	updatePermission = permission.MustCode("core", "user", permission.Update)
	deletePermission = permission.MustCode("core", "user", permission.Delete)
)

type ApplicationService struct {
	repository Repository
	hasher     PasswordHasher
	media      MediaService
	access     access.Service
}

func NewService(
	repository Repository,
	hasher PasswordHasher,
	mediaService MediaService,
	accessService access.Service,
) (*ApplicationService, error) {
	switch {
	case repository == nil:
		return nil, errors.New("user repository is nil")
	case hasher == nil:
		return nil, errors.New("user password hasher is nil")
	case mediaService == nil:
		return nil, errors.New("user media service is nil")
	case accessService == nil:
		return nil, errors.New("user access service is nil")
	}

	return &ApplicationService{
		repository: repository,
		hasher:     hasher,
		media:      mediaService,
		access:     accessService,
	}, nil
}

func (s *ApplicationService) Create(
	ctx context.Context,
	actor security.Actor,
	input CreateInput,
) (User, error) {
	if err := s.access.Check(ctx, actor, createPermission); err != nil {
		return User{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return User{}, err
	}

	item, err := normalize(Record{User: User{
		Login:         input.Login,
		Email:         input.Email,
		Name:          input.Name,
		LastName:      input.LastName,
		MiddleName:    input.MiddleName,
		Phone:         input.Phone,
		AvatarMediaID: input.AvatarMediaID,
	}})
	if err != nil {
		return User{}, err
	}
	item.PasswordHash, err = s.hasher.Hash(input.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash user password: %w", err)
	}

	created, err := s.repository.Create(
		ctx,
		actor.AuditUserID(),
		item,
		s.validateAvatar,
	)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return Clone(created.User), nil
}

func (s *ApplicationService) Get(
	ctx context.Context,
	actor security.Actor,
	id ID,
) (User, error) {
	if err := s.access.Check(ctx, actor, readPermission); err != nil {
		return User{}, err
	}
	record, err := s.byID(ctx, id)
	if err != nil {
		return User{}, err
	}
	return Clone(record.User), nil
}

func (s *ApplicationService) List(
	ctx context.Context,
	actor security.Actor,
) ([]User, error) {
	if err := s.access.Check(ctx, actor, readPermission); err != nil {
		return nil, err
	}
	records, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	result := make([]User, len(records))
	for index, record := range records {
		result[index] = Clone(record.User)
	}
	return result, nil
}

func (s *ApplicationService) Update(
	ctx context.Context,
	actor security.Actor,
	input UpdateInput,
) (User, error) {
	if err := s.access.Check(ctx, actor, updatePermission); err != nil {
		return User{}, err
	}
	current, err := s.byID(ctx, input.ID)
	if err != nil {
		return User{}, err
	}

	next := cloneRecord(current)
	next.Login = input.Login
	next.Email = input.Email
	next.Name = input.Name
	next.LastName = input.LastName
	next.MiddleName = input.MiddleName
	next.Phone = input.Phone
	next.AvatarMediaID = input.AvatarMediaID
	next, err = normalize(next)
	if err != nil {
		return User{}, err
	}

	updated, err := s.repository.Update(
		ctx,
		actor.AuditUserID(),
		current,
		next,
		s.validateAvatar,
	)
	if err != nil {
		return User{}, fmt.Errorf("update user: %w", err)
	}
	return Clone(updated.User), nil
}

func (s *ApplicationService) ChangePassword(
	ctx context.Context,
	actor security.Actor,
	id ID,
	password string,
) (User, error) {
	if err := s.access.Check(ctx, actor, updatePermission); err != nil {
		return User{}, err
	}
	if id <= 0 {
		return User{}, errors.New("invalid user id")
	}
	if err := validatePassword(password); err != nil {
		return User{}, err
	}
	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, fmt.Errorf("hash user password: %w", err)
	}
	updated, err := s.repository.ChangePassword(
		ctx,
		actor.AuditUserID(),
		id,
		passwordHash,
	)
	if err != nil {
		return User{}, fmt.Errorf("change user password: %w", err)
	}
	return Clone(updated.User), nil
}

func (s *ApplicationService) Delete(
	ctx context.Context,
	actor security.Actor,
	id ID,
) (User, error) {
	if err := s.access.Check(ctx, actor, deletePermission); err != nil {
		return User{}, err
	}
	if id <= 0 {
		return User{}, errors.New("invalid user id")
	}
	deleted, err := s.repository.Delete(
		ctx,
		actor.AuditUserID(),
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("delete user: %w", err)
	}
	return Clone(deleted.User), nil
}

func (s *ApplicationService) Restore(
	ctx context.Context,
	actor security.Actor,
	id ID,
) (User, error) {
	if err := s.access.Check(ctx, actor, updatePermission); err != nil {
		return User{}, err
	}
	if id <= 0 {
		return User{}, errors.New("invalid user id")
	}
	restored, err := s.repository.Restore(
		ctx,
		actor.AuditUserID(),
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("restore user: %w", err)
	}
	return Clone(restored.User), nil
}

func (s *ApplicationService) Authenticate(
	ctx context.Context,
	input AuthenticateInput,
) (User, error) {
	if ctx == nil {
		return User{}, errors.New("authenticate context is nil")
	}
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	identifier := strings.ToLower(strings.TrimSpace(input.Identifier))
	if identifier == "" || len(input.Password) > maxPasswordBytes {
		dummyPassword := input.Password
		if len(dummyPassword) > maxPasswordBytes {
			dummyPassword = "invalid-password-too-long"
		}
		_, _, verifyErr := s.hasher.Verify(
			dummyPassword,
			s.hasher.DummyHash(),
		)
		if verifyErr != nil {
			return User{}, fmt.Errorf(
				"verify dummy password: %w",
				verifyErr,
			)
		}
		return User{}, ErrInvalidCredentials
	}

	record, err := s.repository.ByIdentifier(ctx, identifier)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return User{}, fmt.Errorf("load authentication user: %w", err)
		}
		_, _, verifyErr := s.hasher.Verify(
			input.Password,
			s.hasher.DummyHash(),
		)
		if verifyErr != nil {
			return User{}, fmt.Errorf(
				"verify dummy password: %w",
				verifyErr,
			)
		}
		return User{}, ErrInvalidCredentials
	}

	valid, needsRehash, err := s.hasher.Verify(
		input.Password,
		record.PasswordHash,
	)
	if err != nil {
		return User{}, fmt.Errorf("verify user password: %w", err)
	}
	if !valid || record.DeletedAt != nil {
		return User{}, ErrInvalidCredentials
	}

	var passwordHash *string
	if needsRehash {
		hash, err := s.hasher.Hash(input.Password)
		if err != nil {
			return User{}, fmt.Errorf("rehash user password: %w", err)
		}
		passwordHash = &hash
	}

	authenticated, err := s.repository.RecordLogin(
		ctx,
		record.ID,
		passwordHash,
	)
	if err != nil {
		return User{}, fmt.Errorf("record user login: %w", err)
	}
	return Clone(authenticated.User), nil
}

func (s *ApplicationService) byID(
	ctx context.Context,
	id ID,
) (Record, error) {
	if id <= 0 {
		return Record{}, errors.New("invalid user id")
	}
	record, err := s.repository.ByID(ctx, id)
	if err != nil {
		return Record{}, fmt.Errorf("get user: %w", err)
	}
	return cloneRecord(record), nil
}

func (s *ApplicationService) validateAvatar(
	ctx context.Context,
	id media.ID,
) error {
	resolved, err := s.media.Resolve(ctx, security.System(), id)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return ErrInvalidReference
		}
		return err
	}
	return ValidateAvatarMediaFile(
		ctx,
		resolved.File,
		media.Usage{Kind: AvatarMediaUsage},
	)
}

func normalize(record Record) (Record, error) {
	record.Login = strings.ToLower(strings.TrimSpace(record.Login))
	record.Email = strings.ToLower(strings.TrimSpace(record.Email))
	record.Name = strings.TrimSpace(record.Name)
	record.LastName = normalizeOptional(record.LastName)
	record.MiddleName = normalizeOptional(record.MiddleName)
	record.Phone = normalizeOptional(record.Phone)

	if !loginPattern.MatchString(record.Login) {
		return Record{}, errors.New("invalid user login")
	}
	address, err := mail.ParseAddress(record.Email)
	if err != nil || address.Address != record.Email ||
		len(record.Email) > 254 {
		return Record{}, errors.New("invalid user email")
	}
	if record.Name == "" || utf8.RuneCountInString(record.Name) > 200 {
		return Record{}, errors.New("invalid user name")
	}
	return record, nil
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func validatePassword(password string) error {
	size := len(password)
	if size < minPasswordBytes || size > maxPasswordBytes {
		return fmt.Errorf(
			"password must contain between %d and %d bytes",
			minPasswordBytes,
			maxPasswordBytes,
		)
	}
	return nil
}

var _ Service = (*ApplicationService)(nil)
