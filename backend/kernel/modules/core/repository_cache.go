package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/security"
)

const (
	sitesListCacheKey = "core:site:list:v1"
	sitesTag          = cache.Tag("core.sites")
)

type cachedDatabase struct {
	Database
	sites     site.Repository
	resources resource.Repository
}

func newCachedDatabase(
	database Database,
	store cache.Store,
	ttl time.Duration,
	policy *repositoryCachePolicy,
) Database {
	return &cachedDatabase{
		Database: database,
		sites: &cachedSiteRepository{
			base:   database.Sites(),
			store:  store,
			ttl:    ttl,
			policy: policy,
		},
		resources: &cachedResourceRepository{
			base:   database.Resources(),
			store:  store,
			ttl:    ttl,
			policy: policy,
		},
	}
}

func (d *cachedDatabase) Sites() site.Repository {
	return d.sites
}

func (d *cachedDatabase) Resources() resource.Repository {
	return d.resources
}

type cachedSiteRepository struct {
	base   site.Repository
	store  cache.Store
	ttl    time.Duration
	policy *repositoryCachePolicy
}

func (r *cachedSiteRepository) List(
	ctx context.Context,
) ([]site.Site, error) {
	return withRepositoryCacheRead(r.policy, func() ([]site.Site, error) {
		if result, ok := cacheRead[[]site.Site](
			ctx,
			r.store,
			sitesListCacheKey,
		); ok {
			return result, nil
		}

		result, err := r.base.List(ctx)
		if err != nil {
			return nil, err
		}
		cacheWrite(
			ctx,
			r.store,
			sitesListCacheKey,
			result,
			r.ttl,
			[]cache.Tag{sitesTag},
		)
		return result, nil
	})
}

func (r *cachedSiteRepository) FindByID(
	ctx context.Context,
	id site.ID,
) (site.Site, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Site{}, errors.New("site management repository is unavailable")
	}
	return management.FindByID(ctx, id)
}

func (r *cachedSiteRepository) FindByDomain(
	ctx context.Context,
	domain string,
) (site.Site, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Site{}, errors.New("site management repository is unavailable")
	}
	return management.FindByDomain(ctx, domain)
}

func (r *cachedSiteRepository) ListPage(
	ctx context.Context,
	query site.ListQuery,
) (site.Page, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Page{}, errors.New("site management repository is unavailable")
	}
	return management.ListPage(ctx, query)
}

func (r *cachedSiteRepository) Statistics(
	ctx context.Context,
	query site.StatisticsQuery,
) (site.Statistics, error) {
	statistics, ok := r.base.(site.StatisticsRepository)
	if !ok {
		return site.Statistics{}, errors.New("site statistics repository is unavailable")
	}
	return statistics.Statistics(ctx, query)
}

func (r *cachedSiteRepository) Create(
	ctx context.Context,
	actorID *security.UserID,
	item site.Site,
) (site.Site, error) {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return site.Site{}, errors.New("site management repository is unavailable")
	}
	result, err := management.Create(ctx, actorID, item)
	if err != nil {
		return site.Site{}, err
	}
	return result, nil
}

func (r *cachedSiteRepository) Update(
	ctx context.Context,
	actorID *security.UserID,
	item site.Site,
) (site.Site, error) {
	result, err := r.base.Update(ctx, actorID, item)
	if err != nil {
		return site.Site{}, err
	}
	return result, nil
}

