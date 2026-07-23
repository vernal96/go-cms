package site

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type testResolver struct{}

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

type testProfiles map[kernel.ProfileCode]*kernel.ProfileRuntime

func (p testProfiles) ProfileRuntime(
	code kernel.ProfileCode,
) (*kernel.ProfileRuntime, bool) {
	runtime, exists := p[code]
	return runtime, exists
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

func newCatalogForTest(
	t *testing.T,
	item Site,
	access Access,
) *Catalog {
	t.Helper()
	factory, err := kernel.NewProfileRuntimeFactory(testResolver{})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := factory.Make(
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
			ID:       1,
			Domain:   "NEW.EXAMPLE.COM.",
			Locale:   "ru-RU",
			Settings: map[string]any{},
			IsPublic: true,
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

var _ Repository = (*memoryRepository)(nil)
var _ Access = testAccess{}
