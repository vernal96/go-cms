---
name: go-cms-widgets
description: Use for GO CMS widget architecture and implementation work involving widget definitions and references, module widget providers, template widget layouts, resource widget bindings, widget rendering, body/sidebar areas, widget views, admin widget editing, drag-and-drop ordering, or public widget API composition. Preserves site-scoped runtimes and typed declarative project composition.
---

# GO CMS Widgets

## Purpose

Use this skill for any task that adds, changes, renders, stores, configures, or edits widgets in GO CMS.

Always follow repository root `AGENTS.md` and `.codex/skills/go-cms-development/SKILL.md` first. This skill adds widget-specific architectural rules.

The widget system must remain:

- module-owned;
- site/profile-scoped at runtime;
- declarative and strongly typed in `backend/internal/`;
- independent from concrete infrastructure technologies;
- composable from template widgets and resource widgets;
- explicit about presentation settings versus widget business parameters;
- free from manually written runtime/DB identity strings in project-facing Go declarations.

## 1. Inspect Before Editing

Before changing widget code, inspect the current implementation of:

- `backend/kernel/modules/core/widget/`;
- `backend/kernel/modules/core/widgets/`;
- `backend/kernel/modules/core/template/`;
- `backend/kernel/modules/core/resource/`;
- profile/runtime assembly in `backend/kernel/runtime.go`;
- owning module runtimes and `widget.Provider` implementations;
- resource persistence and migrations;
- admin resource metadata/widget APIs;
- `frontend-admin/src/views/ResourceEditView.vue` and resource-widget components;
- the public resource/page renderer.

Do not redesign from memory when the repository already contains part of the widget pipeline.
Preserve correct persistence, admin, rendering, and runtime behavior when refining declaration APIs.

## 2. Core Rule: Typed Project Declarations, Codes Inside the Framework

Project/profile/template Go declarations should express objects and relationships, not framework discriminators or compiled identity strings.

Bad project-facing declaration:

```go
LayoutItem{
    Kind:         template.ItemWidget,
    Key:          "content",
    Widget:       "core_content",
    Presentation: widget.DefaultPresentation(),
}
```

Preferred direction:

```go
template.Widget{
    Widget: corewidgets.Content,
}
```

With a custom view and params:

```go
template.Widget{
    Widget: projectwidgets.Gallery,
    View:   widgetviews.GallerySlider,
    Params: map[string]any{
        "limit": 12,
    },
}
```

Codes are still valid and necessary inside:

- widget/module registries;
- compiled runtime catalogs;
- persistence;
- HTTP APIs;
- logs/errors;
- metadata sent to frontend clients.

But `backend/internal/` should not manually reproduce globally-qualified codes such as `core_content` when it can reference a typed exported widget object.

The framework/compiler is responsible for translating a widget reference into its compiled/global code.

## 3. Widget Reference vs Widget Runtime

Do not place a site-scoped runtime widget instance directly into a reusable profile/template declaration.

Separate:

```text
Widget reference/descriptor
    reusable, immutable, safe for profile/template declarations

Widget runtime implementation
    assembled through ModuleRuntime for a concrete site runtime
    may capture cache/storage/logger/repositories/services
```

A module/package should export stable widget references for project declarations, for example conceptually:

```go
corewidgets.Content
projectwidgets.Gallery
```

The exact type name (`Ref`, `Reference`, `Descriptor`, etc.) may follow repository style, but it must be a typed immutable value/object rather than a raw global-code string.

The actual module widget implementation must be associated with the same reference during widget catalog compilation. Avoid duplicating identity in several unrelated declarations when a single reference can be the source of truth.

## 4. Widget Ownership and Lifetime

A widget implementation belongs to a module.

A module runtime may expose widgets through the widget provider contract. The final widget catalog is assembled only from modules enabled for the current profile/site runtime.

Therefore:

- installed packages do not automatically make widgets available to every site;
- only widgets from modules enabled by the site's profile are available;
- widget implementations may capture module-runtime dependencies;
- widget rendering is request-scoped;
- final widget runtime availability is site/profile scoped;
- never use a mutable global widget registry or active-site singleton.

## 5. Definition, Reference, Binding, Placement and Rendered Output Are Different

Keep these concepts separate.

### Widget definition

Describes a widget type:

- typed identity/reference;
- owning module;
- human label;
- description;
- configurable fields;
- optional editor tabs;
- optional summary fields.

### Widget reference

