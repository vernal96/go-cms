---
name: go-cms-resource-fields
description: Use for GO CMS template/resource field definitions and values, field persistence, field storage kinds, filtering/sorting/indexing, migration away from resource settings JSONB, file/reference field persistence, or field query adapter work for ordinary resources and LibraryItems.
---

# GO CMS Resource Fields

Follow root `AGENTS.md` and `backend/AGENTS.md`. Load `go-cms-resources` when resource identity, Library/LibraryItem storage or resource lifecycle is also affected. Load `go-cms-development` for cross-package reusable architecture changes.

Preserve the existing separation between field **definitions/schema** and field **values**. Field definitions belong to templates/code/runtime metadata; persisted field values belong to resource instances.

## Core rule

Template field values must not be stored in generic resource `Settings` JSONB.

Keep these concepts separate:

```text
Field Definition   typed declaration in template/module/project code
Field Schema       compiled validation/normalization metadata
Field Values       persisted values of one ResourceEntity
Type Settings      resource-type configuration, not template fields
```

A resource-facing domain/API model may still expose field values conveniently as:

```go
Fields map[string]any
```

but the persistence adapter should store queryable scalar values in a typed form.

Do not mix Library configuration such as URL patterns/default templates into `Fields`; those belong to the resource type settings model.

## Reuse current field validation

Keep the current field `Definition`, `Type`, `ValueType`, schema compilation and validation model unless a concrete deficiency requires change.

The normal write flow remains conceptually:

```text
input Fields map
 -> template FieldSchema.Validate/Normalize
 -> validate references/files
 -> persist normalized typed field values
```

Do not duplicate field normalization rules in PostgreSQL or the admin frontend.

A field type used in persisted template fields must expose valid storage semantics at template/schema compilation time. Do not allow a template to compile successfully and fail only on the first resource save because its custom `ValueType` lacks a supported storage kind.

## Storage kind abstraction

Field type codes are extensible and must not be hard-coded into SQL adapters one-by-one.

Introduce a small infrastructure-neutral storage/value kind contract associated with the compiled field value type, conceptually:

```text
string
integer
float
boolean
timestamp
reference
json
```

Multiple semantic field types can share one storage kind, for example:

```text
string / textarea / email / phone / radio / scalar select -> string
int                                                   -> integer
float                                                 -> float
checkbox                                              -> boolean
file/resource reference                               -> reference
json                                                  -> json
```

A future custom field type may define its own semantic validation/editor while reusing a standard storage kind. Keep storage kind about persistence/comparison semantics, not admin presentation.

Do not leak PostgreSQL column names into field type contracts.

## Typed field-value persistence

Use a dedicated repository/table keyed by stable resource identity rather than storing all fields in one JSONB object.

A suitable PostgreSQL shape is conceptually:

```text
resource_id
site_id                when required for integrity/scoping
field_key
position               0 for scalar, ordinal for multi-value
value_kind
value_string
value_integer
value_float
value_boolean
value_timestamp
value_reference
value_json
```

Exact naming may follow repository conventions, but preserve these invariants:

- one stable owning resource identity;
- field key is explicit;
- scalar values are typed;
- multi-value fields have deterministic order/position;
- only the column appropriate to the storage kind is populated;
- values for ordinary TreeResources and LibraryItems use the same field-value abstraction.

Avoid creating physical SQL columns dynamically for every project-defined field.

## Multi-value fields

Do not serialize a multi-select/list into one opaque JSON value when its members are expected to be filterable individually.

Represent filterable multi-values as multiple rows with the same `(resource_id, field_key)` and deterministic `position`.

For scalar fields, use exactly one logical value with position `0`.

Keep JSON storage for genuinely structured/opaque data where relational filtering is not part of the field contract.

Define negative multi-value predicates using set semantics, not per-row inequality. For example, `roles NOT IN ["editor"]` means the resource must have no matching `editor` row; it must not pass merely because another row such as `author` is different. Implement negative membership/equality with `NOT EXISTS` (or an equivalent correct relational form) where required.

Also define behavior for missing fields explicitly. Do not let SQL `NULL`/absence produce accidental semantics that differ between scalar and multi-value fields.

## Query contract

Preserve adapter-neutral field paths such as:

```text
resource.field.<key>
```

Do not expose SQL/EAV details to callers.

The resource query layer should continue to express filters/sorts semantically. PostgreSQL translates them into typed field-value predicates/joins/`EXISTS` expressions using the field's storage kind.

Examples:

```text
resource.field.salary >= 150000
resource.field.city == "Moscow"
resource.field.remote == true
```

must compare typed columns, not cast JSONB text at runtime.

This vocabulary must be reusable by both ordinary resource queries and high-volume LibraryItem queries. Do not build a second weaker LibraryItem filter model that cannot query typed template fields.

Validate that an operator is compatible with the field/storage kind before hitting persistence. Ordering operators require an orderable storage kind.

When sorting by field values, define deterministic behavior for missing values and add resource ID as a stable tie-breaker.

