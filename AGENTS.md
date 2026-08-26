# GO CMS Agent Instructions

## Branch and source of truth

- Work only on `main`. Do not inspect, compare, create, switch to, or modify other branches unless the user explicitly asks.
- Treat the current files on `main` as the source of truth for APIs and package layout.
- Do not add legacy compatibility, fallback paths, transitional APIs, or dead compatibility code unless explicitly requested.

## Pre-production compatibility policy

GO CMS is still in active development and has no production data that must be preserved.

- Existing development data, old schemas, obsolete APIs and previous internal behavior are **not compatibility constraints**.
- When a cleaner current architecture conflicts with preserving old development data or an old contract, prefer the clean architecture and change the code/schema directly.
- Do not add data-conversion pipelines, compatibility shims, dual reads/writes, fallback columns, deprecated endpoints, transitional aliases, legacy DTOs or migration-only application commands merely to preserve pre-production state.
- Destructive schema changes and resetting/recreating development data are acceptable when they simplify the correct current design.
- When an existing not-yet-production migration is wrong, prefer correcting/squashing the migration history according to the repository's current development conventions instead of layering production-style compatibility migrations on top solely to preserve local data.
- Update fixtures, seeds and tests to the new contract rather than keeping old behavior alive.
- Preserve legacy behavior/data only when the user explicitly states that compatibility with an external consumer or real production data is required for that task.

Do not interpret "data already exists in a developer database" as a reason to retain a bad design.

## Context and tool budget

Keep discovery proportional to the task.

- Start with the files/packages named by the task, then inspect their direct consumers, dependencies, and focused tests.
- Do not scan the whole repository when a local change can be understood locally.
- Do not call `codebase-memory-mcp/get_architecture` by default. Use it only for genuinely broad/cross-cutting architecture work when targeted repository inspection is insufficient.
- Do not reload files already inspected unless they changed or a missing detail requires it.
- Load the smallest applicable skill set. Usually one skill is enough; load an additional skill only when the task genuinely crosses both scopes. Never load all `.codex/skills/*` proactively.
- Prefer targeted searches and bounded command output over repository-wide dumps.

## Requirements gate

For a non-trivial new feature, architecture change, cross-cutting refactor, or materially ambiguous request, do not start implementation immediately.

1. Inspect the relevant current implementation first.
2. Resolve everything that can be answered from the repository without asking the user.
3. Identify only decisions that materially affect behavior, architecture, public APIs, persistence, authorization, lifecycle, compatibility, performance, or UX.
4. If material ambiguity remains, load `go-cms-requirements` and ask focused clarification questions before editing code.
5. Recommend a default when there is a clear preferred option and briefly explain the trade-off.
6. After the answers, restate the resolved goal, important constraints, chosen approach, and any explicit assumptions.
7. Implement only when no material product/architecture decision remains unresolved.

Do not use the requirements gate for a small mechanical change, routine CRUD whose contract is already established, a focused bug fix, or a task where the user has already specified the relevant decisions.

Never ask the user for information that can be reliably discovered from the current code, tests, configuration, or established project instructions.

## Repository boundaries

- `backend/` is the Go backend and backend infrastructure root.
- `backend/internal/` is the project layer: composition, profiles, project declarations, config and customizations.
- Frontends such as `frontend-admin/` are independent applications and communicate with the backend through public HTTP APIs.
- Do not import/share frontend source with the backend or embed frontend applications into the Go binary.

## Architecture invariants

