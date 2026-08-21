package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type managementSiteRepository struct {
	query site.ListQuery
	page  site.Page
	err   error
}

func (r *managementSiteRepository) List(context.Context) ([]site.Site, error) {
	return append([]site.Site(nil), r.page.Items...), nil
}
func (r *managementSiteRepository) FindByID(context.Context, site.ID) (site.Site, error) {
	return site.Site{}, site.ErrNotFound
}
func (r *managementSiteRepository) FindByDomain(context.Context, string) (site.Site, error) {
	return site.Site{}, site.ErrNotFound
}
func (r *managementSiteRepository) ListPage(_ context.Context, query site.ListQuery) (site.Page, error) {
	r.query = query
	return r.page, r.err
}
func (r *managementSiteRepository) Create(context.Context, *security.UserID, site.Site) (site.Site, error) {
	return site.Site{}, nil
}
func (r *managementSiteRepository) Update(context.Context, *security.UserID, site.Site) (site.Site, error) {
	return site.Site{}, nil
}
func (r *managementSiteRepository) Delete(context.Context, site.ID) error { return nil }

type managementAuthorizer struct {
	denied map[permission.Code]error
}

func (a managementAuthorizer) Check(_ context.Context, _ security.Actor, code permission.Code) error {
	return a.denied[code]
}

type scopedPolicy struct {
	scope  SiteAccessScope
	scopes map[SiteAccessAction]SiteAccessScope
	checks map[SiteAccessAction]error
}

func (p scopedPolicy) Scope(_ context.Context, _ security.Actor, action SiteAccessAction) (SiteAccessScope, error) {
	if scope, exists := p.scopes[action]; exists {
		return scope, nil
	}
	return p.scope, nil
}
func (p scopedPolicy) Check(_ context.Context, _ security.Actor, _ site.ID, action SiteAccessAction) error {
	return p.checks[action]
}

func TestManagementRequireSiteKeepsGlobalAndSiteChecksIndependent(t *testing.T) {
	t.Parallel()
	management := &Management{
		authorizer: managementAuthorizer{denied: map[permission.Code]error{ResourceReadPermission: security.ErrForbidden}},
		policy:     scopedPolicy{checks: map[SiteAccessAction]error{SiteAccessEdit: nil}},
	}
	if err := management.requireSite(context.Background(), security.User(1), 7, ResourceReadPermission, SiteAccessEdit); !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("global permission error = %v", err)
	}
	management.authorizer = managementAuthorizer{denied: map[permission.Code]error{}}
	management.policy = scopedPolicy{checks: map[SiteAccessAction]error{SiteAccessEdit: security.ErrForbidden}}
	if err := management.requireSite(context.Background(), security.User(1), 7, ResourceReadPermission, SiteAccessEdit); !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("site access error = %v", err)
	}
}

