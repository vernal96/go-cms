---
name: go-cms-api
description: Use for GO CMS HTTP API design or implementation: public/admin CRUD endpoints, route contracts, request/response DTOs, validation, transport errors, pagination, filtering, sorting, site context, API versioning/compatibility, and wiring authorization into handlers/application services. Do not use for domain-only changes with no HTTP contract impact.
---

# GO CMS API

Follow root `AGENTS.md` and `backend/AGENTS.md`. Load `go-cms-development` only when the API task also changes reusable backend architecture. Load `go-cms-authorization` when permission semantics or enforcement boundaries are materially involved.

## Discovery

Before designing a new endpoint, inspect:

1. the nearest existing route/handler in the same API family;
2. the domain/application service it calls;
3. request/response/error conventions already used by the project;
4. focused handler/service tests;
5. permission/site-resolution middleware or contracts when relevant.

Prefer extending an established API vocabulary over inventing a parallel convention.

## Boundary

Keep the flow conceptually:

```text
HTTP route/handler
  -> parse + transport validation
  -> resolve request/site/actor context
  -> application/domain service
  -> repository/domain behavior
  -> map result/error to HTTP response
```

Handlers are transport adapters, not the home of business behavior.

Do not put persistence queries, cache-coherence policy, resource lifecycle rules, permission policy, or cross-module orchestration directly in handlers when they belong to reusable services/domain layers.

HTTP DTOs may differ from domain entities. Do not expose persistence structs merely to avoid mapping code.

## CRUD consistency

Prefer a consistent resource-oriented surface for ordinary management APIs, for example:

```text
GET    /api/sites
POST   /api/sites
GET    /api/sites/{id}
PATCH  /api/sites/{id}
DELETE /api/sites/{id}
```

Use domain actions when the operation is not naturally CRUD (move, restore, publish, reorder, rebuild, etc.) rather than forcing every behavior into generic update payloads.

Do not create separate domain/service implementations for admin and public APIs when both can reuse the same underlying application/domain behavior with different transport policy, projections or authorization.

## Requests and validation

Separate:

- syntactic/transport validation: malformed IDs, invalid JSON, invalid enum representation;
- domain validation: illegal state transition, invalid owner/type, duplicate domain value;
- authorization: actor is not allowed to perform the operation.

Reject unknown or unsupported fields when silent acceptance would hide client mistakes.

For partial updates, define presence semantics explicitly. Do not conflate "missing", `null`, zero and empty string unless the contract intentionally does so.

Do not let transport defaults silently override domain defaults when omission has semantic meaning.

## Responses and errors

Use one established response/error convention across API families.

Map errors by meaning, not by string matching. Prefer typed/sentinel/domain errors that handlers can translate deterministically.

Distinguish at least where applicable:

```text
400 malformed/invalid transport input
401 unauthenticated
403 authenticated but forbidden
404 target not visible/found according to API policy
409 state/version/uniqueness conflict
422 structurally valid request rejected by domain validation (when this convention is established)
500 unexpected server failure
```

Do not leak internal package names, SQL errors, connector details, stack traces or sensitive authorization reasons in public responses.

## List APIs

List endpoints must be bounded.

Use a common query vocabulary for:

- pagination;
- filtering;
- sorting;
- search where supported.

Prefer deterministic ordering with a stable tie-breaker.

For high-cardinality collections such as LibraryItems, follow the domain skill and prefer cursor/keyset pagination on hot/deep paths rather than designing around large `OFFSET` scans.

Do not fetch all rows and paginate/filter in Go or in the frontend when the repository can do it efficiently.

Return enough pagination metadata for the client without exposing database implementation details.

## Site context

Site-scoped APIs must obtain site identity through the established resolver/request context. Do not rely on a mutable global current-site variable.

Validate that target entities belong to the resolved site where the operation is site-scoped. Avoid insecure direct-object access caused by looking up an entity globally and forgetting the site boundary.

A site selector in the admin UI is presentation state; backend authorization and ownership checks remain authoritative.

## Authorization wiring

Transport may authenticate the actor and pass actor/request context inward, but authorization policy must not exist only as button visibility or handler-local condition chains.

Prefer a reusable authorizer/application/domain boundary appropriate to the permission being checked. Every state-changing endpoint and every protected read must enforce the relevant permission even if the frontend hides the action.

Use `go-cms-authorization` for changes to the permission model itself.

## Compatibility and evolution

Current `main` is the source of truth. Do not add legacy aliases, duplicate endpoints, fallback payload formats or version bridges unless explicitly requested.

When intentionally changing an established public contract, identify direct consumers and tests before editing. If the change is destructive or compatibility expectations are unclear, use `go-cms-requirements` before implementation.

## API-driven frontend rule

Backend APIs should expose semantic capabilities and data required by clients, not frontend implementation details.

Do not send Vue component paths, executable JS, CSS classes tied to one framework, or frontend package internals through generic backend contracts.

For dynamic admin forms/navigation/widgets, expose semantic IDs, capabilities, field metadata and validated values; frontend resolves these through its installed implementation.

## Tests

Choose focused tests for the changed contract:

- request decoding and validation;
- success response shape/status;
- domain error -> HTTP error mapping;
- unauthenticated/forbidden behavior;
- site isolation/ownership;
- pagination/filter/sort determinism;
- partial update presence semantics;
- domain service called rather than behavior duplicated in handler.

For a new CRUD family, cover at least create/read/update/delete/list behavior that has distinct semantics; do not write repetitive tests that only exercise the router.

Run focused package tests during iteration. Use broad backend validation once near completion when the API change crosses several packages.

## Anti-patterns

Do not:

```text
handler -> SQL/connector directly
handler-local business lifecycle rules
permission enforced only by frontend visibility
unbounded list endpoints
fetch-all then paginate in memory
raw persistence structs as accidental API contracts
string-match errors to choose status codes
site-scoped entity lookup without site ownership validation
new /admin-specific domain service when generic domain behavior already exists
silent backward-compatibility/fallback endpoints not requested by the user
```
