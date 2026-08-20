---
name: go-cms-admin-ui
description: Use for GO CMS admin-panel extensibility work involving backend-driven navigation, admin UI manifests, module-contributed admin menu items, frontend admin plugins, module-owned routes/components/fields/tabs, permission-aware menu composition, and site-scoped admin customization. Preserves separation between backend availability/composition and frontend rendering.
---

# GO CMS Admin UI

## Purpose

Use this skill for any task that changes how GO CMS modules extend the admin panel.

Always follow repository root `AGENTS.md` and `.codex/skills/go-cms-development/SKILL.md` first. This skill adds admin-UI-specific rules.

The admin extension system must remain:

- module-owned;
- backend-driven for availability and navigation composition;
- frontend-driven for actual Vue rendering and interaction;
- site/profile-scoped where module availability depends on the current site;
- independent from direct dependencies from `admin` to feature modules;
- permission-aware on the backend;
- explicit and declarative;
- free from a mutable global active-site/menu singleton;
- compatible with frontend modules distributed as separate npm packages/repositories.

## 1. Inspect Before Editing

Before changing admin extensibility code, inspect the current implementation of:

- `backend/kernel/runtime.go`;
- `backend/kernel/modules/admin/`;
- `backend/kernel/modules/core/`;
- site/runtime resolution and admin HTTP routing;
- permission/authorization contracts;
- `frontend-admin/src/components/AdminDashboard.vue`;
- `frontend-admin/src/router.ts`;
- `frontend-admin/src/api/`;
- `frontend-admin/src/types/`;
- `frontend-admin/package.json`;
- current admin tests.

Do not redesign from memory.

Work against the branch explicitly requested by the user. For this project, when the task says `main`, inspect and implement against `main`.

## 2. Core Separation: Backend Decides What, Frontend Decides How

The backend is the source of truth for:

- which admin navigation items are currently available;
- which modules contribute them;
- nesting/order;
- permission-based visibility;
- whether an item is global or site-dependent;
- semantic route identifiers and semantic icon identifiers.

The frontend is the source of truth for:

- Vue components;
- Vue Router route records;
- Element Plus rendering;
- concrete icons;
- forms/fields;
- tabs;
- module-specific interaction logic.

Do not send Vue component names, Element Plus component names, import paths, or executable JavaScript from the backend.

Bad backend payload:

```json
{
  "component": "FormsListView",
  "icon": "UserFilled"
}
```

Preferred backend payload:

```json
{
  "code": "forms",
  "label": "Формы",
  "route": "forms.list",
  "icon": "forms"
}
```

`route` and `icon` are semantic identifiers. The frontend plugin resolves them.

## 3. Admin Must Not Depend on Feature Modules

`admin` may depend on mandatory lower-level capabilities such as `core` when required by the current architecture.

It must not gain direct dependencies on optional feature modules only to discover UI contributions.

Bad:

```go
func (Module) Dependencies() []kernel.ModuleCode {
    return []kernel.ModuleCode{
        core.ModuleCode,
        forms.ModuleCode,
        seo.ModuleCode,
        search.ModuleCode,
    }
}
```

Preferred direction:

```text
forms runtime ─┐
seo runtime ───┼─ implements generic admin UI capability
search runtime ┘

admin/navigation composer
    -> inspects generic capability
    -> does not import feature packages
```

Adding a new feature module must not require modifying the admin module.

## 4. Navigation Is a Generic Capability

Introduce a small generic admin-UI contract rather than expanding the mandatory `kernel.Module` interface.

Preferred conceptual shape:

```go
type NavigationItem struct {
    Code       string
    Label      string
    Route      string
    Icon       string
    Order      int
    Permission permission.Code
    Scope      NavigationScope
    Children   []NavigationItem
}

type NavigationProvider interface {
    AdminNavigation() []NavigationItem
}
```

Exact package/type names may follow repository conventions after inspection.

Keep the capability optional. A module with no admin navigation implements nothing.

Do not make every module implement empty admin methods.

## 5. Ownership and Scope

Navigation has two different scopes:

```text
global
site
```

### Global items

Examples:

- Sites;
- Files;
- Users;
- Groups;
- application-level settings.

They must remain available when semantically appropriate even if no site is currently selected.

### Site items

Examples:

- Forms;
- SEO;
- site-specific search/configuration;
- module screens that only make sense when the selected site enables that module.

Site-scoped items must be derived from the selected site's final runtime/profile, not from every package installed in the application.

Preserve the project rule:

```text
installed package != module enabled for this site
```

Do not use a global mutable menu registry tied to the "current site".