func TestManagementUsesViewForCatalogEditForOptionsAndReturnsCapabilities(t *testing.T) {
	t.Parallel()
	repository := &managementSiteRepository{page: site.Page{Items: []site.Site{
		{ID: 1, Domain: "view.example", ProfileCode: "dev", Locale: "ru-RU"},
		{ID: 2, Domain: "edit.example", ProfileCode: "dev", Locale: "ru-RU"},
	}, Total: 2}}
	management := &Management{
		repository: repository,
		authorizer: managementAuthorizer{denied: map[permission.Code]error{}},
		policy: scopedPolicy{scopes: map[SiteAccessAction]SiteAccessScope{
			SiteAccessView:   {SiteIDs: []site.ID{1, 2}},
			SiteAccessEdit:   {SiteIDs: []site.ID{2}},
			SiteAccessDelete: {SiteIDs: []site.ID{2}},
		}},
	}
	list, err := management.ListSites(context.Background(), security.User(1), "", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !list.Items[0].Capabilities.View || list.Items[0].Capabilities.Edit ||
		!list.Items[1].Capabilities.Edit || !list.Items[1].Capabilities.Delete {
		t.Fatalf("capabilities = %#v", list.Items)
	}
	if _, err := management.ListSiteOptions(context.Background(), security.User(1), "", 1, 10); err != nil {
		t.Fatal(err)
	}
	if repository.query.Scope.All || len(repository.query.Scope.SiteIDs) != 1 || repository.query.Scope.SiteIDs[0] != 2 {
		t.Fatalf("option scope = %#v", repository.query.Scope)
	}
}

func TestManagementListSitesAppliesDefaultsScopeAndPermissions(t *testing.T) {
	t.Parallel()
	repository := &managementSiteRepository{page: site.Page{
		Items: []site.Site{{ID: 7, Domain: "example.com", ProfileCode: "dev", Locale: "ru-RU"}},
		Total: 1,
	}}
	management := &Management{
		repository: repository,
		authorizer: managementAuthorizer{denied: map[permission.Code]error{
			SiteDeletePermission: security.ErrForbidden,
		}},
		policy: scopedPolicy{scope: SiteAccessScope{SiteIDs: []site.ID{7}}},
	}

	result, err := management.ListSites(context.Background(), security.User(1), " example ", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if repository.query.Page != 1 || repository.query.PerPage != 10 ||
		repository.query.Scope.All || len(repository.query.Scope.SiteIDs) != 1 ||
		repository.query.Scope.SiteIDs[0] != 7 {
		t.Fatalf("query = %#v", repository.query)
	}
	if len(result.Items) != 1 || result.Items[0].Domain != "example.com" ||
		!result.Permissions.Read || !result.Permissions.Create ||
		!result.Permissions.Update || result.Permissions.Delete {
		t.Fatalf("result = %#v", result)
	}
}

func TestManagementListSitesRejectsPermissionAndPagination(t *testing.T) {
	t.Parallel()
	management := &Management{
		repository: &managementSiteRepository{},
		authorizer: managementAuthorizer{denied: map[permission.Code]error{
			SiteReadPermission: security.ErrForbidden,
		}},
		policy: scopedPolicy{},
	}
	if _, err := management.ListSites(context.Background(), security.User(1), "", 1, 10); !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("permission error = %v", err)
	}
	management.authorizer = managementAuthorizer{denied: map[permission.Code]error{}}
	if _, err := management.ListSites(context.Background(), security.User(1), "", -1, 10); !errors.Is(err, ErrValidation) {
		t.Fatalf("pagination error = %v", err)
	}
}

func TestResourceTreeItemUsesSafeTitleAndIconFallbacks(t *testing.T) {
	t.Parallel()
	page := treeItem(nil, resource.Child{
		ID:        1,
		Type:      resourcetype.Page,
		Title:     "Title",
		MenuTitle: " Menu title ",
	}, false)
	if page.DisplayTitle != "Menu title" || page.Icon != "document" || page.CanCreateChild {
		t.Fatalf("page item = %#v", page)
	}
	link := treeItem(nil, resource.Child{ID: 2, Type: resourcetype.Link, Title: "Link"}, true)
	if link.DisplayTitle != "Link" || link.Icon != "link" || !link.CanCreateChild {
		t.Fatalf("link item = %#v", link)
	}
	if iconOrDefault("unsafe/path") != "document" {
		t.Fatal("unsafe icon did not fall back to document")
	}
}

func TestResourceTreeItemReportsEffectivePublication(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	cases := []struct {
		name string
		item resource.Child
		want bool
	}{
		{name: "public", item: resource.Child{IsPublic: true}, want: true},
		{name: "private", item: resource.Child{IsPublic: false}},
		{name: "starts later", item: resource.Child{IsPublic: true, PublishedAt: &future}},
		{name: "already ended", item: resource.Child{IsPublic: true, UnpublishedAt: &past}},
		{name: "deleted", item: resource.Child{IsPublic: true, DeletedAt: &past}},
	}
	for _, current := range cases {
		t.Run(current.name, func(t *testing.T) {
			if got := isPublished(current.item); got != current.want {
				t.Fatalf("published = %v, want %v", got, current.want)
			}
		})
	}
}

var _ site.ManagementRepository = (*managementSiteRepository)(nil)
var _ security.Authorizer = managementAuthorizer{}
var _ SiteAccessPolicy = scopedPolicy{}
