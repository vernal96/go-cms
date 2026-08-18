package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core/access"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/permission"
)

// Services is the application-scoped domain runtime owned by cms.core.
// Site-specific module runtimes may reference these concurrency-safe services,
// while their registries, module instances, and bindings remain site-scoped.
type Services struct {
	Sites         *site.Catalog
	Resources     *resource.Service
	Files         file.ManagementService
	Media         media.Service
	Users         user.Service
	Groups        group.Service
	Authorization access.Service

	database    Database
	cachePolicy *repositoryCachePolicy
}

// NewServices assembles the site-independent part of the core domain. Site
// and resource services are completed after profile blueprints are available.
func NewServices(
	database Database,
	permissions *permission.Catalog,
	filesystems filesystem.Catalog,
	passwordHashers user.PasswordHasherFactory,
) (*Services, error) {
	coherent, err := newCoherentDatabase(database)
	if err != nil {
		return nil, err
	}
	database = coherent
	if permissions == nil {
		return nil, errors.New("core permission catalog is nil")
	}
	if filesystems == nil {
		return nil, errors.New("core filesystem catalog is nil")
	}
	if passwordHashers == nil {
		return nil, errors.New("core password hasher factory is nil")
	}

	authorization, err := access.NewService(
		database.Access(),
		permissions,
	)
	if err != nil {
		return nil, err
	}
	files, err := file.NewService(
		database.Files(),
		filesystems,
		authorization,
	)
	if err != nil {
		return nil, err
	}
	mediaService, err := media.NewService(
		database.Media(),
		files,
		media.FilePolicies{
			resource.ImageMediaUsage: resource.ValidateImageMediaFile,
			user.AvatarMediaUsage:    user.ValidateAvatarMediaFile,
		},
		authorization,
	)
	if err != nil {
		return nil, err
	}
	groups, err := group.NewService(database.Groups(), authorization)
	if err != nil {
		return nil, err
	}
	passwordHasher, err := passwordHashers.Open()
	if err != nil {
		return nil, fmt.Errorf("open core password hasher: %w", err)
	}
	if passwordHasher == nil {
		return nil, errors.New("core password hasher factory returned nil")
	}
	users, err := user.NewService(
		database.Users(),
		passwordHasher,
		mediaService,
		groups,
		authorization,
	)
	if err != nil {
		return nil, err
	}

	return &Services{
		Files:         files,
		Media:         mediaService,
		Users:         users,
		Groups:        groups,
		Authorization: authorization,
		database:      database,
		cachePolicy:   coherent.policy,
	}, nil
}

// Database returns the cache-coherent core persistence boundary used by all
// application services and site runtimes.
func (s *Services) Database() Database {
	if s == nil {
		return nil
	}
	return s.database
}

// BuildContent completes the core runtime once profile definitions can build
// final site-scoped runtimes. It loads all sites before publishing the catalog.
func (s *Services) BuildContent(
	ctx context.Context,
	profiles site.ProfileResolver,
) error {
	if s == nil {
		return errors.New("core services are nil")
	}
	if s.Sites != nil || s.Resources != nil {
		return errors.New("core content services are already built")
	}

	catalog, err := site.NewCatalog(
		s.database.Sites(),
		profiles,
		s.Authorization,
		s.Files,
	)
	if err != nil {
		return err
	}
	if err := catalog.Reload(ctx); err != nil {
		return fmt.Errorf("compile site runtimes: %w", err)
	}
	resources, err := resource.NewService(
		s.database.Resources(),
		catalog,
		s.Media,
		s.Authorization,
		s.Files,
	)
	if err != nil {
		return err
	}

	s.Sites = catalog
	s.Resources = resources
	return nil
}

func validateDatabase(database Database) error {
	if database == nil {
		return errors.New("core database is nil")
	}
	repositories := []struct {
		name  string
		value any
	}{
		{name: "site", value: database.Sites()},
		{name: "resource", value: database.Resources()},
		{name: "file", value: database.Files()},
		{name: "media", value: database.Media()},
		{name: "user", value: database.Users()},
		{name: "group", value: database.Groups()},
		{name: "access", value: database.Access()},
	}
	for _, repository := range repositories {
		if repository.value == nil {
			return fmt.Errorf("core %s repository is nil", repository.name)
		}
	}
	return nil
}