If the existing runtime architecture requires separate collection paths for global and site contributions, implement that explicitly rather than faking global state through an arbitrary site runtime.

## 6. Prefer Runtime Contributions for Runtime-Dependent UI

If a navigation contribution can depend on site/module configuration, expose it from the final module runtime or another site-scoped compiled provider.

This permits:

```text
Site A -> Forms -> Forms / Responses / Templates
Site B -> Forms -> Forms only
```

Do not rebuild the SiteRuntime on each HTTP request merely to compute navigation.

The final site runtime is assembled/reloaded outside the request hot path and reused.

Static application-wide contributions may be compiled once at application/profile composition time if that better matches their lifetime.

## 7. Navigation Validation

The reusable navigation compiler/composer should validate at least:

- empty item codes;
- duplicate sibling/global codes according to the chosen identity rule;
- invalid/empty labels where required;
- invalid order values if constrained;
- invalid scope;
- invalid permission references when validation is possible;
- malformed child trees;
- deterministic ordering.

Do not rely on Go map iteration order.

Use stable ordering, for example:

```text
Order ascending
then Code ascending as deterministic tie-breaker
```

Exact rule may follow repository conventions.

## 8. Permissions

Navigation visibility must be filtered on the backend using the current actor/authorizer.

A user who cannot access a child item should not receive that child in the navigation response.

If a parent is only a group and all children are removed, remove the empty parent.

Frontend visibility is usability only, not security.

Every target API endpoint must continue to enforce its own authorization independently of navigation visibility.

Do not weaken existing permission middleware/handlers because a menu endpoint already filters items.

## 9. Backend API

Expose a dedicated admin navigation/manifest endpoint rather than embedding menu knowledge in Vue.

Preferred direction:

```http
GET /api/admin/navigation
```

or, if explicit site selection is required by existing APIs:

```http
GET /api/admin/navigation?site_id=<id>
```

Follow the current admin API conventions after inspection.

The response should be frontend-neutral, for example:

```json
{
  "items": [
    {
      "code": "sites",
      "label": "Сайты",
      "route": "core.sites",
      "icon": "sites",
      "order": 100,
      "scope": "global"
    },
    {
      "code": "forms",
      "label": "Формы",
      "route": "",
      "icon": "forms",
      "order": 400,
      "scope": "site",
      "children": [
        {
          "code": "forms.list",
          "label": "Формы",
          "route": "forms.list",
          "order": 100,
          "scope": "site"
        }
      ]
    }
  ]
}
```

Do not expose raw Go type information or implementation package paths.

## 10. Core Navigation Must Move Out of Vue Hardcode

The existing built-in items such as:

- Sites;
- Filesystem;
- Users;
- Groups;

must be represented by backend navigation contributions and returned by the navigation API.

The top-level Vue shell must render received navigation generically rather than contain module-specific `v-if="can(...)"` links for these items.

Module-specific menu additions must not require editing `AdminDashboard.vue`.

## 11. Frontend Admin Plugin Contract

The admin frontend shell needs an explicit plugin contract for compiled frontend extensions.

Preferred conceptual shape:

```ts
export interface AdminPlugin {
  code: string
  routes?: AdminRouteDefinition[]
  icons?: Record<string, Component>
  fields?: AdminFieldDefinition[]
  resourceTabs?: AdminResourceTabDefinition[]
}
```

Start with the smallest contract required by the task. Do not add unused extension categories merely because they may exist someday.

The plugin `code` must correspond to the backend module/plugin identity where appropriate.

A frontend plugin owns the concrete implementation of semantic route identifiers returned by the backend.

Example:

```ts
{
  code: 'forms',
  routes: [
    {
      name: 'forms.list',
      path: '/admin/forms',
      component: FormsListView,
    },
  ],
}
```

The backend knows `forms.list`. It does not know `/admin/forms` or `FormsListView`.

## 12. Frontend Module Distribution

Feature admin frontends are expected to be distributable as independent npm packages, potentially stored in separate Git repositories.

Preferred future package shape:

```text
@go-cms/admin-forms
@go-cms/admin-seo
@go-cms/admin-search
```

A project that uses them declares them in `frontend-admin/package.json` or the final project's admin package:

```json
{
  "dependencies": {
    "@go-cms/admin-forms": "1.2.0",
    "@go-cms/admin-seo": "1.1.0"
  }
}
```

Prefer a registry/GitHub Packages with semver for production use. Direct Git dependencies may be supported during development.

Do not implement runtime remote-JS loading, Module Federation, or a microfrontend architecture unless explicitly requested.

