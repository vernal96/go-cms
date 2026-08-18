---
name: go-cms-widgets
description: Use for GO CMS widget architecture and implementation work involving widget definitions, module widget providers, template widget layouts, resource widget bindings, widget rendering, body/sidebar areas, widget views, admin widget editing, drag-and-drop ordering, or public widget API composition. Preserves site-scoped runtimes and declarative project widget customization.
---

# GO CMS Widgets

## Purpose

Use this skill for any task that adds, changes, renders, stores, configures, or edits widgets in GO CMS.

Always follow repository root `AGENTS.md` and `.codex/skills/go-cms-development/SKILL.md` first. This skill adds widget-specific architectural rules.

The widget system must remain:

- module-owned;
- site/profile-scoped at runtime;
- declarative in `backend/internal/`;
- independent from concrete infrastructure technologies;
- composable from template widgets and resource widgets;
- explicit about presentation settings versus widget business parameters.

## 1. Inspect Before Editing

Before changing widget code, inspect the current implementation of:

- `backend/kernel/modules/core/widget/`;
- `backend/kernel/modules/core/template/`;
- `backend/kernel/modules/core/resource/`;
- profile/runtime assembly in `backend/kernel/runtime.go`;
- the owning module runtime and its `widget.Provider` implementation;
- resource persistence and migrations;
- admin resource metadata/update APIs;
- `frontend-admin/src/views/ResourceEditView.vue` and reusable dynamic-field components;
- the public resource/page response renderer.

Do not redesign from memory when the repository already contains part of the widget pipeline.

Preserve correct existing behavior unless the task explicitly replaces it.

## 2. Widget Ownership and Lifetime

A widget implementation belongs to a module.

A module runtime may expose widgets through the widget provider contract. The final widget catalog is assembled only from modules enabled for the current profile/site runtime.

Therefore:

- an installed package does not automatically make its widgets available to every site;
- only widgets from modules enabled by the site's profile are available;
- widget implementations may capture module-runtime dependencies;
- widget rendering is request-scoped, but widget definitions/implementations belong to the assembled site/module runtime;
- never use a mutable global widget registry or global active-site state.

## 3. Widget Definition, Binding, and Rendered Output Are Different Concepts

Keep these concepts separate.

### Definition

Describes a widget type:

- code;
- owning module;
- human label;
- description;
- configurable fields;
- optional editor tab metadata;
- optional summary-field metadata.

The definition is not a concrete widget placed on a resource.

### Binding

Describes one concrete resource widget placement:

- stable binding ID;
- widget code;
- area;
- position;
- presentation settings;
- widget params.

The same widget code may be added multiple times to the same resource. Therefore widget code cannot be used as the identity of a resource widget binding.

### Rendered widget

Represents final API output after template composition, resource insertion, widget instantiation, and rendering.

Do not collapse definition, binding, and rendered output into one struct.

## 4. Widget Areas

The MVP has two widget areas:

```text
body
sidebar
```

Model them as area codes, not as two widget types.

A widget remains the same widget when moved from body to sidebar. Moving it changes placement, not widget identity or implementation.

Use a backend-neutral concept such as:

```go
type AreaCode string

const (
    AreaBody    AreaCode = "body"
    AreaSidebar AreaCode = "sidebar"
)
```

Do not create separate `BodyWidget` and `SidebarWidget` abstractions.

## 5. Template Composition

A template may contain:

- static widgets defined by the template;
- an explicit resource-widget insertion slot;
- more static widgets after the slot.

Example conceptually:

```text
Template body:
    static content widget
    RESOURCE_WIDGETS_SLOT
    static gallery widget

Template sidebar:
    static navigation widget
    RESOURCE_WIDGETS_SLOT
```

During final composition, the resource-widget slot is replaced with the resource widget bindings assigned to that area.

The result preserves template order.

Do not append all resource widgets blindly after all template widgets.

For MVP, keep at most one resource-widget slot per area unless the architecture is explicitly extended to support named multiple slots. If multiple slots are introduced later, bindings will need an explicit slot identity.

## 6. Resource Widget Tab Availability

The admin resource editor must show the `Widgets` tab only when the selected resource template contains at least one resource-widget slot.

The backend metadata API must provide enough template metadata for the frontend to determine:

- whether resource widgets are supported;
- which areas accept resource widgets.

Do not infer this in the frontend from hard-coded template names.

## 7. Presentation Is Not Widget Params

