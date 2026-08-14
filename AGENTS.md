# GO CMS Agent Instructions

## Start Here

- For architecture or cross-cutting backend work, inspect the current implementation before changing it.
- If `codebase-memory-mcp` is available, call `get_architecture` first, then verify important assumptions against the actual files you will modify.
- Treat repository code as the source of truth for current APIs and package layout.
- For substantial GO CMS backend work, follow `.codex/skills/go-cms-development/SKILL.md`.
- Do not add legacy compatibility checks, fallback paths, transitional APIs, or dead compatibility code unless explicitly requested. Implement the intended architecture directly.

## Repository Boundaries

- `backend/` is the only root for Go backend code and backend infrastructure files.
- `backend/internal/` is the project layer: project composition, profiles, project-specific declarations, customizations, and application bootstrap.
- Every frontend is an independent top-level application, such as `frontend-admin/`.
- Never place frontend source, Node tooling, or frontend build artifacts inside `backend/`.
- Frontends communicate with the backend only through public HTTP APIs. Do not import or share source code across backend/frontend application boundaries.
- Do not embed frontend applications into the Go binary or create compatibility paths between applications.

## Architectural Direction

GO CMS is a modular API-first CMS assembled from reusable Go packages.

Conceptually, dependencies point inward/downward:

```text
project/internal
      |
feature modules
      |
   cms.core
      |
    kernel
```

Higher layers may depend on lower layers. Lower layers must not depend on project-specific code or concrete project configuration.

### Kernel

Kernel owns generic contracts and mechanics: application lifecycle primitives, registries, runtime contracts/factories, infrastructure abstractions, managers, generic transport/runtime orchestration, and validation.

Kernel must not depend directly on PostgreSQL, Redis, S3, Kafka, RabbitMQ, local filesystem implementations, or other concrete infrastructure technologies.

Kernel must not become the owner of feature-specific CMS business logic merely because a feature is used by every project.

### Modules

- `cms.core` and `admin` are mandatory modules for every profile.
- Other modules are enabled explicitly by profiles.
- A module owns its domain behavior, module runtime, module-specific services, repository contracts, and persistence contracts.
- Cross-module dependencies must be explicit. Do not use a generic context as an unrestricted service locator.
- Modules must not rely on a global mutable active-site singleton.

### Infrastructure, connectors, adapters, repositories

Keep these responsibilities distinct:

```text
project selects concrete technology
        -> connector/client
        -> module database/storage adapter
        -> module repository implementation
        -> module service/runtime
```

- Connectors are low-level technology clients/pools.
- Module adapters translate a concrete connector into module-owned contracts.
- Repositories know module entities and persistence behavior.
- The project/bootstrap layer selects concrete connectors and adapters.
- Do not collapse connector, adapter, repository, and service into one abstraction.
- Concrete module adapters may know the concrete connector type. Generic kernel/module contracts must not.

### Cache and filesystem bindings

Modules refer to storage/cache by semantic purpose or alias, not technology name.

Prefer concepts such as:

```text
resources
repository
templates
runtime
media
```

Project configuration decides whether an alias maps to Redis, memory, file storage, S3-compatible storage, database storage, null storage, or another implementation.

Do not add one global `Cache` field that assumes one cache technology for the entire application.

## Profiles and Site Runtime

A site is not a folder. It is a persisted site record combined with a code-defined profile and its assembled runtime.

A profile is a declarative description of capabilities and configuration. Keep profile declarations close to the current `backend/internal/profiles/dev/profile.go` style: codes, modules, configs, params, templates, and bindings.

Site runtime rules:

- final runtime state is site-scoped;
- site runtimes are built during application initialization/reload and kept in process memory;
- do not rebuild a full SiteRuntime for every HTTP request;
- do not use Redis or another external cache as storage for runtime objects merely to avoid holding them in process memory;
- immutable profile-level definitions/blueprints may be shared between sites;
- site-specific mutable/final registries, runtime state, bindings, and overrides must have a clear site scope;
- runtime lookup on a request should be a cheap in-process lookup after site resolution.

## App

`App` is intentionally the composition root, lifecycle owner, and central access point to the assembled application.

It may own/expose infrastructure managers, services, profiles, site runtimes, console/lifecycle components, and other assembled subsystems.

It must not become the implementation of every domain service.

Prefer:

```go
app.Users().Create(...)
app.Services().Resources.Create(...)
```

instead of duplicating domain APIs as forwarding methods:

```go
app.CreateUser(...)
app.CreateResource(...)
```

Domain behavior belongs to domain services/modules. `App` assembles, owns, coordinates, and exposes them.

