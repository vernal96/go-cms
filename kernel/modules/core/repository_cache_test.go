package core

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

func TestCachedSiteRepositoryUsesCacheAndInvalidatesUpdate(t *testing.T) {
	store := newMemoryCacheStore()
	base := &siteRepositoryStub{
		items: []site.Site{{
			ID:          1,
			ProfileCode: "dev",
			Domain:      "example.test",
			Locale:      "en",
			Settings: map[string]any{
				"limit": json.Number("12"),
			},
		}},
	}
	repository := &cachedSiteRepository{
		base:  base,
		store: store,
		ttl:   5 * time.Minute,
	}

	first, err := repository.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first[0].Settings["limit"] = json.Number("99")
	second, err := repository.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if base.listCalls != 1 {
		t.Fatalf("site list calls = %d", base.listCalls)
	}
	if value, ok := second[0].Settings["limit"].(json.Number); !ok ||
		value.String() != "12" {
		t.Fatalf("cached settings = %#v", second[0].Settings)
	}
	if options := store.options[sitesListCacheKey]; options.TTL != 5*time.Minute ||
		!reflect.DeepEqual(options.Tags, []cache.Tag{sitesTag}) {
		t.Fatalf("site cache options = %#v", options)
	}

	updated := second[0]
	updated.Domain = "new.example.test"
	if _, err := repository.Update(
		context.Background(),
		nil,
		updated,
	); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{sitesTag, siteTag(1)},
	) {
		t.Fatalf("invalidated tags = %v", store.invalidated)
	}
}

func TestCachedRepositoriesFailOpen(t *testing.T) {
	store := newMemoryCacheStore()
	store.getErr = errors.New("redis unavailable")
	store.setErr = errors.New("redis unavailable")
	store.invalidateErr = errors.New("redis unavailable")
	base := &siteRepositoryStub{
		items: []site.Site{{ID: 1, Settings: map[string]any{}}},
	}
	repository := &cachedSiteRepository{
		base:  base,
		store: store,
		ttl:   time.Minute,
	}
	if _, err := repository.List(context.Background()); err != nil {
		t.Fatalf("cache read/write error escaped: %v", err)
	}
	if base.listCalls != 1 {
		t.Fatalf("site list calls = %d", base.listCalls)
	}
	if _, err := repository.Update(
		context.Background(),
		nil,
		base.items[0],
	); err != nil {
		t.Fatalf("cache invalidation error escaped: %v", err)
	}
}

func TestCachedSiteRepositoryDoesNotInvalidateFailedMutation(t *testing.T) {
	store := newMemoryCacheStore()
	updateErr := errors.New("database unavailable")
	base := &siteRepositoryStub{updateErr: updateErr}
	repository := &cachedSiteRepository{
		base:  base,
		store: store,
		ttl:   time.Minute,
	}
	if _, err := repository.Update(
		context.Background(),
		nil,
		site.Site{ID: 1},
	); !errors.Is(err, updateErr) {
		t.Fatalf("update error = %v", err)
	}
	if len(store.invalidated) != 0 {
		t.Fatalf("failed update invalidated tags: %v", store.invalidated)
	}
}