Widget-specific business configuration belongs to `params` and is validated against widget fields.

Common visual/layout settings do not belong to widget params.

Keep presentation data separate, including at minimum:

- view;
- columns;
- margin top;
- margin bottom;
- enabled state.

Use backend-neutral names.

Prefer:

```text
columns = 1..12
margin_top = 0..3
margin_bottom = 0..3
```

Do not name backend fields after Bootstrap/Tailwind classes such as `bootstrap_width`, `col-12`, `mt-2`, or `mb-3`.

Frontend technology may change independently from backend storage/API contracts.

## 8. Default Widget View Is Implicit

Every widget always has a built-in default visual view.

`default` must not need to be declared by a module, project, profile, or site.

Rules:

- project/profile declarations contain only additional custom views;
- the admin UI always presents `Default` automatically;
- an empty/internal default view value may be stored as empty string or null according to repository conventions;
- the public API should normalize the default view to the string `default` so frontend consumers do not need backend-specific fallback rules;
- project-defined views must be validated against the widget they belong to.

Do not require declarations such as a manually registered `default` view.

## 9. Project-Level Widget Views

`backend/internal/` may declaratively add visual view codes for widgets.

A view declaration is only backend metadata telling the frontend which visual template/variant to use. The backend does not render the frontend template.

A view declaration should contain only unique project information such as:

- widget code;
- view code;
- human label where useful.

Reusable widget/profile machinery must own:

- collection;
- validation;
- merging;
- duplicate detection;
- runtime lookup;
- metadata exposure.

Follow extension precedence when overrides are supported:

```text
core < package < project < site
```

Do not put generic registration loops into `backend/internal/`.

## 10. Widget Module Context and Dependencies

Widgets may need module infrastructure and services such as:

- semantic cache stores;
- filesystems/storage;
- logger;
- event bus;
- repositories;
- module services.

These dependencies should be resolved while building the module runtime and captured by the widget implementation through constructors/composition.

Conceptually:

```text
ModuleContext
    -> ModuleRuntime
        -> Widget implementation
            cache / filesystem / logger / repositories / services
```

Do not pass the complete kernel `ModuleContext` into every `Render` call as a service locator.

The module build step already knows the module dependencies. Capture only the dependencies the widget needs.

This keeps dependencies explicit and prevents unrestricted runtime service lookup.

## 11. Render Context

Widget render input should contain request/resource context, not infrastructure wiring.

At minimum widgets may need neutral snapshots for:

- current site;
- current resource.

These snapshots must live in a package that does not create dependency cycles and must not expose persistence implementation structs directly.

The resource snapshot should be extended only with data actually needed by widgets.

Do not pass repository models directly just because they are convenient.

Use `context.Context` for request cancellation/deadlines and request-scoped context propagation.

## 12. Widget Codes and Module Ownership

Widget declaration codes are module-local before compilation and profile catalog codes are qualified globally according to the existing catalog convention.

The compiled widget definition/catalog must retain explicit owning-module metadata.

Do not recover the owner later by parsing the qualified widget code string.

The admin API needs explicit module ownership to group available widgets by module.

## 13. Module Human Metadata

The widget picker should group widgets by human-readable module names.

Do not overload the minimal required `kernel.Module` interface solely for admin presentation if module metadata is optional.

Prefer a small optional descriptor/provider contract for module label/description if the repository does not already provide equivalent metadata.

Widget-producing modules should expose enough metadata for a useful admin picker.

## 14. Widget Field Schema and Editor Tabs

`field.Definition` remains the source of truth for widget parameter type, validation, defaults, and options.

Editor tabs are presentation metadata only.

A widget may optionally declare editor tabs containing references to field codes. For example:

```text
Content
Design
Filters
```

Do not duplicate field definitions inside tabs.

Validation rules should catch:

- duplicate tab codes;
- unknown field references;
- the same field assigned inconsistently when duplicates are not intended.

If no custom editor tabs are declared, all widget-specific fields may be displayed in one default widget-settings tab.

## 15. Widget Summary Metadata

Widget cards in the resource editor may show selected short parameter values.

The widget definition may declare `summary fields` as field codes.

The widget author controls which params are meaningful in the collapsed card.

Do not let the frontend guess arbitrary fields from map iteration order.

## 16. Resource Widget Persistence

A resource widget binding needs a stable database identity independent from its position.

Persistence must support at least:

