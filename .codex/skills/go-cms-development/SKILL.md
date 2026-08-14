---
name: go-cms-development
description: Use for GO CMS backend architecture and implementation work involving App, profiles, sites, runtimes, modules, repositories, adapters, connectors, caches, filesystems, routes, migrations, seeds, registries, or project/internal composition. Enforces declarative project wiring and existing architecture boundaries.
---

# GO CMS Development

## Purpose

Use this skill when changing the GO CMS backend architecture or adding routine extensibility points.

The main objective is to preserve a modular Go architecture while making project-level composition under `backend/internal/` as declarative and boilerplate-free as possible.

Always follow the repository root `AGENTS.md` first. This skill adds the implementation workflow.

## 1. Inspect Before Editing

Before changing code:

1. Inspect the current implementation of the affected packages.
2. For broad architectural work, use `codebase-memory-mcp/get_architecture` if available, then verify against actual files.
3. Find an existing nearby pattern before inventing a new one.
4. Identify whether the task changes an existing abstraction or introduces a new extension category.
5. Do not infer package APIs from memory when the repository can answer the question.

For project declarations, use `backend/internal/profiles/dev/profile.go` as a style reference: declarations should mostly describe modules, configs, bindings, params, templates, or other unique project data rather than execute framework lifecycle logic.

## 2. Classify Ownership and Scope

Before writing code, classify the new behavior by owner and lifetime.

### Owner

Choose the narrowest correct owner:

- **kernel** — generic contracts, registries, factories, managers, runtime/lifecycle mechanics;
- **module** — module domain behavior, services, repository contracts, module runtime, module-specific extension mechanics;
- **adapter/connector** — concrete infrastructure integration;
- **project/internal** — declarations, composition choices, project-specific implementations/customizations;
- **frontend** — independent client application using public HTTP APIs only.

Do not place behavior in `App` or `internal` merely because it is convenient to reach from there.

### Scope

Classify state as one of:

```text
application scoped
profile scoped
site scoped
module scoped
request scoped
```

Use that scope to decide where the state is created and stored.

Important runtime rules:

- final site runtime state is site-scoped;
- SiteRuntime is assembled on application initialization/reload and stored in process memory;
- do not rebuild SiteRuntime per request;
- immutable profile blueprints may be shared;
- request context/actor/request data stays request-scoped;
- no mutable global active-site singleton.

## 3. Apply the Declarative Internal Rule

This is the primary project-specific rule.

For anything routinely added through `backend/internal/`, project code should state **what** is being added and its unique configuration/behavior. Reusable application/module code should implement **how** all declarations of that category are collected, registered, validated, ordered, initialized, compiled, run, reloaded, and closed.

### Boilerplate gate

Before adding project glue, ask:

> If another item of this same category is added tomorrow, which lines would be copied unchanged?

If the answer includes lifecycle/registration mechanics, do not copy them into `internal`.

Instead, improve the reusable declaration mechanism.

The project declaration should contain only:

- identity/code;
- unique config;
- unique bindings;
- unique dependencies;
- unique handler/business behavior;
- explicit ordering/override metadata only when semantically required.

Generic behavior must live in the appropriate reusable owner.

### Routine categories

Apply this rule aggressively to:

- profiles;
- module lists/configuration;
- connectors and adapter bindings;
- migrations;
- seeds;
- routes/route groups;
- middleware registrations;
- permissions;
- templates/widgets;
- resource/field types;
- cache/filesystem aliases;
- project extensions.

### Bad signal

Stop and redesign if adding one routine element requires project code to repeatedly:

- create a registry;
- manually register itself in several places;
- manually invoke common lifecycle hooks;
- manually sort/validate the whole category;
- manually resolve standard dependencies;
- manually update runtime maps;
- manually mount common route groups;
- manually wire the same infrastructure chain;
- manually append the same item to several unrelated bootstrap lists.

A routine element should normally have one obvious declaration/registration point.

## 4. App Changes

`App` is the composition root and central access point, not a universal domain service.

When changing App:

