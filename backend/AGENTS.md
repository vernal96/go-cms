# GO CMS Backend Rules

These rules extend the repository root `AGENTS.md` for all Go backend work.

## Profile Invariants

- `core` and `admin` are mandatory modules.
- `core` must be the first module in every profile.
- Module dependency order must be explicit and deterministic. A dependency must be declared before the dependent module unless the runtime system later gains an explicit dependency graph/topological build mechanism.
- Profile files remain declarative. Validation, dependency resolution, runtime assembly, ordering, and lifecycle belong to reusable runtime/application code.

## Profile Blueprint and Site Runtime

Keep a strict distinction between reusable profile definitions and final site runtime state.

```text
Profile declaration
    -> immutable ProfileBlueprint
        -> SiteRuntime A
        -> SiteRuntime B
```

- `ProfileBlueprint` may contain immutable compiled definitions such as schemas, templates, static registry definitions, and module declarations.
- Every site must receive its own final runtime registry and module runtime instances.
- Site runtimes are built during application initialization, site creation/update, or explicit reload, then kept in process memory.
- Never rebuild a full site runtime on every HTTP request.
- Publish runtime catalog changes atomically: a failed build/preparation must not partially replace the currently usable runtime snapshot.

## Runtime Scope

Site-specific module construction must have access to generic site scope without making kernel depend on the `core/site` domain package.

- Represent site identity/configuration through kernel-owned generic runtime scope contracts.
- `ModuleContext` must expose the runtime scope needed by modules to build site-specific state.
- Do not solve this by importing `kernel/modules/core/site` into generic kernel runtime contracts.
- Do not store request-specific values in long-lived runtime scope.
- Keep scope data immutable or defensively copied when exposed to modules.

## Cache Coherence Is a Correctness Invariant

Cache invalidation is not an optional optimization detail. If a read path can return cached domain data, every write path that changes that data must preserve cache coherence.

Required invariant:

```text
cached read
    -> write through any supported API/service/admin path
        -> relevant cache invalidated or updated
            -> next read observes the new value
```

Rules:

- Do not create a cached repository/runtime used for reads while application/admin writes bypass the invalidation mechanism through an uncached base repository.
- A service must not accidentally have separate read and write persistence paths with incompatible cache behavior.
- Cache invalidation ownership should be centralized close to the persistence/cache decorator or a dedicated domain cache policy, not duplicated across HTTP/admin callers.
- Site-scoped namespaces must not make global/admin writes unable to invalidate affected site data.
- When data may affect multiple namespaces/sites, invalidation must address every affected scope deterministically.
- Do not fix coherence by clearing the entire cache unless the domain operation genuinely requires it.

For every cache-backed mutable entity, add at least one regression test with this sequence:

1. read and populate cache;
2. mutate through the real management/service path;
3. read again through the cached runtime path;
4. assert fresh data is returned.

Prefer tests that prove invalidation behavior rather than tests that only verify cache keys/namespaces.

## App and Module Ownership

`App` is the application composition root and lifecycle owner. It may assemble and expose subsystems, but module business composition belongs to modules.

- Keep domain CRUD methods on domain services, not as duplicated `App` forwarding APIs.
- `cms.core` owns construction of core domain services where practical.
- `admin` owns admin-specific management/runtime behavior.
- Generic kernel code should not learn module-specific domain behavior merely to assemble the application.

### Dependency Direction

Target dependency direction:

```text
project/internal
    -> application/CMS composition
        -> feature modules
            -> cms.core
                -> generic kernel contracts/mechanics
```

Treat any import from generic kernel packages into `kernel/modules/core/*` or `kernel/modules/admin/*` as architectural debt that requires justification.

Do not perform package moves only for cosmetic purity. When fixing this boundary, first extract generic contracts/mechanics so dependencies can point in the correct direction with minimal churn.

## HTTP and Transport Ownership

Domain runtime packages should not own transport-specific compiled state unless the type is intentionally a transport abstraction.

- `site.Runtime` should represent site runtime/domain state, not become an HTTP server container.
- Avoid storing `net/http.Handler` directly in core site/domain packages.
- HTTP compilation and handler ownership should live in the HTTP transport/server layer, or behind a genuinely generic runtime attachment mechanism that does not make domain packages import `net/http`.
- Site HTTP handlers may still be compiled once per site and kept in memory. The rule is about ownership/dependency direction, not about rebuilding handlers per request.
- Runtime/handler publication must be atomic: a preparation failure must leave the previous working routing state intact.

## Cache and Runtime Scope

Cache alias remains semantic and module-facing:

```text
repository
resources
templates
runtime
media
```

The project chooses the physical store.

Default runtime namespaces may include site/profile/module identity, but an explicit namespace is a project decision and must not accidentally remove required isolation. Validate intentional sharing when the same explicit namespace is reused across site runtimes.

## Extension Precedence

Target extension precedence remains:

```text
core < package < project < site
```

- Duplicate registration and intentional override are different operations.
- Continue rejecting accidental duplicates.
- Where replacement is a supported feature, require explicit override intent and deterministic precedence.
- Do not depend on map iteration or incidental registration order.
- Do not implement override support in registries that have no concrete replacement use case yet.

## Architecture Regression Tests

For cross-cutting runtime refactors, add focused tests for the architectural contract, not only compilation.

Important cases include:

- two sites with the same profile receive distinct final registries/module runtime instances;
- immutable profile blueprints may be shared safely;
- site runtime lookup remains in-memory and does not rebuild on request;
- declared module dependencies resolve and undeclared dependencies fail clearly;
- `core` is required and first; `admin` is required;
- cache aliases resolve only explicitly bound stores;
- site-scoped cache namespaces are isolated where required;
- writes invalidate cached reads regardless of whether mutation originates from admin or another service;
- failed site runtime rebuild/preparation does not replace a working runtime snapshot;
- HTTP compilation does not require core domain packages to own `net/http.Handler` state.

## Refactoring Discipline

When fixing architecture:

1. reproduce or test the actual problem first when possible;
2. fix the ownership/contract, not only the immediate call site;
3. preserve the declarative `internal` layer;
4. avoid introducing a general DI container or service locator;
5. avoid unrelated package moves and naming cleanup;
6. keep existing behavior unless the behavior itself violates the target architecture;
7. run `gofmt`, `go test ./...`, `go vet ./...`, and `go build ./...` for broad backend refactors.