- `backend/internal/` should describe **what** exists/configuration; reusable kernel/module code owns common registration, validation, ordering, build, reload and lifecycle mechanics.
- Generic kernel code owns contracts/mechanics and must not learn project-specific configuration or concrete infrastructure merely for convenience.
- Modules own their domain behavior, services, repository contracts and module runtime. Cross-module dependencies are explicit.
- Infrastructure stays layered: project selection -> connector/client -> module adapter -> repository -> service/runtime. Connectors do not know CMS entities.
- `App` is the composition/lifecycle root, not a forwarding facade for every domain method.
- Final runtime state is site-scoped. Build/rebuild it on boot/create/update/reload and keep it in process memory; never rebuild a full `SiteRuntime` per request and never use a mutable global active-site singleton.
- Immutable profile definitions/blueprints may be shared; final site registries/module runtimes must have the correct site scope.
- `core` and `admin` are mandatory profile modules; `core` is first. Dependency order must be deterministic.
- Cache stores are application-owned physical infrastructure; module cache aliases describe module-local storage policy/capability, not a concrete technology and not necessarily a domain component. One domain component may use several aliases/stores at once. Cache keys identify cached data; cache tags identify dependencies. Cache coherence is correctness: every supported mutation path must invalidate/update all affected cached reads, including dependencies spanning aliases/stores.
- Extension precedence is `core < package < project < site`; accidental duplicates are errors and intentional replacement must be explicit/deterministic.
- Prefer explicit constructors, factories, interfaces and registries over reflection DI/service locators. Avoid mutable global state and unnecessary `any`.

## Change discipline

- Make the smallest coherent change that satisfies the task; avoid unrelated renames, package moves and cosmetic refactors.
- Reuse a nearby established pattern before inventing a new abstraction.
- If adding another routine item would copy generic registration/lifecycle glue into `internal`, improve the reusable declaration mechanism instead.
- Pass `context.Context` through blocking/I/O/request/lifecycle operations; do not store request contexts in long-lived objects.
- Return explicit errors; do not hide failures behind silent fallbacks.

## Validation budget

Validate narrowly while iterating and broadly only when justified.

- Go: `gofmt` changed files and run focused package tests first.
- Run `go test ./...`, `go vet ./...`, and `go build ./...` once near completion for broad/cross-cutting backend changes, not after every edit.
- Frontend: run the narrowest relevant tests/typecheck first; run the full build near completion when the change spans the application.
- Do not claim a command succeeded unless it actually ran successfully.

## Skill routing

Use the smallest set of skills that covers the actual task. A workflow skill such as `go-cms-requirements` may be combined with the relevant domain skill. A second domain/cross-cutting skill is justified only when both sets of invariants are materially involved.

- `go-cms-requirements`: requirement discovery/clarification before non-trivial or ambiguous feature/architecture work.
- `go-cms-development`: cross-package backend architecture or reusable extension/composition work.
- `go-cms-runtime-integrity`: SiteRuntime/ProfileBlueprint/reload/publication/runtime cache-coherence work.
- `go-cms-cache`: cache contracts, stores, module cache aliases, cache keys/tags, TTL, invalidation/coherence, Remember/result caching, cache connectors or cache maintenance.
- `go-cms-architecture-review`: architecture/refactor/PR/commit review.
- `go-cms-admin-ui`: backend-driven admin extensibility/navigation/frontend plugin work.
- `go-cms-administration`: global system administration reserved for the protected built-in `admin` group, including administration navigation/pages, destructive maintenance and global cleanup operations.
- `go-cms-api`: HTTP API contracts, CRUD endpoint design, DTOs, validation, pagination/filter/sort, transport errors, site context and API-layer authorization wiring.
- `go-cms-authorization`: groups/roles/permissions, site-scoped access, create/view/edit/delete rules, authorizer contracts, permission composition and enforcement boundaries.
- `go-cms-widgets`: widget definitions, layouts, persistence, editing or rendering.
- `go-cms-resources`: resource types/tree/identity, Library/LibraryItem architecture, resource paths/routing, storage/lifecycle/moves or resource admin capabilities.
- `go-cms-resource-revisions`: resource version counters, immutable revision snapshots, optimistic locking, restore/rollback, revision authors, revision persistence and purge/retention behavior.
- `go-cms-resource-fields`: template/resource field values, typed persistence, filtering/sorting/indexing, storage kinds or migration away from JSONB resource settings.

For a focused local bug fix or routine CRUD change whose architecture and API contract are already established, root/backend instructions plus the affected code are normally enough.

## Human-facing Codex prompts

When asked to draft a Codex implementation prompt for this repository, also recommend the currently optimal available model and reasoning-effort/acceleration setting for that specific task. Do not hardcode a model recommendation in repository instructions; model availability changes over time.
