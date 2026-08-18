package core

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/security"
)

const repositoryCacheInvalidationTimeout = 5 * time.Second

type repositoryCacheTarget struct {
	code      cache.Code
	namespace string
}

// repositoryCachePolicy owns cache coherence for all runtime-scoped core
// repository caches in this application. Targets are keyed by their physical
// store and namespace so rebuilding the same site runtime is idempotent. A
// target remains registered for the application lifetime: a retired runtime
// may still serve an in-flight read and must not retain stale data.
type repositoryCachePolicy struct {
	coherenceMu sync.RWMutex
	targetMu    sync.RWMutex
	targets     map[repositoryCacheTarget]cache.Store
}

func newRepositoryCachePolicy() *repositoryCachePolicy {
	return &repositoryCachePolicy{
		targets: make(map[repositoryCacheTarget]cache.Store),
	}
}

func (p *repositoryCachePolicy) register(
	descriptor RepositoryCacheDescriptor,
	store cache.Store,
) {
	if p == nil || store == nil {
		return
	}
	p.targetMu.Lock()
	p.targets[repositoryCacheTarget{
		code:      descriptor.Code,
		namespace: descriptor.Namespace,
	}] = store
	p.targetMu.Unlock()
}

func (p *repositoryCachePolicy) invalidate(
	ctx context.Context,
	tags ...cache.Tag,
) {
	if p == nil {
		return
	}
	p.targetMu.RLock()
	targets := make([]cache.Store, 0, len(p.targets))
	for _, store := range p.targets {
		targets = append(targets, store)
	}
	p.targetMu.RUnlock()
	if ctx == nil {
		ctx = context.Background()
	}
	invalidationCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		repositoryCacheInvalidationTimeout,
	)
	defer cancel()
	for _, store := range targets {
		invalidateTags(invalidationCtx, store, tags...)
	}
}

func withRepositoryCacheRead[T any](
	policy *repositoryCachePolicy,
	read func() (T, error),
) (T, error) {
	if policy == nil {
		return read()
	}
	policy.coherenceMu.RLock()
	defer policy.coherenceMu.RUnlock()
	return read()
}

func withRepositoryCacheWrite(
	policy *repositoryCachePolicy,
	write func() error,
) error {
	if policy == nil {
		return write()
	}
	policy.coherenceMu.Lock()
	defer policy.coherenceMu.Unlock()
	return write()
}

type coherentDatabase struct {
	Database
	sites     site.Repository
	resources resource.Repository
	policy    *repositoryCachePolicy
}

func newCoherentDatabase(
	database Database,
) (*coherentDatabase, error) {
	if err := validateDatabase(database); err != nil {
		return nil, err
	}
	policy := newRepositoryCachePolicy()
	return &coherentDatabase{
		Database: database,
		sites: &invalidatingSiteRepository{
			base: database.Sites(), policy: policy,
		},
		resources: &invalidatingResourceRepository{
			base: database.Resources(), policy: policy,
		},
		policy: policy,
	}, nil
}

func (d *coherentDatabase) Sites() site.Repository {
	return d.sites
}

func (d *coherentDatabase) Resources() resource.Repository {
	return d.resources
}

type invalidatingSiteRepository struct {
	base   site.Repository
	policy *repositoryCachePolicy
}

func (r *invalidatingSiteRepository) List(
	ctx context.Context,
) ([]site.Site, error) {
	return r.base.List(ctx)
}

func (r *invalidatingSiteRepository) FindByID(
	ctx context.Context,
	id site.ID,
) (site.Site, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Site{}, errors.New("site management repository is unavailable")
	}
	return management.FindByID(ctx, id)
}

func (r *invalidatingSiteRepository) FindByDomain(
	ctx context.Context,
	domain string,
) (site.Site, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Site{}, errors.New("site management repository is unavailable")
	}
	return management.FindByDomain(ctx, domain)
}

func (r *invalidatingSiteRepository) ListPage(
	ctx context.Context,
	query site.ListQuery,
) (site.Page, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Page{}, errors.New("site management repository is unavailable")
	}
	return management.ListPage(ctx, query)
}

func (r *invalidatingSiteRepository) Statistics(
	ctx context.Context,
	query site.StatisticsQuery,
) (site.Statistics, error) {
	statistics, ok := r.base.(site.StatisticsRepository)
	if !ok {
		return site.Statistics{}, errors.New("site statistics repository is unavailable")
	}
	return statistics.Statistics(ctx, query)
}