An immutable Go value used by templates/profiles/project code to refer to the widget without manually writing its global runtime code.

### Resource binding

A persisted concrete resource widget instance:

- stable binding ID;
- compiled widget code for persistence/runtime lookup;
- area;
- position;
- presentation;
- params.

The same widget type may be added multiple times to one resource, so widget code is not binding identity.

### Placement

The composed result of static template widgets and resource widget bindings before rendering.

### Rendered widget

The public API result after resolving the site/profile widget runtime and calling the widget implementation.

Do not collapse these concepts into one struct.

## 6. Widget Areas

The MVP has exactly two areas:

```text
body
sidebar
```

They are layout areas, not widget types.

A widget remains the same widget when moved from body to sidebar.

Use backend-neutral area codes internally/persistently, but project template declarations should normally express area through the typed `Layout.Body` and `Layout.Sidebar` collections rather than manually adding `Area: "body"` to every static item.

## 7. Template Items Must Be Typed, Not `Kind`-Driven

Do not require project code to set a discriminator such as:

```go
Kind: template.ItemWidget
Kind: template.ItemResourceSlot
```

This is internal compiler information and should be represented by Go types/interfaces.

Preferred conceptual model:

```go
type Item interface {
    isTemplateItem()
}

type Widget struct {
    Widget widget.Ref
    View   widget.ViewRef
    Params map[string]any
    // optional presentation overrides
}

type ResourceWidgets struct{}
```

Then a template can read naturally:

```go
Layout: template.Layout{
    Body: []template.Item{
        template.Widget{Widget: corewidgets.Content},
        template.ResourceWidgets{},
    },
    Sidebar: []template.Item{
        template.ResourceWidgets{},
    },
}
```

The exact implementation may use interfaces, sealed marker methods, or another type-safe Go pattern, but callers must not need a `Kind` field.

## 8. Static Template Keys Are Framework Concerns by Default

Do not require every static template widget declaration to manually provide a technical `Key` solely for rendering identity.

The template compiler can generate a deterministic identity from stable template structure, for example conceptually:

```text
template:<template-code>:<area>:<index>
```

A user-defined key may be added later only if there is a real semantic use case for addressable static template items.

Do not force boilerplate before that use case exists.

## 9. Template Composition

A template may contain:

- static typed widget declarations;
- an explicit typed resource-widget insertion slot;
- more static widgets after the slot.

Example:

```text
Body:
    core content widget
    RESOURCE_WIDGETS_SLOT
    gallery widget

Sidebar:
    navigation widget
    RESOURCE_WIDGETS_SLOT
```

During composition, the resource slot is replaced by resource widget bindings assigned to that area.

Preserve order. Do not blindly append resource widgets after template widgets.

For MVP, allow at most one resource-widget slot per area.

## 10. Resource Widget Tab Availability

Show the admin `Widgets` tab only when the selected template contains at least one resource-widget slot.

Backend metadata must expose:

- whether resource widgets are supported;
- which areas accept resource widgets.

Do not hard-code this in the frontend by template name.

## 11. Views Are Typed Profile Declarations

A custom visual view is a real project/profile declaration and should be referenceable as a Go object.

Bad:

```go
WidgetViews: []widget.ViewDeclaration{
    {Widget: "core_content", Code: "article", Label: "Статья"},
}
```

Preferred direction:

```go
var ContentArticle = widget.NewView(
    corewidgets.Content,
    "article",
    "Статья",
)
```

Profile:

```go
WidgetViews: []widget.View{
    widgetviews.ContentArticle,
}
```

Template:

```go
template.Widget{
    Widget: corewidgets.Content,
    View:   widgetviews.ContentArticle,
}
```

Exact constructors/type names may differ, but the important rules are:

- a custom view object carries/knows which widget it belongs to;
- templates reference the view object directly;
- the compiler validates that the view belongs to the selected widget;
- profile declarations contain view objects, not repeated widget-code/view-code string joins;
- runtime/DB/API may still serialize the compiled view code.

Use a sibling package such as `internal/profiles/<profile>/widgetviews` when needed to avoid Go import cycles between `profile.go` and template packages.

## 12. Default View Is Fully Implicit

Every widget always has a built-in default visual view.

`default` must not be explicitly declared in:

- module definitions;
- profile views;
- template declarations;
- project declarations.

A template using the default view should simply omit `View`:

```go
template.Widget{
    Widget: corewidgets.Content,
}
```

