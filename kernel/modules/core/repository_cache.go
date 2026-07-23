package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
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
) Database {
	return &cachedDatabase{
		Database: database,
		sites: &cachedSiteRepository{
			base:  database.Sites(),
			store: store,
			ttl:   ttl,
		},
		resources: &cachedResourceRepository{
			base:  database.Resources(),
			store: store,
			ttl:   ttl,
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
	base  site.Repository
	store cache.Store
	ttl   time.Duration
}

func (r *cachedSiteRepository) List(
	ctx context.Context,
) ([]site.Site, error) {
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
	invalidateTags(
		ctx,
		r.store,
		sitesTag,
		siteTag(result.ID),
	)
	return result, nil
}

type cachedResourceRepository struct {
	base  resource.Repository
	store cache.Store
	ttl   time.Duration
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
	invalidateTags(
		ctx,
		r.store,
		siteTag(result.SiteID),
		resourceTag(result.ID),
	)
	return result, nil
}

func (r *cachedResourceRepository) ByID(
	ctx context.Context,
	id resource.ID,
) (resource.Resource, error) {
	key := fmt.Sprintf("core:resource:id:v1:%d", id)
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
}

func (r *cachedResourceRepository) ByPath(
	ctx context.Context,
	siteID site.ID,
	pathValue string,
) (resource.Resource, error) {
	sum := sha256.Sum256([]byte(pathValue))
	key := fmt.Sprintf(
		"core:resource:path:v1:%d:%s",
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
}

func (r *cachedResourceRepository) ListBySite(
	ctx context.Context,
	siteID site.ID,
) ([]resource.Resource, error) {
	key := fmt.Sprintf("core:resource:list:v1:%d", siteID)
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
	invalidateTags(
		ctx,
		r.store,
		siteTag(current.SiteID),
		siteTag(result.SiteID),
		resourceTag(current.ID),
		resourceTag(result.ID),
	)
	return result, nil
}

func (r *cachedResourceRepository) Delete(
	ctx context.Context,
	id resource.ID,
) error {
	current, err := r.base.ByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.base.Delete(ctx, id); err != nil {
		return err
	}
	invalidateTags(
		ctx,
		r.store,
		siteTag(current.SiteID),
		resourceTag(current.ID),
	)
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
var _ resource.Repository = (*cachedResourceRepository)(nil)