1. Keep lifecycle/composition ownership in App.
2. Keep assembled subsystems accessible through App when useful.
3. Put domain behavior in services/modules.
4. Avoid forwarding every service method through App.

Prefer:

```go
app.Users().Create(ctx, input)
app.Services().Resources.Find(ctx, id)
```

not:

```go
app.CreateUser(ctx, input)
app.FindResource(ctx, id)
```

If `app.go` grows because several concerns are implemented in one file, split package implementation by responsibility without inventing unnecessary public interfaces.

Examples of reasonable internal package separation:

```text
app.go
boot.go
infrastructure.go
services.go
runtimes.go
migrations.go
console.go
close.go
```

The exact file names are not architectural requirements.

## 5. Adding or Changing a Module

For a module task, follow this order:

1. Define the module responsibility.
2. Confirm which contracts belong to the module.
3. Define/configure the module declaration/runtime.
4. Keep module-specific services inside the module boundary.
5. Add repository/persistence contracts only when needed.
6. Add concrete DB/storage adapter separately.
7. Register the module in a profile declaratively.
8. Add tests.

Do not make kernel import feature-specific business packages just so a module can work.

Cross-module dependency must be explicit. If `admin` requires capabilities from `cms.core`, represent that dependency clearly rather than exposing arbitrary module services through an unrestricted generic context.

`cms.core` and `admin` are mandatory profile modules by project rule.

## 6. Adding a Profile

A profile declaration should be mostly data.

Good responsibilities for a profile file:

- profile code/name;
- params/schema declaration;
- templates;
- ordered module declarations;
- module configs;
- cache/filesystem/database bindings;
- project/site extension declarations.

Do not put generic profile build logic in the profile package.

The runtime factory/application must know how to:

- validate mandatory modules;
- validate duplicates;
- resolve bindings;
- build module runtimes;
- assemble final site runtime;
- handle common ordering/lifecycle.

If every profile needs a helper with the same body, that helper probably belongs outside `internal/profiles/<name>`.

## 7. Adding Routes or Middleware

Project/module code should declare route-specific information and actual handler behavior.

Reusable HTTP/runtime code should own:

- route collection;
- system/group prefix mounting;
- ordering;
- middleware composition;
- conflict detection;
- site-runtime compilation;
- method/path dispatch;
- 404/405 behavior;
- common lifecycle.

Avoid project bootstrap code such as repeated manual `router.Route(...)`, `Use(...)`, group mounting, or per-profile loops when a declaration/provider model can express the same intent.

Site-facing routes should use the final SiteRuntime. Explicit application-wide/control-plane routes may be global when that is their intended scope.

## 8. Adding Migrations or Seeds

First determine ownership:

- reusable module schema migration -> module infrastructure adapter/package;
- project-only schema/data -> project/internal declaration/provider.

Migration/seed files should contain only their unique metadata and execution logic.

Generic mechanics belong to the migration/seed subsystem:

- discovery/collection;
- ordering/versioning;
- filtering/tags;
- execution tracking;
- transactions where supported;
- lifecycle orchestration;
- reporting/errors.

Do not require every new migration/seed to edit several bootstrap files if a provider/list/registry can make registration local and declarative.

Prefer a stable declaration API over magic filesystem scanning when explicit Go registration provides better compile-time visibility. The goal is minimal project boilerplate, not hidden behavior for its own sake.

## 9. Adding Persistence

Preserve this chain:

```text
project/bootstrap
    -> concrete connector factory
    -> module adapter factory
    -> module Database/repository contracts
    -> repository implementations
    -> module service/runtime
```

### Connector

A connector is a low-level concrete technology client/pool. It should not know CMS entities.

### Module adapter

A concrete adapter may type-assert the generic connector to the technology it supports and build module repositories.

### Repository

Repository contracts belong to the domain/module that consumes them.

Do not put module repository interfaces into generic kernel just to make infrastructure wiring easier.

### Project layer

`internal` should select/configure factories and bindings. It should not manually construct every repository and service when a reusable adapter/runtime factory already has enough information to do it.

