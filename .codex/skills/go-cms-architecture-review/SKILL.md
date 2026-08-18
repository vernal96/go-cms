---
name: go-cms-architecture-review
description: Use when reviewing GO CMS architecture, a refactor, a commit/PR, or checking whether changes drift from project rules. Focuses on ownership, scope, runtime isolation, cache coherence, dependency direction, declarative internal code, and transport boundaries.
---

# GO CMS Architecture Review

Use this skill for architecture reviews, refactor reviews, PR/commit reviews, and before/after comparisons of cross-cutting backend changes.

Always apply the repository root `AGENTS.md` and `backend/AGENTS.md` first.

## 1. Review the Actual Change

Do not review from filenames or intent alone.

1. Inspect the changed files/diff.
2. Read the consumers of changed contracts.
3. Trace at least one real read path and one real write path for affected domain data.
4. For runtime changes, trace application boot/reload and request dispatch.
5. For cache changes, trace cache population and invalidation.
6. Compare behavior against the architecture rules, not against personal stylistic preference.

## 2. Classify Findings

Use these severity levels:

- **Critical** — likely data corruption, security issue, stale-data correctness bug, broken runtime publication, or major production failure.
- **Major** — violates a core architecture invariant and will materially increase coupling or block expected extensibility.
- **Medium** — meaningful design debt that should be corrected before the affected subsystem grows.
- **Minor** — local quality/clarity issue without architectural impact.
- **Acceptable trade-off** — intentional deviation with a clear reason and contained cost.

For each non-trivial finding state:

1. current behavior;
2. why it matters;
3. target behavior;
4. smallest coherent fix;
5. regression test that should prove the fix.

## 3. Ownership Check

For every changed responsibility identify its owner:

```text
kernel
module
adapter/connector
project/internal
HTTP transport/server
frontend
```

Flag behavior placed in a lower/generic layer only because it is convenient to access there.

Key questions:

- Did generic kernel gain `cms.core` or `admin` business knowledge?
- Did `App` regain domain CRUD behavior instead of exposing services?
- Did project/internal gain generic lifecycle or registration glue?
- Did a connector learn CMS entities?
- Did an HTTP concern leak into a domain/runtime package?

## 4. Scope Check

Classify state as:

```text
application
profile blueprint
site runtime
module runtime
request
```

Verify:

- immutable profile definitions are shared only when safe;
- final module runtimes/registries are distinct per site where required;
- no mutable active-site singleton exists;
- request context/actor is not retained in long-lived objects;
- site runtime rebuild occurs outside the request hot path;
- runtime replacement is atomic.

## 5. Cache Coherence Review

Treat cache coherence as correctness.

For every mutable cached entity trace:

```text
READ -> cache population
WRITE -> persistence mutation
INVALIDATE/UPDATE -> cache state
READ -> expected fresh result
```

Specifically look for split paths such as:

```text
site runtime reads -> cached repository
admin writes       -> uncached base repository
```

This is a bug unless a separate centralized invalidation mechanism covers the write.

Verify:

- all supported mutation paths invalidate/update affected cached data;
- invalidation does not depend on HTTP caller behavior;
- site-scoped namespaces can still be invalidated by application/admin operations;
- tags/keys include enough domain identity to invalidate narrowly and correctly;
- explicit shared namespaces are intentional;
- tests execute a real cache-populate -> mutate -> cached-read sequence.

Do not accept tests that only assert namespace strings if the concern is stale data.

## 6. Runtime and Module Dependency Review

Verify:

- `core` and `admin` are mandatory;
- `core` is first in every profile;
- module dependencies are declared explicitly;
- undeclared dependency lookup fails;
- dependency ordering is deterministic;
- `ModuleContext` is not a generic service locator;
- modules can access only the generic runtime/site scope they legitimately need;
- generic runtime scope does not import core site domain types.

## 7. HTTP Boundary Review

Site-specific HTTP compilation may happen once at initialization/reload, but transport ownership must remain clear.

Flag:

- `net/http.Handler` stored directly in core domain/site packages;
- core domain packages compiling/mounting generic server infrastructure;
- mutable handler binding that can leave partially prepared runtimes after an error;
- profile-level handler sharing when final behavior should be site-specific.

Prefer HTTP/server-owned maps/catalogs or a generic runtime attachment mechanism if runtime-associated artifacts are required.

## 8. Declarative Internal Review

For every changed file under `backend/internal/`, ask:

> If I add another item of this category tomorrow, what generic lines must I copy?

If common registration, validation, sorting, lifecycle, route mounting, migration discovery, dependency resolution, or runtime-map updates must be copied, flag the abstraction.

`internal` should mostly select/configure declarations and provide genuinely project-specific behavior.

## 9. Extension Review

Target precedence:

```text
core < package < project < site
```

Check whether a registry is:

- append-only;
- duplicate-protected;
- intentionally replaceable.

If replaceable, override intent and precedence must be explicit. If not replaceable, duplicate rejection is correct.

Do not demand override machinery without a concrete use case.

## 10. Required Regression Tests for Major Runtime Refactors

Prefer tests that prove behavior across boundaries:

- two sites/same profile -> different module runtime instances;
- profile blueprint reused safely;
- domain lookup -> prebuilt site runtime, no request rebuild;
- failed runtime rebuild -> old snapshot still available;
- declared dependency succeeds, undeclared dependency fails;
- core-first profile validation;
- cached read -> admin/service write -> next runtime read is fresh;
- site cache isolation when two sites share the same physical store;
- HTTP compilation remains site-specific without domain ownership of `net/http.Handler`.

## 11. Output Format

Return findings in priority order.

Start with a short verdict, then list only material findings. Avoid padding the review with minor style comments when architectural issues exist.

Conclude with:

- what is already correct and should be preserved;
- recommended fix order;
- whether the change is safe to continue building on.