The admin UI adds `Default` automatically.

The public API normalizes the default view to:

```json
"view": "default"
```

No project code should need `widget.DefaultPresentation()` merely to get default values.

## 13. Presentation Is Not Widget Params

Widget-specific business configuration belongs to `Params` and is validated through widget fields.

Common visual/layout settings are separate.

Resource widget presentation supports at least:

- view;
- columns;
- margin top;
- margin bottom;
- enabled.

Use backend-neutral names and ranges:

```text
columns: 1..12
margin_top: 0..3
margin_bottom: 0..3
```

Do not use Bootstrap/Tailwind-specific backend names.

## 14. Static Template Presentation Must Not Require Default Boilerplate

A static template widget should not need:

```go
Presentation: widget.DefaultPresentation()
```

The template/compiler layer must normalize omitted presentation values.

Conceptually, omission means:

```text
view = default
columns = 12
margin top = 0
margin bottom = 0
enabled = true
```

If Go zero values make a field ambiguous (especially `bool`), use a template-specific override type, pointers/options, or another clear representation rather than forcing callers to fill all defaults manually.

Do not over-generalize resource-binding persistence structs as project template declaration structs when their zero-value semantics differ.

## 15. Widget Module Context and Dependencies

Widgets may need module infrastructure/services such as:

- semantic cache stores;
- filesystems/storage;
- logger;
- event bus;
- repositories;
- module services.

Resolve those dependencies while building the module runtime and capture only the dependencies needed by each widget implementation.

Conceptually:

```text
ModuleContext
    -> ModuleRuntime
        -> Widget implementation
            cache / filesystem / logger / repositories / services
```

Do not pass the entire kernel `ModuleContext` into each `Render()` call as an unrestricted service locator.

## 16. Render Context

`Render()` receives request/resource context, not infrastructure wiring.

Use `context.Context` plus neutral snapshots for the current site and resource.

Do not expose persistence implementation structs directly.
Extend snapshots only when widgets actually need additional resource/site data.

## 17. Widget Codes and Module Ownership

Codes remain internal identity primitives.

Module-local widget identity is compiled/qualified by reusable widget catalog machinery.

The compiled definition/catalog must retain explicit owning-module metadata.
Do not recover ownership by parsing the global widget code.

Project templates should use exported typed widget references; persistence/API may use compiled codes.

## 18. Module Human Metadata

Admin widget picker grouping requires human-readable module metadata.

Prefer a small optional descriptor/provider contract if the minimal module interface does not already expose it.

Do not make project code repeat module labels on every widget.

## 19. Widget Fields, Editor Tabs and Summary Metadata

`field.Definition` remains the source of truth for widget params.

Editor tabs reference field codes and are editor-layout metadata only.
Validate:

- duplicate tabs;
- unknown field references;
- invalid duplicate assignments.

Summary fields tell widget cards which params to show in collapsed form.
Do not make the frontend guess from map iteration.

## 20. Resource Widget Persistence

Persist stable resource widget bindings with at least:

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

- stable primary key ID;
- resource FK with cascade delete;
- valid area;
- position >= 0;
- columns 1..12;
- margins 0..3;
- params is a JSON object;
- deterministic independent ordering per area;
- unique `(resource_id, area, position)` or equivalent repository invariant.

Reordering/moving widgets must be transactional.

## 21. Admin Mutations and Permissions

Use stable binding IDs for add/edit/delete/reorder operations.

Prefer dedicated resource-widget mutation endpoints rather than rewriting the whole resource during drag operations.

Reuse the existing resource-update permission model unless a separate permission model is explicitly requested.

Validate against the final site/profile runtime:

- widget availability;
- selected template resource slots/areas;
- params;
- view belongs to widget and is available in the profile;
- presentation ranges.

## 22. Admin Metadata

Expose enough metadata for the editor to render widgets without hard-coded module/widget knowledge:

For templates:

- resource-widget support;
- supported areas.

For widgets:

- compiled public code;
- module code/label/description;
- label/description;
- fields;
- editor tabs;
- summary fields;
- available custom views.

Only expose widgets available in the current site/profile runtime.

## 23. Admin UI

The resource widget editor should be extracted from the main resource view into dedicated components.

The Widgets tab contains body/sidebar sections with add buttons and draggable cards.

Support:

- reorder within area;
- move between body/sidebar;
- add/edit/delete;
- search/group picker by module;
- dynamic widget fields;
- common presentation settings;
- implicit `Default` view option.