func (r *cachedSiteRepository) Delete(
	ctx context.Context,
	id site.ID,
) error {
	management, ok := r.base.(site.ManagementRepository)
	if !ok {
		return errors.New("site management repository is unavailable")
	}
	if err := management.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

type cachedResourceRepository struct {
	base   resource.Repository
	store  cache.Store
	ttl    time.Duration
	policy *repositoryCachePolicy
}

func (r *cachedResourceRepository) Create(
	ctx context.Context,
	actorID *security.UserID,
	item resource.Resource,
	validate resource.ValidateImageMedia,
) (resource.Resource, error) {
	result, err := r.base.Create(ctx, actorID, item, validate)
	if err != nil {
		return resource.Resource{}, err
	}
	return result, nil
}

func (r *cachedResourceRepository) ByID(
	ctx context.Context,
	id resource.ID,
) (resource.Resource, error) {
	return withRepositoryCacheRead(r.policy, func() (resource.Resource, error) {
		key := fmt.Sprintf("core:resource:id:v2:%d", id)
		if result, ok := cacheRead[resource.Resource](
			ctx,
			r.store,
			key,
		); ok {
			return result, nil
		}
		result, err := r.base.ByID(ctx, id)
		if err != nil {
			return resource.Resource{}, err
		}
		cacheWrite(
			ctx,
			r.store,
			key,
			result,
			r.ttl,
			resourceTags(result),
		)
		return result, nil
	})
}

func (r *cachedResourceRepository) ByPath(
	ctx context.Context,
	siteID site.ID,
	pathValue string,
) (resource.Resource, error) {
	return withRepositoryCacheRead(r.policy, func() (resource.Resource, error) {
		sum := sha256.Sum256([]byte(pathValue))
		key := fmt.Sprintf(
			"core:resource:path:v2:%d:%s",
			siteID,
			hex.EncodeToString(sum[:]),
		)
		if result, ok := cacheRead[resource.Resource](
			ctx,
			r.store,
			key,
		); ok {
			return result, nil
		}
		result, err := r.base.ByPath(ctx, siteID, pathValue)
		if err != nil {
			return resource.Resource{}, err
		}
		cacheWrite(
			ctx,
			r.store,
			key,
			result,
			r.ttl,
			resourceTags(result),
		)
		return result, nil
	})
}

func (r *cachedResourceRepository) ListBySite(
	ctx context.Context,
	siteID site.ID,
) ([]resource.Resource, error) {
	return withRepositoryCacheRead(r.policy, func() ([]resource.Resource, error) {
		key := fmt.Sprintf("core:resource:list:v2:%d", siteID)
		if result, ok := cacheRead[[]resource.Resource](
			ctx,
			r.store,
			key,
		); ok {
			return result, nil
		}
		result, err := r.base.ListBySite(ctx, siteID)
		if err != nil {
			return nil, err
		}
		cacheWrite(
			ctx,
			r.store,
			key,
			result,
			r.ttl,
			[]cache.Tag{siteTag(siteID)},
		)
		return result, nil
	})
}

func (r *cachedResourceRepository) ExistsInSite(
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

func (r *cachedResourceRepository) ListChildren(
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

func (r *cachedResourceRepository) Statistics(
	ctx context.Context,
	query resource.StatisticsQuery,
) (resource.Statistics, error) {
	statistics, ok := r.base.(resource.StatisticsRepository)
	if !ok {
		return resource.Statistics{}, errors.New("resource statistics repository is unavailable")
	}
	return statistics.Statistics(ctx, query)
}

func (r *cachedResourceRepository) Update(
	ctx context.Context,
	actorID *security.UserID,
	current resource.Resource,
	item resource.Resource,
	validate resource.ValidateImageMedia,
) (resource.Resource, error) {
	result, err := r.base.Update(
		ctx,
		actorID,
		current,
		item,
		validate,
	)
	if err != nil {
		return resource.Resource{}, err
	}
	return result, nil
}

func (r *cachedResourceRepository) CreateWidget(ctx context.Context, id resource.ID, binding widget.Binding) (widget.Binding, error) {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return widget.Binding{}, errors.New("resource widget repository is unavailable")
	}
	return repository.CreateWidget(ctx, id, binding)
}

func (r *cachedResourceRepository) UpdateWidget(ctx context.Context, id resource.ID, binding widget.Binding) (widget.Binding, error) {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return widget.Binding{}, errors.New("resource widget repository is unavailable")
	}
	return repository.UpdateWidget(ctx, id, binding)
}

func (r *cachedResourceRepository) DeleteWidget(ctx context.Context, id resource.ID, bindingID widget.BindingID) error {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return errors.New("resource widget repository is unavailable")
	}
	return repository.DeleteWidget(ctx, id, bindingID)
}

func (r *cachedResourceRepository) ReorderWidgets(ctx context.Context, id resource.ID, order []widget.Order) ([]widget.Binding, error) {
	repository, ok := r.base.(resource.WidgetRepository)
	if !ok {
		return nil, errors.New("resource widget repository is unavailable")
	}
	return repository.ReorderWidgets(ctx, id, order)
}

func (r *cachedResourceRepository) Delete(
	ctx context.Context,
	id resource.ID,
) error {
	if err := r.base.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (r *cachedResourceRepository) SoftDelete(
	ctx context.Context,
	actorID *security.UserID,
	id resource.ID,
) error {
	lifecycle, ok := r.base.(resource.LifecycleRepository)
	if !ok {
		return errors.New("resource lifecycle repository is unavailable")
	}
	if err := lifecycle.SoftDelete(ctx, actorID, id); err != nil {
		return err
	}
	return nil
}

func (r *cachedResourceRepository) Restore(
	ctx context.Context,
	actorID *security.UserID,
	id resource.ID,
	withDescendants bool,
) error {
	lifecycle, ok := r.base.(resource.LifecycleRepository)
	if !ok {
		return errors.New("resource lifecycle repository is unavailable")
	}
	if err := lifecycle.Restore(ctx, actorID, id, withDescendants); err != nil {
		return err
	}
	return nil
}

func cacheRead[T any](
	ctx context.Context,
	store cache.Store,
	key string,
) (T, bool) {
	var result T
	raw, err := store.Get(ctx, key)
	if err != nil {
		return result, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		_ = store.Delete(ctx, key)
		return result, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		_ = store.Delete(ctx, key)
		return result, false
	}
	return result, true
}

func cacheWrite(
	ctx context.Context,
	store cache.Store,
	key string,
	value any,
	ttl time.Duration,
	tags []cache.Tag,
) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = store.Set(ctx, key, raw, cache.SetOptions{
		TTL:  ttl,
		Tags: tags,
	})
}

func invalidateTags(
	ctx context.Context,
	store cache.Store,
	tags ...cache.Tag,
) {
	seen := make(map[cache.Tag]struct{}, len(tags))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		_ = store.InvalidateTag(ctx, tag)
	}
}

func siteTag(id site.ID) cache.Tag {
	return cache.Tag(fmt.Sprintf("core.site:%d", id))
}

func resourceTag(id resource.ID) cache.Tag {
	return cache.Tag(fmt.Sprintf("core.resource:%d", id))
}

func resourceTags(item resource.Resource) []cache.Tag {
	return []cache.Tag{
		siteTag(item.SiteID),
		resourceTag(item.ID),
	}
}

var _ Database = (*cachedDatabase)(nil)
var _ site.Repository = (*cachedSiteRepository)(nil)
var _ site.ManagementRepository = (*cachedSiteRepository)(nil)
var _ site.StatisticsRepository = (*cachedSiteRepository)(nil)
var _ resource.Repository = (*cachedResourceRepository)(nil)
var _ resource.WidgetRepository = (*cachedResourceRepository)(nil)
var _ resource.ManagementRepository = (*cachedResourceRepository)(nil)
var _ resource.StatisticsRepository = (*cachedResourceRepository)(nil)
