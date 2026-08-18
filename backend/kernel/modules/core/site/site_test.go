package site

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type testResolver struct{}

type testEventBus struct{}

func (testEventBus) Publish(context.Context, eventbus.Message) error {
	return nil
}

func (testEventBus) Consume(
	context.Context,
	eventbus.Subscription,
	eventbus.Handler,
) error {
	return nil
}

func (testResolver) MainModuleDatabase(
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}
func (testResolver) ModuleDatabase(
	kernel.ConnectionCode,
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}

type testProfiles map[kernel.ProfileCode]*kernel.ProfileBlueprint

func (p testProfiles) ProfileBlueprint(
	code kernel.ProfileCode,
) (*kernel.ProfileBlueprint, bool) {
	blueprint, exists := p[code]
	return blueprint, exists
}

type testAccess struct {
	allow bool
}

func (a testAccess) Check(
	context.Context,
	security.Actor,
	permission.Code,
) error {
	if !a.allow {
		return security.ErrForbidden
	}
	return nil
}
func (testAccess) IsGuestSubject(
	_ context.Context,
	actor security.Actor,
) (bool, error) {
	return actor.IsGuest(), nil
}

type memoryRepository struct {
	mu    sync.Mutex
	items []Site
}

func (r *memoryRepository) List(
	context.Context,
) ([]Site, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Site(nil), r.items...), nil
}

func (r *memoryRepository) Update(
	_ context.Context,
	actorID *security.UserID,
	item Site,
) (Site, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.items {
		if r.items[index].ID != item.ID {
			continue
		}
		item.CreatedAt = r.items[index].CreatedAt
		item.UpdatedAt = time.Now().UTC()
		item.CreatedBy = r.items[index].CreatedBy
		item.UpdatedBy = cloneUserID(actorID)
		r.items[index] = item
		return item, nil
	}
	return Site{}, ErrNotFound
}

func (r *memoryRepository) FindByID(_ context.Context, id ID) (Site, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return Site{}, ErrNotFound
}

func (r *memoryRepository) FindByDomain(_ context.Context, domain string) (Site, error) {
	for _, item := range r.items {
		if item.Domain == domain {
			return item, nil
		}
	}
	return Site{}, ErrNotFound
}

func (r *memoryRepository) ListPage(_ context.Context, query ListQuery) (Page, error) {
	return Page{Items: append([]Site(nil), r.items...), Total: len(r.items)}, nil
}

func (r *memoryRepository) Create(_ context.Context, actorID *security.UserID, item Site) (Site, error) {
	item.ID = ID(len(r.items) + 1)
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	item.CreatedBy = cloneUserID(actorID)
	item.UpdatedBy = cloneUserID(actorID)
	r.items = append(r.items, item)
	return item, nil
}

