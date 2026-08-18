package core

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
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
	policy := newRepositoryCachePolicy()
	policy.register(
		RepositoryCacheDescriptor{Code: store.Code(), Namespace: "test"},
		store,
	)
	repository := &cachedSiteRepository{
		base:   &invalidatingSiteRepository{base: base, policy: policy},
		store:  store,
		ttl:    5 * time.Minute,
		policy: policy,
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

	store.invalidated = nil
	created, err := repository.Create(
		context.Background(),
		nil,
		site.Site{ID: 2, Domain: "created.example.test"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{sitesTag, siteTag(created.ID)},
	) {
		t.Fatalf("create invalidated tags = %v", store.invalidated)
	}

	store.invalidated = nil
	if err := repository.Delete(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{sitesTag, siteTag(created.ID)},
	) {
		t.Fatalf("delete invalidated tags = %v", store.invalidated)
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
	policy := newRepositoryCachePolicy()
	policy.register(
		RepositoryCacheDescriptor{Code: store.Code(), Namespace: "test"},
		store,
	)
	repository := &cachedSiteRepository{
		base:   &invalidatingSiteRepository{base: base, policy: policy},
		store:  store,
		ttl:    time.Minute,
		policy: policy,
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
	policy := newRepositoryCachePolicy()
	policy.register(
		RepositoryCacheDescriptor{Code: store.Code(), Namespace: "test"},
		store,
	)
	repository := &cachedSiteRepository{
		base:   &invalidatingSiteRepository{base: base, policy: policy},
		store:  store,
		ttl:    time.Minute,
		policy: policy,
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
			Widgets: []resource.WidgetBinding{{
				Code:     widget.Code("content_summary"),
				Position: 0,
				Params: map[string]any{
					"limit": json.Number("3"),
				},
			}},
		},
	}
	policy := newRepositoryCachePolicy()
	policy.register(
		RepositoryCacheDescriptor{Code: store.Code(), Namespace: "test"},
		store,
	)
	repository := &cachedResourceRepository{
		base: &invalidatingResourceRepository{
			base: base, policy: policy,
		},
		store:  store,
		ttl:    5 * time.Minute,
		policy: policy,
	}

	first, err := repository.ByID(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	first.Widgets[0].Params["limit"] = json.Number("99")
	second, err := repository.ByID(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if base.byIDCalls != 1 {
		t.Fatalf("resource ByID calls = %d", base.byIDCalls)
	}
	if len(second.Widgets) != 1 ||
		second.Widgets[0].Code != "content_summary" {
		t.Fatalf("cached widgets = %#v", second.Widgets)
	}
	if value, ok := second.Widgets[0].Params["limit"].(json.Number); !ok ||
		value.String() != "3" {
		t.Fatalf("cached widget params = %#v", second.Widgets[0].Params)
	}
	key := "core:resource:id:v2:7"
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
	updatedInput := base.item
	updatedInput.SiteID = 4
	updated, err := repository.Update(
		context.Background(),
		nil,
		base.item,
		updatedInput,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{siteTag(3), siteTag(4), resourceTag(7)},
	) {
		t.Fatalf("update invalidated = %v", store.invalidated)
	}

	store.invalidated = nil
	if err := repository.SoftDelete(context.Background(), nil, updated.ID); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{siteTag(4), resourceTag(7)},
	) {
		t.Fatalf("soft delete invalidated = %v", store.invalidated)
	}

	store.invalidated = nil
	if err := repository.Restore(context.Background(), nil, updated.ID, true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{siteTag(4), resourceTag(7)},
	) {
		t.Fatalf("restore invalidated = %v", store.invalidated)
	}

	store.invalidated = nil
	if err := repository.Delete(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		store.invalidated,
		[]cache.Tag{siteTag(4), resourceTag(7)},
	) {
		t.Fatalf("delete invalidated = %v", store.invalidated)
	}
}

func TestRepositoryCacheCoherencePreventsStaleReadFillAfterWrite(t *testing.T) {
	store := newMemoryCacheStore()
	base := &blockingResourceRepository{
		resourceRepositoryStub: &resourceRepositoryStub{item: resource.Resource{
			ID: 7, SiteID: 3, Title: "before",
		}},
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	policy := newRepositoryCachePolicy()
	policy.register(
		RepositoryCacheDescriptor{Code: store.Code(), Namespace: "test"},
		store,
	)
	repository := &cachedResourceRepository{
		base: &invalidatingResourceRepository{
			base: base, policy: policy,
		},
		store:  store,
		ttl:    time.Minute,
		policy: policy,
	}

	readDone := make(chan resource.Resource, 1)
	readErr := make(chan error, 1)
	go func() {
		result, err := repository.ByID(context.Background(), 7)
		readDone <- result
		readErr <- err
	}()
	<-base.started
	if policy.coherenceMu.TryLock() {
		policy.coherenceMu.Unlock()
		t.Fatal("cache fill did not hold the coherence read barrier")
	}

	writeDone := make(chan error, 1)
	go func() {
		_, err := repository.Update(
			context.Background(),
			nil,
			base.item,
			resource.Resource{ID: 7, SiteID: 3, Title: "after"},
			nil,
		)
		writeDone <- err
	}()
	close(base.release)
	if err := <-readErr; err != nil {
		t.Fatal(err)
	}
	if result := <-readDone; result.Title != "before" {
		t.Fatalf("in-flight read = %#v", result)
	}
	if err := <-writeDone; err != nil {
		t.Fatal(err)
	}

	fresh, err := repository.ByID(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Title != "after" {
		t.Fatalf("stale cache fill survived write: %#v", fresh)
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
	if s.invalidateErr != nil {
		return s.invalidateErr
	}
	for key, options := range s.options {
		for _, current := range options.Tags {
			if current != tag {
				continue
			}
			delete(s.values, key)
			delete(s.options, key)
			break
		}
	}
	return nil
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
	for index := range r.items {
		if r.items[index].ID == item.ID {
			r.items[index] = item
			return item, nil
		}
	}
	return item, nil
}

func (r *siteRepositoryStub) FindByID(
	_ context.Context,
	id site.ID,
) (site.Site, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return site.Site{}, site.ErrNotFound
}

func (r *siteRepositoryStub) FindByDomain(
	_ context.Context,
	domain string,
) (site.Site, error) {
	for _, item := range r.items {
		if item.Domain == domain {
			return item, nil
		}
	}
	return site.Site{}, site.ErrNotFound
}

func (r *siteRepositoryStub) ListPage(
	_ context.Context,
	_ site.ListQuery,
) (site.Page, error) {
	return site.Page{Items: append([]site.Site(nil), r.items...), Total: len(r.items)}, nil
}

func (r *siteRepositoryStub) Create(
	_ context.Context,
	_ *security.UserID,
	item site.Site,
) (site.Site, error) {
	r.items = append(r.items, item)
	return item, nil
}

func (r *siteRepositoryStub) Delete(
	_ context.Context,
	id site.ID,
) error {
	for index, item := range r.items {
		if item.ID == id {
			r.items = append(r.items[:index], r.items[index+1:]...)
			return nil
		}
	}
	return site.ErrNotFound
}

type resourceRepositoryStub struct {
	item      resource.Resource
	byIDCalls int
	deleteErr error
}

type blockingResourceRepository struct {
	*resourceRepositoryStub
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (r *blockingResourceRepository) ByID(
	context.Context,
	resource.ID,
) (resource.Resource, error) {
	result := r.item
	r.once.Do(func() {
		close(r.started)
		<-r.release
	})
	r.byIDCalls++
	return result, nil
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
	_ context.Context,
	_ *security.UserID,
	_ resource.Resource,
	item resource.Resource,
	_ resource.ValidateImageMedia,
) (resource.Resource, error) {
	r.item = item
	return item, nil
}

func (r *resourceRepositoryStub) Delete(
	context.Context,
	resource.ID,
) error {
	return r.deleteErr
}

func (r *resourceRepositoryStub) ExistsInSite(
	_ context.Context,
	siteID site.ID,
	id resource.ID,
) (bool, error) {
	return r.item.SiteID == siteID && r.item.ID == id, nil
}

func (r *resourceRepositoryStub) ListChildren(
	context.Context,
	site.ID,
	*resource.ID,
) ([]resource.Child, error) {
	return nil, nil
}

func (*resourceRepositoryStub) SoftDelete(
	context.Context,
	*security.UserID,
	resource.ID,
) error {
	return nil
}

func (*resourceRepositoryStub) Restore(
	context.Context,
	*security.UserID,
	resource.ID,
	bool,
) error {
	return nil
}

var _ cache.Store = (*memoryCacheStore)(nil)
var _ site.ManagementRepository = (*siteRepositoryStub)(nil)
var _ resource.ManagementRepository = (*resourceRepositoryStub)(nil)
var _ resource.LifecycleRepository = (*resourceRepositoryStub)(nil)
