# GO CMS Agent Instructions

## Branch and source of truth

- Work only on `main`. Do not inspect, compare, create, switch to, or modify other branches unless the user explicitly asks.
- Treat the current files on `main` as the source of truth for APIs and package layout.
- Do not add legacy compatibility, fallback paths, transitional APIs, or dead compatibility code unless explicitly requested.

## Context and tool budget

Keep discovery proportional to the task.

- Start with the files/packages named by the task, then inspect their direct consumers, dependencies, and focused tests.
- Do not scan the whole repository when a local change can be understood locally.
- Do not call `codebase-memory-mcp/get_architecture` by default. Use it only for genuinely broad/cross-cutting architecture work when targeted repository inspection is insufficient.
- Do not reload files already inspected unless they changed or a missing detail requires it.
- Load only the skill that directly matches the task. Never load all `.codex/skills/*` proactively.
- Prefer targeted searches and bounded command output over repository-wide dumps.

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

Load a skill only when its scope matches:

- `go-cms-development`: cross-package backend architecture or reusable extension/composition work.
- `go-cms-runtime-integrity`: SiteRuntime/ProfileBlueprint/reload/publication/runtime cache-coherence work.
- `go-cms-cache`: cache contracts, stores, module cache aliases, cache keys/tags, TTL, invalidation/coherence, Remember/result caching, cache connectors or cache maintenance.
- `go-cms-architecture-review`: architecture/refactor/PR/commit review.
- `go-cms-admin-ui`: backend-driven admin extensibility/navigation/frontend plugin work.
- `go-cms-widgets`: widget definitions, layouts, persistence, editing or rendering.
- `go-cms-resources`: resource types/tree/identity, Library/LibraryItem architecture, resource paths/routing, storage/lifecycle/moves or resource admin capabilities.
- `go-cms-resource-fields`: template/resource field values, typed persistence, filtering/sorting/indexing, storage kinds or migration away from JSONB resource settings.

For a focused local bug fix or routine CRUD change, root/backend instructions plus the affected code are normally enough.

## Human-facing Codex prompts

When asked to draft a Codex implementation prompt for this repository, also recommend the currently optimal available model and reasoning-effort/acceleration setting for that specific task. Do not hardcode a model recommendation in repository instructions; model availability changes over time.