```text
id
resource_id
widget_code
area
position
view
columns
margin_top
margin_bottom
enabled
params
```

Recommended invariants:

- stable primary key `id`;
- resource FK with cascade delete;
- valid area;
- non-negative position;
- columns in 1..12;
- margins in 0..3;
- params is a JSON object;
- deterministic ordering within each area;
- unique `(resource_id, area, position)` or an equivalent repository-level invariant.

Do not use `(resource_id, position)` as the only identity because moving/reordering widgets would change their identity and positions are independent between body/sidebar.

Reordering multiple widgets must be transactional.

## 17. Resource Widget Mutations

Prefer widget-specific resource endpoints instead of repeatedly replacing the entire resource document for every drag/edit operation.

Conceptually:

```text
POST   /resources/{resource}/widgets
PATCH  /resources/{resource}/widgets/{widgetID}
DELETE /resources/{resource}/widgets/{widgetID}
PUT    /resources/{resource}/widgets/order
```

Exact route placement should follow the existing admin API conventions.

For MVP, reuse the existing resource update/edit permission for widget mutations unless a task explicitly introduces finer permissions.

Every mutation must still validate:

- site/resource scope;
- widget availability in the current profile runtime;
- template resource-widget slot availability;
- area allowed by the selected template;
- widget params;
- presentation ranges;
- selected custom view.

## 18. Widget Reordering

The admin UI must support:

- ordering widgets inside body;
- ordering widgets inside sidebar;
- moving a widget from body to sidebar and back.

A reorder request should send stable widget IDs plus final area/position rather than relying on widget codes.

Example conceptual payload:

```json
[
  {"id": 41, "area": "body", "position": 0},
  {"id": 17, "area": "body", "position": 1},
  {"id": 22, "area": "sidebar", "position": 0}
]
```

Backend applies and validates the final ordering transactionally.

## 19. Admin Widget Editor

The resource editor widget tab should be implemented as reusable widget-specific components rather than adding all logic directly to the already-large resource edit view.

Use Element Plus for UI controls and the existing dynamic-field system for widget fields where possible.

The widget tab contains two sections:

```text
Body
    + Add widget
    draggable widget cards

Sidebar
    + Add widget
    draggable widget cards
```

### Widget picker

The add-widget dialog should:

- show only widgets available in the current profile/site runtime;
- group widgets by module;
- hide modules that expose no available widgets;
- provide search;
- show widget label and description;
- allow selection before opening settings.

### Widget settings

Keep common settings separate from widget-specific params.

Common settings include:

- view select (`Default` always implicit plus project/profile views);
- width/columns 1..12;
- margin top 0..3;
- margin bottom 0..3;
- enabled state.

Use Element Plus ready controls. A 1..12 slider with stops is suitable for columns. For the small discrete margin range, use an appropriate discrete Element Plus control rather than encoding CSS classes.

Widget-specific fields use the current dynamic field components and optional editor-tab metadata.

### Widget card

Collapsed cards should show:

- widget label;
- description where useful;
- selected summary params;
- current presentation summary where useful;
- edit/delete/expand controls;
- drag handle.

Do not use widget code as the only user-facing title.

## 20. Public API Shape

The public API must return body and sidebar widgets separately.

Conceptually:

```json
{
  "widgets": {
    "body": [],
    "sidebar": []
  }
}
```

Do not return a single mixed array that requires frontend filtering by area.

Each rendered widget should contain enough presentation metadata for the frontend, including:

- code;
- view, normalized to `default` when no custom view is selected;
- columns;
- margins;
- rendered data;
- stable frontend key/identity where needed.

Resource binding IDs may be used for dynamic resource widget identity. Static template widgets need their own deterministic template item key if duplicate widget codes can occur.

Do not assume widget code is unique within a rendered page.

## 21. Rendering and Error Isolation

Final rendering pipeline is conceptually:

```text
selected template
    + resource widget bindings
    -> compose body/sidebar placements
    -> resolve widget runtime from current SiteRuntime/ProfileRuntime catalog
    -> validate params / create instance
    -> render with site/resource request context
    -> public body/sidebar response
```

Preserve per-widget error isolation where already supported.

One broken widget should not automatically make the whole page unavailable unless the task explicitly requires strict failure behavior.

Errors must be logged with useful widget/resource/site context without exposing internal details in the public API.

Disabled resource widgets remain stored/editable in admin but are omitted from public rendering.

## 22. Core Content Widget