func (r *invalidatingSiteRepository) Create(
	ctx context.Context,
	actorID *security.UserID,
	item site.Site,
) (site.Site, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Site{}, errors.New("site management repository is unavailable")
	}
	var result site.Site
	err := withRepositoryCacheWrite(r.policy, func() error {
		var err error
		result, err = management.Create(ctx, actorID, item)
		if err != nil {
			return err
		}
		r.policy.invalidate(ctx, sitesTag, siteTag(result.ID))
		return nil
	})
	return result, err
}

func (r *invalidatingSiteRepository) Update(
	ctx context.Context,
	actorID *security.UserID,
	item site.Site,
) (site.Site, error) {
	var result site.Site
	err := withRepositoryCacheWrite(r.policy, func() error {
		var err error
		result, err = r.base.Update(ctx, actorID, item)
		if err != nil {
			return err
		}
		r.policy.invalidate(ctx, sitesTag, siteTag(result.ID))
		return nil
	})
	return result, err
}

func (r *invalidatingSiteRepository) Delete(
	ctx context.Context,
	id site.ID,
) error {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return errors.New("site management repository is unavailable")
	}
	return withRepositoryCacheWrite(r.policy, func() error {
		if err := management.Delete(ctx, id); err != nil {
			return err
		}
		r.policy.invalidate(ctx, sitesTag, siteTag(id))
		return nil
	})
}

type invalidatingResourceRepository struct {
	base   resource.Repository
	policy *repositoryCachePolicy
}

func (r *invalidatingResourceRepository) Create(
	ctx context.Context,
	actorID *security.UserID,
	item resource.Resource,
	validate resource.ValidateImageMedia,
) (resource.Resource, error) {
	var result resource.Resource
	err := withRepositoryCacheWrite(r.policy, func() error {
		var err error
		result, err = r.base.Create(ctx, actorID, item, validate)
		if err != nil {
			return err
		}
		r.policy.invalidate(ctx, resourceTags(result)...)
		return nil
	})
	return result, err
}

func (r *invalidatingResourceRepository) ByID(
	ctx context.Context,
	id resource.ID,
) (resource.Resource, error) {
	return r.base.ByID(ctx, id)
}

func (r *invalidatingResourceRepository) ByPath(
	ctx context.Context,
	siteID site.ID,
	path string,
) (resource.Resource, error) {
	return r.base.ByPath(ctx, siteID, path)
}

func (r *invalidatingResourceRepository) ListBySite(
	ctx context.Context,
	siteID site.ID,
) ([]resource.Resource, error) {
	return r.base.ListBySite(ctx, siteID)
}

func (r *invalidatingResourceRepository) ExistsInSite(
	ctx context.Context,
	siteID site.ID,
	id resource.ID,
) (bool, error) {
	management, ok := r.base.(resource.ManagementRepository)
	if !ok {
		return false, errors.New("resource management repository is unavailable")
	}
	return management.ExistsInSite(ctx, siteID, id)
}

func (r *invalidatingResourceRepository) ListChildren(
	ctx context.Context,
	siteID site.ID,
	parentID *resource.ID,
) ([]resource.Child, error) {
	management, ok := r.base.(resource.ManagementRepository)
	if !ok {
		return nil, errors.New("resource management repository is unavailable")
	}
	return management.ListChildren(ctx, siteID, parentID)
}

func (r *invalidatingResourceRepository) Statistics(
	ctx context.Context,
	query resource.StatisticsQuery,
) (resource.Statistics, error) {
	statistics, ok := r.base.(resource.StatisticsRepository)
	if !ok {
		return resource.Statistics{}, errors.New("resource statistics repository is unavailable")
	}
	return statistics.Statistics(ctx, query)
}

func (r *invalidatingResourceRepository) Update(
	ctx context.Context,
	actorID *security.UserID,
	current resource.Resource,
	item resource.Resource,
	validate resource.ValidateImageMedia,
) (resource.Resource, error) {
	var result resource.Resource
	err := withRepositoryCacheWrite(r.policy, func() error {
		var err error
		result, err = r.base.Update(ctx, actorID, current, item, validate)
		if err != nil {
			return err
		}
		r.policy.invalidate(
			ctx,
			siteTag(current.SiteID),
			siteTag(result.SiteID),
			resourceTag(current.ID),
			resourceTag(result.ID),
		)
		return nil
	})
	return result, err
}

