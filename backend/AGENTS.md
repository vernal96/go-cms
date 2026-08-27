# GO CMS Backend Rules

These rules extend the repository root `AGENTS.md` for Go backend work. Do not repeat the root rules in task reasoning.

## Profiles and runtime

- `core` and `admin` are mandatory; `core` is first. Other module dependencies are explicit and deterministic.
- Profile files stay declarative. Reusable runtime/application code owns validation, dependency resolution, ordering and lifecycle.
- Preserve the model `Profile declaration -> immutable ProfileBlueprint -> per-site SiteRuntime`.
- Final registries and module runtime instances are site-scoped. Shared blueprint data must be immutable/safe to share.
- Build/rebuild runtimes outside the request hot path and publish replacements atomically. A failed candidate build must leave the previous working snapshot available.
- Runtime scope exposed to modules must use generic kernel contracts; do not make generic kernel depend on `core/site` domain types. Never persist request actor/context in runtime state.

## Ownership boundaries

- `App` owns composition/lifecycle; domain CRUD stays on module/domain services.
- Generic kernel packages must not import feature business packages just to assemble them.
- Transport-specific compiled state belongs to transport/server code. Do not turn core site/domain types into `net/http.Handler` containers.
- Concrete connector/adapter knowledge stays at infrastructure edges; repository contracts belong to the consuming domain/module.

## Cache correctness

If reads are cached, every supported writer must preserve coherence.

Avoid an uncovered split such as:

```text
site runtime read -> cached repository
admin write       -> uncached repository
```

Centralize invalidation/update near the persistence/cache boundary or a domain cache policy. Site namespaces must remain invalidatable by global/admin mutations. Prefer a regression test that performs `cached read -> real write -> cached read` and observes fresh data.

## Events, outbox and background work

- Keep primary business writes synchronous. Domain events describe facts that already committed; jobs describe asynchronous work to perform.
- When a committed DB mutation must reliably produce an external event, persist its outbox message in the **same physical database transaction** as that mutation. Never rely on `commit -> EventBus.Publish` as a reliable boundary and never publish externally before the DB transaction commits.
- Treat EventBus delivery as at-least-once. Consumers and job handlers that cause side effects must be idempotent or deduplicate by stable message/job identity; do not claim cross-system exactly-once semantics.
- Background publishers/workers are application-scoped lifecycle components. Start them only after their dependencies are ready and stop/cancel them before closing the EventBus/databases. Never retain request contexts in worker state.
- Keep broker technology at the connector edge. Domain/module code may define semantic events/jobs but must not import Kafka/RabbitMQ/PostgreSQL transport types for convenience.

For event/job/outbox work, load `go-cms-events-jobs` and follow its transactional, retry, retention and scope rules.

## Cross-cutting tests

For runtime/cache architecture changes, prefer focused behavioral tests for the invariant being changed, such as:

- two sites sharing a profile still receive correct site-scoped runtime instances;
- failed rebuild does not replace the active snapshot;
- declared module dependencies resolve and invalid ones fail clearly;
- cached data is fresh after real admin/service writes;
- site cache namespaces remain isolated when required;
- request dispatch uses a prebuilt runtime rather than rebuilding it.

During iteration run the affected package tests. Use repository-wide `test/vet/build` once near completion only for changes whose scope warrants it.