## 10. Cache and Filesystem Changes

Use aliases by purpose, not technology.

Good:

```text
repository
resources
templates
runtime
media
```

Bad module-facing API:

```text
redis
s3
local-disk
```

The project decides the implementation through bindings.

When runtime becomes site-specific, ensure namespaces/isolation can be site-specific where required without forcing every module to know the backing technology.

## 11. Registries and Overrides

The intended extension priority is:

```text
core < package < project < site
```

When extending a registry:

1. Decide whether the item is append-only or replaceable.
2. Keep accidental duplicate detection.
3. If replacement is supported, make override intent explicit.
4. Apply deterministic precedence.
5. Do not rely on incidental map iteration or registration timing.

Do not introduce override machinery to registries that have no actual override use case.

## 12. Dependency Rules

Before introducing a dependency, verify the direction.

Avoid:

```text
kernel -> cms.core business package
kernel -> project/internal
module -> project/internal
module -> concrete project connector configuration
```

Expected examples:

```text
project/internal -> modules/kernel
module -> kernel contracts
module postgres adapter -> postgres connector + module contracts
feature module -> explicit lower-level/module capability when designed
```

Do not introduce reflection-based DI containers. Prefer constructors, factories, explicit interfaces, registries, and composition.

## 13. Avoid Premature Complexity

The declarative rule is not permission to build a generic metaframework for every one-off behavior.

Use a reusable declaration mechanism when the category is inherently repeatable or lifecycle/registration logic is generic.

Keep unique business logic direct and readable.

Prefer:

```text
small declaration + reusable mechanism
```

instead of either extreme:

```text
large repeated project bootstrap
```

or:

```text
abstract framework with no concrete current need
```

## 14. Implementation Workflow

For each task:

1. Read the relevant declarations and runtime/manager/registry that consumes them.
2. State internally who owns the behavior and what its scope is.
3. Identify existing boilerplate and whether this task would add another copy.
4. If needed, improve the reusable declaration/consumer API first.
5. Add the smallest project/module declaration needed.
6. Preserve current correct behavior unrelated to the task.
7. Add/update focused tests.
8. Run formatting and validation.

For large refactors, keep changes incremental and compiling when practical.

Do not mix architectural refactors with unrelated package renames or cosmetic cleanup.

## 15. Review Checklist

Before finishing, verify:

### Architecture

- Is dependency direction still correct?
- Did kernel gain concrete technology or feature-specific business knowledge?
- Is module ownership clear?
- Is cross-module dependency explicit?
- Is state stored at the correct application/profile/site/module/request scope?
- Is App still primarily composition/lifecycle rather than domain behavior?

### Declarative project layer

- Does `internal` describe intent/configuration rather than framework mechanics?
- Would the next similar item require copying generic glue?
- Can routine registration be done from one obvious place?
- Did I accidentally require edits to several unrelated bootstrap lists?
- Does reusable code own common validation/order/build/run behavior?

### Infrastructure

- Are connector, adapter, repository, and service responsibilities separate?
- Are concrete technologies selected at the project/bootstrap edge?
- Are cache/filesystem bindings semantic rather than technology-named?

### Runtime

- Is final site runtime state site-scoped where required?
- Is runtime built/reloaded outside the request hot path?
- Are shared objects actually immutable/safe to share?
- Is there any new mutable global singleton?

### Quality

- Did I avoid unrelated cleanup?
- Are errors explicit rather than hidden by fallback behavior?
- Are tests focused on the changed contract/behavior?

## 16. Validation

For Go changes, run from `backend/` when practical:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

Use narrower package tests while iterating, but use broad checks for architecture/cross-cutting changes.

Never report a command as successful unless it was actually executed successfully.

## 17. Final Response

After implementation, report concisely:

1. what changed;
2. why the new ownership/scope is correct;
3. how project/internal boilerplate was kept minimal or reduced;
4. important files changed;
5. tests/build commands actually run and their results;
6. any intentionally deferred architectural work.