func (r *memoryRepository) Delete(_ context.Context, id ID) error {
	for index, item := range r.items {
		if item.ID == id {
			r.items = append(r.items[:index], r.items[index+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func newCatalogForTest(
	t *testing.T,
	item Site,
	access Access,
) *Catalog {
	t.Helper()
	factory, err := kernel.NewProfileRuntimeFactory(
		testResolver{},
		kernel.RuntimeServices{
			EventBus: testEventBus{},
			Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := factory.Compile(
		context.Background(),
		kernel.Profile{Code: item.ProfileCode},
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(
		&memoryRepository{items: []Site{item}},
		testProfiles{item.ProfileCode: profile},
		access,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestResolveRequiresPermissionAndPublicGuestSite(t *testing.T) {
	t.Parallel()

	item := Site{
		ID:          1,
		ProfileCode: "test",
		Domain:      "example.com",
		Locale:      "en-US",
		IsPublic:    true,
	}
	allowed := newCatalogForTest(t, item, testAccess{allow: true})
	if _, err := allowed.ResolveByDomain(
		context.Background(),
		security.Guest(),
		item.Domain,
	); err != nil {
		t.Fatal(err)
	}

	privateItem := item
	privateItem.IsPublic = false
	private := newCatalogForTest(
		t,
		privateItem,
		testAccess{allow: true},
	)
	if _, err := private.ResolveByDomain(
		context.Background(),
		security.Guest(),
		item.Domain,
	); !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("private guest error = %v", err)
	}

	denied := newCatalogForTest(t, item, testAccess{})
	if _, err := denied.ResolveByDomain(
		context.Background(),
		security.Guest(),
		item.Domain,
	); !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("permission error = %v", err)
	}
}

func TestUpdateAtomicallyReplacesDomainSnapshotAndAudit(t *testing.T) {
	t.Parallel()

	catalog := newCatalogForTest(t, Site{
		ID:          1,
		ProfileCode: "test",
		Domain:      "old.example.com",
		Locale:      "en-US",
		IsPublic:    false,
	}, testAccess{allow: true})
	updated, err := catalog.Update(
		context.Background(),
		security.User(42),
		UpdateInput{
			ID:          1,
			ProfileCode: "test",
			Domain:      "NEW.EXAMPLE.COM.",
			Locale:      "ru-RU",
			Settings:    map[string]any{},
			IsPublic:    true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Site().Domain != "new.example.com" ||
		updated.Site().Locale != "ru-RU" ||
		!updated.Site().IsPublic ||
		updated.Site().UpdatedBy == nil ||
		*updated.Site().UpdatedBy != 42 {
		t.Fatalf("updated site = %#v", updated.Site())
	}
	if _, exists := catalog.RuntimeByDomain("old.example.com"); exists {
		t.Fatal("old domain remains in snapshot")
	}
	if current, exists := catalog.RuntimeByDomain(
		"new.example.com",
	); !exists || current != updated {
		t.Fatalf("new domain runtime = %#v, %t", current, exists)
	}
}

func TestCreateAndDeleteAtomicallyUpdateSnapshot(t *testing.T) {
	t.Parallel()
	catalog := newCatalogForTest(t, Site{
		ID: 1, ProfileCode: "test", Domain: "existing.test", Locale: "en-US",
	}, testAccess{allow: true})
	created, err := catalog.Create(context.Background(), security.User(42), CreateInput{
		ProfileCode: "test",
		Domain:      "NEW.TEST.",
		Locale:      "ru-RU",
		Settings:    map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Site().ID != 2 || created.Site().Domain != "new.test" ||
		created.Site().CreatedBy == nil || *created.Site().CreatedBy != 42 {
		t.Fatalf("created site = %#v", created.Site())
	}
	if current, exists := catalog.RuntimeByID(2); !exists || current != created {
		t.Fatalf("created runtime = %#v, %t", current, exists)
	}
	if err := catalog.Delete(context.Background(), security.User(42), 2); err != nil {
		t.Fatal(err)
	}
	if _, exists := catalog.RuntimeByID(2); exists {
		t.Fatal("deleted site remains in snapshot")
	}
}

func TestReloadPreparesRuntimesBeforeAtomicSnapshotPublication(t *testing.T) {
	item := Site{
		ID:          1,
		ProfileCode: "test",
		Domain:      "first.example.com",
		Locale:      "en-US",
		IsPublic:    true,
	}
	repository := &memoryRepository{items: []Site{item}}
	factory, err := kernel.NewProfileRuntimeFactory(
		testResolver{},
		kernel.RuntimeServices{
			EventBus: testEventBus{},
			Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	blueprint, err := factory.Compile(
		context.Background(),
		kernel.Profile{Code: item.ProfileCode},
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(
		repository,
		testProfiles{item.ProfileCode: blueprint},
		testAccess{allow: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	prepared := make(map[ID]int)
	if err := catalog.AddRuntimePreparer(
		context.Background(),
		func(_ context.Context, runtime *Runtime) error {
			id := runtime.Site().ID
			prepared[id]++
			if id == 3 {
				return errors.New("compile failed")
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if prepared[1] != 1 {
		t.Fatalf("initial runtime prepare calls = %#v", prepared)
	}

	repository.items = []Site{{
		ID:          2,
		ProfileCode: "test",
		Domain:      "second.example.com",
		Locale:      "en-US",
		IsPublic:    true,
	}}
	if err := catalog.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if prepared[2] != 1 {
		t.Fatalf("replacement runtime prepare calls = %#v", prepared)
	}

	repository.items = []Site{{
		ID:          3,
		ProfileCode: "test",
		Domain:      "broken.example.com",
		Locale:      "en-US",
		IsPublic:    true,
	}}
	if err := catalog.Reload(context.Background()); err == nil {
		t.Fatal("runtime preparation failure was ignored")
	}
	if _, exists := catalog.RuntimeByID(2); !exists {
		t.Fatal("failed runtime preparation replaced the previous snapshot")
	}
}

var _ Repository = (*memoryRepository)(nil)
var _ ManagementRepository = (*memoryRepository)(nil)
var _ Access = testAccess{}