func TestCachedResourceRepositoryKeysTagsAndInvalidation(t *testing.T) {
	store := newMemoryCacheStore()
	base := &resourceRepositoryStub{
		item: resource.Resource{
			ID:       7,
			SiteID:   3,
			Title:    "cached",
			Settings: map[string]any{"count": json.Number("2")},
		},
	}
	repository := &cachedResourceRepository{
		base:  base,
		store: store,
		ttl:   5 * time.Minute,
	}

	if _, err := repository.ByID(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ByID(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if base.byIDCalls != 1 {
		t.Fatalf("resource ByID calls = %d", base.byIDCalls)
	}
	key := "core:resource:id:v1:7"
	if !reflect.DeepEqual(
		store.options[key].Tags,
		[]cache.Tag{siteTag(3), resourceTag(7)},
	) {
		t.Fatalf("resource tags = %v", store.options[key].Tags)
	}

	store.invalidated = nil
	created, err := repository.Create(
		context.Background(),
		nil,
		base.item,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 7 {
		t.Fatalf("created resource = %#v", created)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{siteTag(3), resourceTag(7)},
	) {
		t.Fatalf("create invalidated = %v", store.invalidated)
	}

	store.invalidated = nil
	if err := repository.Delete(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{siteTag(3), resourceTag(7)},
	) {
		t.Fatalf("delete invalidated = %v", store.invalidated)
	}
}

type memoryCacheStore struct {
	values        map[string][]byte
	options       map[string]cache.SetOptions
	invalidated   []cache.Tag
	getErr        error
	setErr        error
	invalidateErr error
}

func newMemoryCacheStore() *memoryCacheStore {
	return &memoryCacheStore{
		values:  make(map[string][]byte),
		options: make(map[string]cache.SetOptions),
	}
}

func (*memoryCacheStore) Code() cache.Code {
	return "test"
}

func (*memoryCacheStore) Ping(context.Context) error {
	return nil
}

func (s *memoryCacheStore) Get(
	_ context.Context,
	key string,
) ([]byte, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	value, exists := s.values[key]
	if !exists {
		return nil, cache.ErrMiss
	}
	return append([]byte(nil), value...), nil
}

func (s *memoryCacheStore) Set(
	_ context.Context,
	key string,
	value []byte,
	options cache.SetOptions,
) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.values[key] = append([]byte(nil), value...)
	options.Tags = append([]cache.Tag(nil), options.Tags...)
	s.options[key] = options
	return nil
}

func (s *memoryCacheStore) Exists(
	ctx context.Context,
	key string,
) (bool, error) {
	_, err := s.Get(ctx, key)
	if errors.Is(err, cache.ErrMiss) {
		return false, nil
	}
	return err == nil, err
}

func (s *memoryCacheStore) Delete(
	_ context.Context,
	key string,
) error {
	delete(s.values, key)
	return nil
}

func (s *memoryCacheStore) InvalidateTag(
	_ context.Context,
	tag cache.Tag,
) error {
	s.invalidated = append(s.invalidated, tag)
	return s.invalidateErr
}

func (*memoryCacheStore) Close() error {
	return nil
}

type siteRepositoryStub struct {
	items     []site.Site
	listCalls int
	updateErr error
}

func (r *siteRepositoryStub) List(
	context.Context,
) ([]site.Site, error) {
	r.listCalls++
	result := make([]site.Site, len(r.items))
	copy(result, r.items)
	return result, nil
}

func (r *siteRepositoryStub) Update(
	_ context.Context,
	_ *security.UserID,
	item site.Site,
) (site.Site, error) {
	if r.updateErr != nil {
		return site.Site{}, r.updateErr
	}
	r.items = []site.Site{item}
	return item, nil
}

type resourceRepositoryStub struct {
	item      resource.Resource
	byIDCalls int
	deleteErr error
}

func (r *resourceRepositoryStub) Create(
	context.Context,
	*security.UserID,
	resource.Resource,
	resource.ValidateImageMedia,
) (resource.Resource, error) {
	return r.item, nil
}

func (r *resourceRepositoryStub) ByID(
	context.Context,
	resource.ID,
) (resource.Resource, error) {
	r.byIDCalls++
	return r.item, nil
}

func (r *resourceRepositoryStub) ByPath(
	context.Context,
	site.ID,
	string,
) (resource.Resource, error) {
	return r.item, nil
}

func (r *resourceRepositoryStub) ListBySite(
	context.Context,
	site.ID,
) ([]resource.Resource, error) {
	return []resource.Resource{r.item}, nil
}

func (r *resourceRepositoryStub) Update(
	context.Context,
	*security.UserID,
	resource.Resource,
	resource.Resource,
	resource.ValidateImageMedia,
) (resource.Resource, error) {
	return r.item, nil
}

func (r *resourceRepositoryStub) Delete(
	context.Context,
	resource.ID,
) error {
	return r.deleteErr
}

var _ cache.Store = (*memoryCacheStore)(nil)
var _ site.Repository = (*siteRepositoryStub)(nil)
var _ resource.Repository = (*resourceRepositoryStub)(nil)