Use Element Plus for UI controls. Do not introduce another UI framework.

## 24. Public API

Return body and sidebar separately:

```json
{
  "widgets": {
    "body": [],
    "sidebar": []
  }
}
```

Rendered items should include frontend-useful identity/presentation metadata such as:

- key;
- compiled widget code;
- normalized view;
- columns;
- margins;
- data/error.

Resource binding IDs may contribute to public keys.
Static template widget keys should be generated deterministically by the compiler/composer unless semantic custom keys become a real requirement.

Do not leak raw widget params into public output unless the widget explicitly renders them as data.

## 25. Rendering Pipeline

Preserve this shape:

```text
resolve SiteRuntime
    -> resolve resource/template
    -> compose typed template items + resource bindings
    -> body/sidebar placements
    -> resolve widget runtime from site/profile catalog
    -> validate/normalize params and presentation
    -> instantiate widget
    -> Render(ctx, SiteSnapshot + ResourceSnapshot)
    -> body/sidebar public API
```

Preserve per-widget failure isolation where currently implemented.
A broken widget should not unnecessarily fail the entire page.

Disabled resource widgets stay persisted/editable but are omitted from public rendering.

## 26. Core Content Widget

The core module's content widget is the canonical minimal widget.

Requirements:

- module-local identity is `content`;
- project declarations access it through an exported typed reference such as `corewidgets.Content`;
- no widget-specific fields;
- no required params;
- no explicit default view declaration;
- no repository lookup during render;
- output comes from `RenderInput.Resource.Content`;
- it is provided by the normal core module runtime/provider path;
- it works as a static template widget without special HTTP handling.

Do not create a duplicate content widget.

## 27. Declarative Internal Rule

`backend/internal/` may declare:

- templates;
- typed widget references;
- project-owned widgets;
- custom view objects;
- widget params/presentation overrides;
- profile view lists.

It must not manually:

- build widget catalogs;
- qualify global widget codes;
- set framework item discriminators;
- generate technical placement keys;
- register implicit default views;
- collect module widget providers;
- render widgets;
- merge template/resource widgets;
- duplicate generic validation/registration loops.

If adding the next widget/template/view requires copying framework mechanics, improve the reusable declaration/compiler API instead.

## 28. Do Not Overengineer

Do not introduce unless explicitly requested:

- recursive/nested widget trees;
- arbitrary page-builder containers;
- multiple named resource slots;
- global widget registries;
- server-side frontend template rendering;
- DI containers;
- unrestricted render-time ModuleContext access;
- generic rendered-widget caching;
- separate widget permissions;
- legacy compatibility wrappers for superseded project declaration APIs.

Implement the intended API directly.

## 29. Tests

Cover at least:

- exported widget reference resolves to the correct module/runtime widget;
- project templates compile without manually specified global widget code;
- typed template widget/resource-slot items compose correctly;
- no caller-supplied `Kind` is required;
- generated static placement keys are deterministic;
- default view is implicit;
- custom view objects are registered by profile and validated against their widget;
- template referencing a view belonging to another widget is rejected;
- static presentation omission normalizes to defaults;
- zero-field content widget accepts empty params and renders resource content;
- profile runtime exposes widgets only from enabled modules;
- body/sidebar composition and resource slots work independently;
- duplicate resource slots are rejected;
- stable resource binding identity survives reorder/move;
- disabled resource widget omission;
- per-widget render failure isolation;
- persistence/admin/public API behavior remains correct.

## 30. Validation

For Go changes, run from `backend/` when practical:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

For frontend changes, inspect `frontend-admin/package.json` and run the configured typecheck/lint/test/build commands that apply.

Never claim a command succeeded unless it actually ran successfully.

## 31. Final Review Checklist

Before finishing, verify:

- Are project template declarations free from raw global widget-code strings?
- Are template items represented by types rather than manually assigned `Kind` values?
- Are technical static keys generated by reusable machinery?
- Can a template reference an exported widget object directly?
- Can it reference a profile-declared custom view object directly?
- Is `default` still implicit everywhere?
- Are runtime/DB/API codes still available where identity/serialization requires them?
- Are widget runtime dependencies still resolved through the owning ModuleRuntime?
- Is site/profile-scoped widget availability preserved?
- Did the change preserve existing admin/persistence/rendering functionality?
- Did project/internal become simpler rather than more framework-aware?