If the same field key can resolve to incompatible storage kinds across the queried template scope, reject/resolve the ambiguity explicitly rather than performing unsafe casts or silently choosing one kind.

## Indexing

The purpose of dedicated typed persistence is to make filtering/sorting/indexing predictable.

Prefer bounded generic indexes appropriate to common scalar access patterns, e.g. by field key + typed value and/or owning site/library scope where justified by real queries.

Do not create one SQL index automatically for every field declaration during routine runtime startup.

Avoid B-tree indexing large textarea/opaque JSON values by default. Search/full-text indexing is a separate capability and should not be conflated with ordinary equality/range filters.

If field metadata later gains `Filterable`, `Sortable`, `Searchable` or similar capabilities, treat them as semantic/query metadata rather than PostgreSQL-specific DDL flags.

For high-volume LibraryItems, inspect query shapes before finalizing composite indexes so library ownership/site scope participates where needed.

## Writes and transactions

Updating resource/template fields must be atomic with the owning resource mutation when partial state would be invalid.

A repository may implement a replace/upsert strategy, but it must:

- remove values no longer present;
- preserve normalized multi-value order;
- reject duplicate/invalid scalar state;
- keep resource/site ownership integrity;
- update file/reference relations consistently;
- avoid leaving old values after a template change.

Do not silently preserve values whose field key no longer exists in the selected template unless an explicit migration/archive feature is designed.

## Template changes

Changing a resource template changes the valid field schema.

The service must validate the resulting `Fields` against the new template and remove/reject incompatible old values deterministically. Admin confirmation may warn the user, but backend correctness cannot rely on the frontend clearing values.

`DefaultItemTemplate` for a Library affects creation only; once a LibraryItem stores a concrete template, its fields are validated against that template like any other resource.

## File and reference fields

Preserve existing file/media authorization and validation semantics when moving values out of JSONB.

A file/reference field should persist a stable referenced ID using the reference storage kind while any required FK/reference table remains adapter-owned. Do not weaken validation merely because storage changed.

When field values are replaced or a resource is permanently deleted, stale reference rows must be cleaned consistently.

## API and admin transport

Prefer explicit transport naming:

```json
{
  "template_code": "vacancy",
  "fields": {
    "salary": 150000,
    "city": "Moscow"
  },
  "type_settings": {}
}
```

Do not continue using `settings` as an ambiguous alias for template field values when making the breaking migration, unless an explicit compatibility requirement is given. Root instructions prefer the current clean contract over transitional dead compatibility.

The admin dynamic field UI may continue working with a `Record<string, unknown>` map; persistence normalization is a backend concern.

## Migration from current JSONB settings

When current `main` still stores template field values in `resources.settings`, implement a deliberate migration path rather than silently dropping values.

Before writing migration code, inspect whether `settings` currently contains only template field values or also resource-type/system configuration. Separate any non-field data into the new type-settings storage before removing/reinterpreting the column.

For existing rows:

1. determine the selected template and its field definitions;
2. normalize/map each known field value to its declared storage kind;
3. insert typed field-value rows;
4. preserve invalid legacy data only if the task explicitly requires compatibility; otherwise fail migration clearly rather than corrupting silently;
5. remove or repurpose the old `settings` field only after all readers/writers/query paths have moved.

Do not infer persisted semantics only from JSON shape when the template schema says otherwise. A JSON array belonging to an opaque `json` field must remain JSON; it must not automatically become a multi-value relational field. File/reference/custom field types must migrate according to declared field semantics, not merely according to `jsonb_typeof`.

If SQL migrations cannot access code-defined template schemas safely, use an explicit application/data migration mechanism or clearly constrain the migration policy for development databases. Do not silently perform a lossy best-effort conversion and call it schema-aware.

If this repository's migration policy prefers rebuilding early migrations rather than append-only production migrations, follow the current `main` conventions instead of assuming one strategy.

## Resource identity

Field values should attach to the common stable resource identity used by both TreeResources and LibraryItems. Do not create separate parallel field tables such as `resource_fields` and `library_item_fields` unless a measured adapter constraint requires it.

This is what allows templates, filtering and extensions to treat LibraryItems as resources without duplicating field behavior.

## Tests

Add focused tests for changed invariants, including as applicable:

- schema validation remains the source of normalized values;
- persisted template field types always expose storage semantics at compile time;
- scalar storage kind maps to the correct typed value;
- multi-value ordering and positive filtering;
- multi-value negative filters use correct set semantics;
- missing-field semantics are deterministic;
- update removes stale values;
- template change rejects/removes incompatible fields;
- file/reference validation remains enforced;
- field filters use typed semantics for string/numeric/boolean/reference values;
- ordinary Resource and LibraryItem share the same field-value query vocabulary;
- migration preserves existing valid field data according to declared template field semantics;
- opaque JSON arrays are not mis-migrated into multi-value rows;
- no resource query path still depends on JSONB `settings -> field_key` after migration.

Run focused package/adapter tests first, then broader backend integration tests when the migration affects resource service, PostgreSQL and public/admin query paths.