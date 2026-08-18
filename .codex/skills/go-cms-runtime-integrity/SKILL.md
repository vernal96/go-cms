---
name: go-cms-runtime-integrity
description: Use for GO CMS changes involving SiteRuntime, ProfileBlueprint, ModuleContext/RuntimeScope, site reload/publication, runtime-scoped caches, runtime HTTP compilation, or cache invalidation across admin/service writes.
---

# GO CMS Runtime Integrity

Use this skill when changing site/profile runtime construction, runtime-scoped infrastructure, site reload/update behavior, or cached domain reads/writes.

Follow root `AGENTS.md`, `backend/AGENTS.md`, and the general `go-cms-development` skill.

## 1. Preserve the Runtime Model

The intended model is:

```text
Profile declaration
    -> compiled immutable ProfileBlueprint
        -> per-site final runtime
            -> per-site RuntimeRegistry
            -> per-site ModuleRuntime instances
            -> per-site runtime-scoped bindings
```

Do not move module runtime construction back to profile-global shared instances.

Site runtimes are built on boot/create/update/reload and stored in process memory. Do not build them per request and do not use an external cache as runtime storage.

## 2. RuntimeScope Contract

When a module requires site-specific information during `Build`, extend a generic kernel-owned runtime scope rather than coupling kernel to `core/site`.

A runtime scope may expose stable runtime identity/configuration needed during module construction, for example:

```text
site ID
site/profile code
locale/settings snapshot when genuinely required
```

Rules:

- expose only data needed to build runtime state;
- defensively copy maps/slices;
- never expose request actor/request context as persistent runtime state;
- never make generic kernel import core site entities merely to provide scope;
- keep module dependency access separate from runtime scope.

## 3. Atomic Runtime Publication

Runtime construction/preparation must be transactional from the reader's point of view.

Required behavior:

```text
build complete candidate runtime set
    -> validate/prepare all candidates
        -> atomically publish
```

If any candidate fails, readers continue using the previous complete working runtime state.

Avoid mutating already-published runtime objects during preparation when a later preparation step can still fail.

## 4. Runtime-Associated HTTP State

HTTP handlers may be compiled once per site, but core site/domain packages should not own `net/http.Handler` directly.

Prefer one of:

- HTTP/server-owned immutable handler catalog keyed by stable site/runtime identity;
- a generic runtime attachment facility only if multiple transports genuinely need the same pattern.

Do not invent a generic attachment framework only to move one field.

When site runtime replacement occurs, handler state and runtime state must switch coherently. Do not publish a new runtime without its required compiled HTTP artifact.

## 5. Cache Coherence Workflow

Whenever a runtime uses a cached repository or cache decorator, identify all writers to the same domain data.

Never accept this split without an invalidation bridge:

```text
runtime read  -> cached repository
admin write   -> base repository
```

Preferred design property:

- domain mutation passes through the same cache-aware persistence/service boundary, or
- a centralized domain invalidation policy is invoked by every mutation path.

Invalidation must be domain-driven, not HTTP-driven.

### Mandatory regression sequence

For each affected mutable cached entity:

```text
1. create/load entity
2. read via site runtime to populate cache
3. mutate via real admin/application service path
4. read again via site runtime
5. assert updated value is returned
```

Also test two sites sharing the same physical cache store when site namespace isolation matters.

## 6. Cache Namespace Rules

Semantic alias selection remains module-facing; physical store selection remains project-facing.

Site/profile/module namespace generation is runtime machinery.

When explicit `Binding.Namespace` is provided:

- treat it as an intentional project override;
- verify whether it should be shared or site-isolated;
- do not silently create cross-site cache collisions;
- add validation/documentation if explicit sharing has non-obvious semantics.

## 7. Module Build Order

`core` is mandatory and must be first.

Other dependencies are explicit through `DependencyProvider` or its successor.

Runtime construction must fail clearly for:

- missing dependencies;
- undeclared dependency access;
- self dependency;
- duplicate dependency declaration;
- invalid ordering.

Do not use implicit dependency discovery from concrete runtime types.

## 8. Tests Before Completion

For runtime/cache refactors add focused tests before relying on broad compilation.

At minimum consider:

- per-site module runtime identity;
- runtime scope propagation into ModuleContext;
- defensive copy/immutability of scope settings;
- atomic failure behavior on reload/update/preparation;
- cache read/write coherence;
- site cache isolation;
- HTTP artifact publication with runtime replacement;
- core-first validation.

Then run from `backend/`:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

Do not report success without executing the commands.