The core module should provide a minimal content widget used as the simplest reference implementation.

Requirements:

- local widget code: `content` unless current conventions require another local code;
- final catalog code follows existing module qualification rules;
- human label and description are required;
- no configurable fields;
- no params are required;
- no custom view declaration is required because default is implicit;
- render result exposes the current resource content;
- it obtains resource data from render input rather than querying persistence again;
- it must be provided by the core module runtime through the normal widget provider mechanism;
- it must work both as a static template widget and, if allowed by product rules, as a resource-added widget without special-case rendering code.

This widget is the canonical smoke-test for the widget pipeline.

## 23. Template Validation Timing

Current widget implementations are contributed by assembled module runtimes, while template declarations may exist earlier as profile blueprint data.

Do not introduce an invalid dependency or global catalog merely to validate template widgets too early.

If static template widget references cannot be fully validated during profile blueprint compilation, perform final widget-reference validation when the site/profile runtime widget catalog is available.

Keep syntactic template validation earlier where possible and runtime-resolution validation at the correct assembly stage.

## 24. Declarative Project Rule for Widgets

Project-level widget customization under `backend/internal/` should contain only declarations unique to the project, such as:

- template widget placements;
- resource-widget slots;
- additional view codes/labels;
- project-owned widget implementations when genuinely project-specific.

Project code must not manually:

- collect all module widgets;
- qualify codes;
- build field schemas for every widget;
- merge profile catalogs;
- validate generic widget ordering;
- render widgets;
- mount generic widget admin routes;
- duplicate the same registration lifecycle for every widget/view.

Reusable widget/runtime/admin machinery owns those mechanics.

## 25. Avoid Premature Features

Do not add without a concrete requirement:

- arbitrary nested widget trees;
- recursive containers/page-builder blocks;
- multiple named resource slots per area;
- global cross-site widget registry;
- server-side frontend template rendering;
- generic DI/service locator access from widget render;
- widget output caching at framework level;
- widget-specific permission system separate from resource editing;
- compatibility layers for abandoned widget shapes.

Keep the MVP focused on body/sidebar composition and predictable extension points.

## 26. Tests

For widget changes, add focused tests for the affected contracts.

Important cases include:

- widget catalog includes only current profile modules;
- global widget codes are deterministic and duplicate-safe;
- compiled definition retains owning module metadata;
- params validation works;
- zero-field widget accepts empty params;
- module-runtime dependencies remain available to widget implementation;
- template composition inserts resource widgets exactly at the slot;
- body/sidebar ordering is independent;
- cross-area moves produce correct final ordering;
- disabled widgets are not publicly rendered;
- default view is implicit and public API normalizes it to `default`;
- invalid custom view is rejected;
- invalid area/presentation values are rejected;
- resource widget binding identity survives reorder/move;
- one failed widget does not break unrelated rendered widgets where error isolation is expected;
- core content widget returns resource content.

Add repository/integration tests for persistence and admin mutation behavior when those layers change.

## 27. Validation

For Go changes, run from `backend/` when practical:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

For admin frontend changes, use the repository's existing package manager/scripts and run the relevant typecheck/lint/build commands that are actually configured in `frontend-admin/package.json`.

Do not claim successful validation unless the commands were actually run successfully.

## 28. Final Review Checklist

Before finishing widget work, verify:

### Runtime

- Are widgets still site/profile-scoped?
- Are only enabled profile modules contributing widgets?
- Did any mutable global registry appear?
- Are module infrastructure dependencies captured during module runtime build rather than fetched through a render-time service locator?

### Model

- Are Definition, Binding, and RenderedWidget separate?
- Does every resource binding have a stable ID?
- Are body/sidebar areas explicit?
- Are presentation settings separate from params?
- Is default view implicit?

### Template composition

- Are resource widgets inserted at the explicit template slot rather than appended blindly?
- Does final body/sidebar ordering remain deterministic?
- Is the Widgets admin tab driven by template slot metadata?

### Admin/API

- Does the picker show only current profile widgets and group them by owning module?
- Are human labels used instead of raw codes where appropriate?
- Can cards reorder inside and across body/sidebar?
- Are public body/sidebar arrays separate?
- Is `default` normalized for frontend output?

### Architecture

- Is project/internal still declarative?
- Did kernel avoid concrete cache/storage/database technology knowledge?
- Did widget rendering avoid persistence model coupling?
- Did implementation avoid unrelated refactors and compatibility code?