Large App implementation details should be split by responsibility inside the package when useful, without inventing unnecessary public abstractions or DI containers.

## Critical Rule: Declarative Project Layer

**Routine project code under `backend/internal/` must be maximally small and declarative.**

The project layer should primarily describe **what exists and how it is configured**. Reusable framework/module code must determine **how declarations are registered, validated, ordered, initialized, compiled, executed, reloaded, and shut down**.

This rule applies especially to:

- profiles;
- module inclusion and module configuration;
- connectors/adapters/bindings;
- migrations;
- seeds;
- routes and route groups;
- middleware contributions;
- permissions;
- templates and widgets;
- resource/field registrations;
- project extension registration.

### Required design test

When adding a routine element, ask:

> Does `internal` contain only the information unique to this element, or am I repeating framework mechanics that every similar element will need?

If every new route, migration, seed, profile, module, or other routine declaration requires copying the same registration/boot/execution logic, the abstraction is wrong.

Move repeated mechanics into the reusable owner — kernel, module package, registry, manager, compiler, factory, runner, or provider mechanism — and keep `internal` declarative.

This applies even to a new category that is obviously intended to be repeated later. Do not wait for many copies of boilerplate before designing the declaration mechanism.

### Good

A project profile declares data:

```go
var Profile = kernel.Profile{
    Code: "main",
    Modules: []kernel.ProfileModule{
        {Module: core.Module{}, Config: core.Config{...}},
        {Module: seo.Module{}},
        {Module: admin.Module{}},
    },
}
```

The application/runtime machinery discovers the declarations and knows how to validate/build them.

A route declaration should describe route-specific information and handler behavior; generic mounting, ordering, middleware composition, profile/site compilation, and conflict validation belong to reusable routing machinery.

A migration/seed declaration should describe the migration/seed and its unique execution behavior; discovery, ordering, filtering, transaction/lifecycle handling, and runner orchestration must not be rewritten in each project declaration.

### Bad

Do not make every profile/module/route/migration/seed repeat generic steps such as:

```text
create registry
register itself manually
resolve all generic dependencies manually
run common validation
sort all declarations
mount generic groups
execute generic lifecycle hooks
update global runtime maps
```

unless that code is genuinely unique project behavior.

### Important distinction

Minimal project wiring does **not** mean all project code must be tiny.

Project-specific business behavior may be substantial. A custom handler, service, policy, module, or transformation can contain real logic when that logic is unique to the project.

What must stay minimal is the **routine registration and lifecycle glue** around it.

## Extension Model

The intended extension precedence is:

```text
core < package < project < site
```

When an extensible registry supports replacement, overrides must be explicit and deterministic. Do not silently overwrite accidental duplicates.

Project/site layers should customize through declarations/extensions rather than by copying lower-level framework mechanics.

## HTTP

- Site-facing routes and middleware should resolve against the final site runtime.
- Explicit platform/control-plane endpoints may be global when their semantics are application-wide.
- Generic route compilation, prefix handling, middleware composition, conflict detection, and dispatch belong to reusable HTTP/runtime machinery, not repeated `internal` bootstrap code.
- Keep route declarations focused on path/method/handler/policy-specific information.

## Go Design Rules

- Prefer explicit Go dependency wiring over reflection-based DI containers.
- Introduce interfaces for behavior/boundaries, not merely to mirror every struct.
- Keep interfaces near the code that consumes/owns the contract when practical.
- Prefer composition over inheritance-like abstractions.
- Avoid global mutable state.
- Avoid `any` when a useful compile-time contract exists; use it only where heterogeneous declarations genuinely require it and validate at the boundary.
- Pass `context.Context` through operations that may block, perform I/O, call external systems, or are request/lifecycle scoped. Do not store request contexts in long-lived structs.
- Return and wrap errors with useful operation context. Do not hide failures behind fallbacks.
- Do not introduce abstractions for hypothetical future requirements when the current architecture does not need them.

## Change Discipline

For architecture/refactoring work:

1. Inspect the current code and call graph first.
2. Identify the owner and scope of the behavior being changed: application, profile, site, module, request, or project.
3. Preserve correct existing abstractions instead of rewriting unrelated code.
4. Make the smallest coherent architectural change.
5. If a task exposes repeated `internal` glue, fix the reusable declaration mechanism rather than adding another copy.
6. Keep the code compiling during incremental refactors when practical.
7. Do not perform unrelated cosmetic package moves or mass renames.

## Validation

For backend Go changes, run from `backend/` when practical:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

Use narrower tests during development, then run the broad checks for architecture/refactor work.

Do not claim tests/build succeeded unless they were actually run successfully.