func (r *invalidatingResourceRepository) CreateWidget(ctx context.Context, id resource.ID, binding widget.Binding) (widget.Binding, error) {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return widget.Binding{}, errors.New("resource widget repository is unavailable")
	}
	var result widget.Binding
	err := r.mutateWidgets(ctx, id, func() error {
		var err error
		result, err = repository.CreateWidget(ctx, id, binding)
		return err
	})
	return result, err
}

func (r *invalidatingResourceRepository) UpdateWidget(ctx context.Context, id resource.ID, binding widget.Binding) (widget.Binding, error) {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return widget.Binding{}, errors.New("resource widget repository is unavailable")
	}
	var result widget.Binding
	err := r.mutateWidgets(ctx, id, func() error {
		var err error
		result, err = repository.UpdateWidget(ctx, id, binding)
		return err
	})
	return result, err
}

func (r *invalidatingResourceRepository) DeleteWidget(ctx context.Context, id resource.ID, bindingID widget.BindingID) error {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return errors.New("resource widget repository is unavailable")
	}
	return r.mutateWidgets(ctx, id, func() error {
		return repository.DeleteWidget(ctx, id, bindingID)
	})
}

func (r *invalidatingResourceRepository) ReorderWidgets(ctx context.Context, id resource.ID, order []widget.Order) ([]widget.Binding, error) {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return nil, errors.New("resource widget repository is unavailable")
	}
	var result []widget.Binding
	err := r.mutateWidgets(ctx, id, func() error {
		var err error
		result, err = repository.ReorderWidgets(ctx, id, order)
		return err
	})
	return result, err
}

func (r *invalidatingResourceRepository) mutateWidgets(ctx context.Context, id resource.ID, mutate func() error) error {
	return withRepositoryCacheWrite(r.policy, func() error {
		current, err := r.base.ByID(ctx, id)
		if err != nil {
			return err
		}
		if err := mutate(); err != nil {
			return err
		}
		r.policy.invalidate(ctx, resourceTags(current)...)
		return nil
	})
}

func (r *invalidatingResourceRepository) Delete(
	ctx context.Context,
	id resource.ID,
) error {
	return withRepositoryCacheWrite(r.policy, func() error {
		current, err := r.base.ByID(ctx, id)
		if err != nil {
			return err
		}
		if err := r.base.Delete(ctx, id); err != nil {
			return err
		}
		r.policy.invalidate(ctx, resourceTags(current)...)
		return nil
	})
}

func (r *invalidatingResourceRepository) SoftDelete(
	ctx context.Context,
	actorID *security.UserID,
	id resource.ID,
) error {
	lifecycle, ok := r.base.(resource.LifecycleRepository)
	if !ok {
		return errors.New("resource lifecycle repository is unavailable")
	}
	return withRepositoryCacheWrite(r.policy, func() error {
		current, err := r.base.ByID(ctx, id)
		if err != nil {
			return err
		}
		if err := lifecycle.SoftDelete(ctx, actorID, id); err != nil {
			return err
		}
		r.policy.invalidate(ctx, resourceTags(current)...)
		return nil
	})
}

func (r *invalidatingResourceRepository) Restore(
	ctx context.Context,
	actorID *security.UserID,
	id resource.ID,
	withDescendants bool,
) error {
	lifecycle, ok := r.base.(resource.LifecycleRepository)
	if !ok {
		return errors.New("resource lifecycle repository is unavailable")
	}
	return withRepositoryCacheWrite(r.policy, func() error {
		current, err := r.base.ByID(ctx, id)
		if err != nil {
			return err
		}
		if err := lifecycle.Restore(ctx, actorID, id, withDescendants); err != nil {
			return err
		}
		r.policy.invalidate(ctx, resourceTags(current)...)
		return nil
	})
}

var _ Database = (*coherentDatabase)(nil)
var _ site.ManagementRepository = (*invalidatingSiteRepository)(nil)
var _ site.StatisticsRepository = (*invalidatingSiteRepository)(nil)
var _ resource.ManagementRepository = (*invalidatingResourceRepository)(nil)
var _ resource.WidgetRepository = (*invalidatingResourceRepository)(nil)
var _ resource.LifecycleRepository = (*invalidatingResourceRepository)(nil)
var _ resource.StatisticsRepository = (*invalidatingResourceRepository)(nil)