The intended MVP rule is:

```text
new frontend code/package -> requires npm install + frontend rebuild
backend navigation/config change -> does not require frontend rebuild
enable/disable already-installed module for a site -> does not require frontend rebuild
```

## 13. Plugin Registration Must Be Declarative

The shell should have one obvious composition point for installed frontend plugins.

Conceptually:

```ts
createAdminApp({
  plugins: [
    coreAdminPlugin,
    formsAdminPlugin,
    seoAdminPlugin,
  ],
})
```

or another repository-consistent equivalent.

Do not require every plugin to edit:

- router internals;
- the dashboard component;
- several registries;
- several bootstrap files.

Adding one installed frontend plugin should normally require one dependency plus one declarative registration/import point unless the build system can safely automate that without magic scanning.

## 14. Dynamic Routes

Current Vue routes may be statically declared, but module routes should be registered through the plugin mechanism.

The base shell may keep shell-owned routes such as:

- login/session bootstrap;
- profile;
- dashboard;
- fallback/not-found;

if they truly belong to the shell/admin application.

Module-owned screens should move behind plugin route definitions.

Do not accept a backend-supplied arbitrary URL/component import as an executable route definition.

The backend selects a semantic route ID; installed frontend plugins resolve it.

## 15. Missing Frontend Plugin or Route

Backend and frontend packages can be version-mismatched.

Handle a navigation item whose semantic route is unavailable in the current frontend build deliberately.

Do not crash the whole admin app.

Preferred behavior:

- detect unresolved route IDs;
- omit/disable the item or show a controlled unsupported-feature state;
- log a clear development warning;
- keep the rest of the admin usable.

Do not silently route to an unrelated screen.

## 16. Site Switching

When the selected site changes:

1. keep global navigation available;
2. reload/recompute site-scoped navigation through the backend;
3. remove items belonging to modules not enabled for the newly selected site;
4. preserve deterministic menu order;
5. do not rebuild or reload the whole frontend application unnecessarily.

Avoid stale site-specific navigation after switching sites.

## 17. Icons

Backend icon values are semantic codes:

```text
sites
files
users
forms
seo
```

Concrete Vue/Element Plus icon components are resolved in frontend code.

Do not make the backend depend on `@element-plus/icons-vue` names.

If an icon code is unknown, render a safe generic fallback.

## 18. Do Not Turn Navigation Into Authorization or Routing Source of Truth

Navigation is a discoverability/UI manifest.

It is not:

- an authorization engine;
- a backend HTTP route registry;
- an executable frontend module loader;
- a replacement for Vue Router;
- a replacement for module HTTP registration.

Keep those responsibilities separate.

## 19. Tests

Add focused backend tests for:

- contribution collection;
- deterministic ordering;
- nested items;
- duplicate/invalid definitions;
- permission filtering;
- empty parent removal;
- global items without a selected site when applicable;
- site-specific module availability;
- switching between runtimes/profiles if the relevant layer is testable.

Add frontend tests for:

- generic top-menu rendering;
- nested menu rendering;
- semantic route resolution;
- active state;
- site navigation refresh;
- missing route/plugin behavior;
- removal of hard-coded core menu assumptions.

Do not over-test Element Plus internals.

## 20. Validation

For Go changes, run from `backend/` when practical:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

For frontend changes, run from `frontend-admin/`:

```sh
npm test
npm run typecheck
npm run build
```

Use the package manager/lockfile already used by the repository.

Never claim a command succeeded unless it was actually run successfully.

## 21. Review Checklist

Before finishing, verify:

- `admin` does not import optional feature modules;
- feature modules can contribute navigation through a generic capability;
- site items only come from modules enabled for that site;
- global items are not accidentally lost when no site is selected;
- backend filters navigation by permissions;
- target APIs still enforce permissions independently;
- Vue no longer hard-codes module-specific top-menu items covered by the new system;
- backend sends semantic route/icon IDs, not Vue implementation names;
- frontend plugins own concrete routes/components/icons;
- installed frontend packages are registered declaratively;
- no mutable global active-site/menu registry was introduced;
- no runtime remote-JS/microfrontend machinery was added;
- current unrelated admin behavior remains unchanged.

## 22. Final Response

After implementation, report concisely:

1. backend admin UI/navigation contracts added;
2. how global and site-scoped navigation are composed;
3. how permission filtering works;
4. how frontend plugins/routes are registered;
5. which current hard-coded menu/routes were migrated;
6. important files changed;
7. tests/build commands actually run and their results;
8. any intentionally deferred package extraction to separate Git/npm repositories.
