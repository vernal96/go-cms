---
name: go-cms-mail
description: Use for GO CMS mail templates, rendered messages, SMTP/null/log transports, asynchronous mail jobs, previews/manual sending, delivery attempts/history, attachments, permissions and site-scoped mail administration.
---

# GO CMS Mail

Follow root `AGENTS.md`, `backend/AGENTS.md`, `go-cms-templating` for interpolation, `go-cms-events-jobs` for asynchronous delivery semantics, `go-cms-authorization` for protected operations and `go-cms-api` for management HTTP integration.

## Module boundary

Mail is an optional feature module, not core infrastructure and not an SMTP wrapper embedded in `kernel/app`.

Mail owns:

- mail template CRUD and validation;
- template variable schema;
- preview/render orchestration;
- immutable rendered messages queued for delivery;
- mail send job handler;
- transport contracts/manager;
- delivery attempts/status history;
- mail permissions and admin UI/API contributions.

Mail must not own the generic templating language. Use `kernel/templating`.

Generic `kernel/app` must not import `modules/mail`, keep `*mail.Management`, expose `MailManagement()` or acquire one feature-specific field/method per optional module. Wire management HTTP through the generic optional-feature contribution/composition mechanism described by `go-cms-api`.

SMTP/provider-specific code stays at the infrastructure edge.

## Site scope

Mail templates and normal mail history are site-scoped unless an explicit global use case is introduced.

Use the existing SiteRuntime/module wiring and site-scoped authorization patterns. Do not create a mutable global active mail site/config.

## Template identity and version

Mail templates need persistence identity, stable semantic code and a monotonic version.

Conceptually:

```text
ID      -> database/admin identity
Code    -> integration identity, e.g. feedback_notification
Name    -> human label
Version -> optimistic consistency identity
```

Other modules reference stable template codes rather than database IDs.

Template codes are unique in their intended scope and validated using project conventions.

Increment `Template.Version` for every successful semantic template update. Do not reset or silently reuse versions.

## Template variable schema

Reuse the existing core `field.Definition`/field type system instead of inventing another dynamic-form schema.

Mail template-owned input variables become `data.<key>` placeholders.

A variable may be either required or optional. Preserve the template author's `Required` choice.

Required variable:

- missing or empty at preview/send -> validation error;
- do not render it as an empty string.

Optional variable:

- missing/empty -> empty scalar output or absent semantic reference;
- preview returns a structured missing-value warning;
- omission alone does not block send.

If a non-empty value is provided, normalize/validate it through the established field type/rules.

Keep semantic types intact long enough to resolve files/attachments correctly; do not eagerly stringify all inputs.

Do not globally alter core field semantics to implement Mail behavior.

## Built-in site variables

Mail templates also receive site-owned variables automatically; the administrator does not redeclare them as `data.*` fields.

Use the same namespace conventions as SEO so different features do not invent incompatible names.

Expose template-safe built-in site properties at least as:

```text
site.id
site.profile_code
site.domain
site.locale
site.is_public
```

Expose every field declared by the current site's `Profile.Params` as:

```text
site.field.<key>
```

The values come from the current prepared SiteRuntime/Site snapshot (`Site.Settings` contains validated profile parameter values). Do not accept caller-provided values for `site.*`; they are backend-authoritative.

Do not expose internal/audit/storage implementation fields merely because they exist on the Go `site.Site` struct. In particular, avoid making `CreatedBy`, `UpdatedBy`, `FileReferences` or persistence internals part of the public template contract unless explicitly requested later.

The template editor/metadata API should return the available site variables for the currently selected site/profile so the frontend can present them as a separate `Сайт` group alongside user-declared `data.*` variables.

Keep profile field values typed until the consuming context decides how to represent them. A profile field of file type is not generically converted to a numeric ID string. In Mail text/HTML/header scalar contexts, only convert it to an externally safe representation (for example a public URL) according to Mail policy. In attachment context it may be resolved as an attachment source when the field type is compatible.

SEO and Mail should share one core/site-owned variable catalog/value-source helper where this avoids duplicated hardcoded `site.domain`, `site.locale`, `site.field.*` catalogs. The generic `kernel/templating` package itself still must not import or know the Site domain.

## Template fields

A mail template should support at least:

- From display name/address;
- To recipients;
- CC recipients;
- BCC recipients;
- Reply-To;
- Subject;
- content mode (`text` or `html`);
- text or HTML body according to mode;
- static/variable attachments;
- declared variables;
- enabled/disabled state;
- stable code/name/version.

Every string-capable field may use generic placeholders where appropriate.

Do not store recipients as one opaque comma-separated string. Keep address templates structurally separated into name/address entries.

Do not introduce default From name/address fallback configuration. The template/configured scenario must render a valid non-empty From address; otherwise preview/send fails validation. Required variable placeholders used by From obey the normal required-variable rule.

The rendered From may be used as the SMTP envelope sender under the configured sender allowlist/policy. Do not invent a second default envelope sender merely to make an incomplete template sendable.

## Rendering and preview

Render all mail fields before a message is queued.

Use context-aware templating:

- body text -> plain text;
- HTML body -> HTML context with dynamic values escaped by default;
- Subject/address display names/address fields/Reply-To -> header-safe context with CR/LF/control injection protection.

After rendering, validate final email addresses. Empty optional recipient entries may be skipped; at least one final recipient must remain.

Preview and queue must use the same authoritative backend renderer.

HTML preview in admin uses a sandboxed iframe or equivalent isolation, never unrestricted template HTML injected into the CMS DOM.

## Preview/send consistency

A user must not preview one rendered message and silently send a different one.

Template version is necessary but not sufficient because rendering can also depend on backend-authoritative `site.*` values and resolved static/site attachment metadata.

Preview returns at least:

```text
template_version
preview_fingerprint
```

`preview_fingerprint` is a backend-generated deterministic hash/fingerprint of the semantically rendered preview state required to detect changes in final From/To/CC/BCC/Reply-To/Subject/body and attachment snapshot metadata/checksums. It must not be based on frontend-trusted rendered data.

Manual Send submits template identity, raw typed values, `expected_template_version` and `expected_preview_fingerprint`.

Backend re-loads current template/site/runtime/file dependencies, checks template version, re-validates and re-renders from authoritative inputs, computes the current fingerprint and compares it with the expected preview fingerprint. A mismatch returns the established conflict error / HTTP 409 and the UI requires a fresh Preview.

This catches not only template edits, but also relevant site setting changes or resolved attachment metadata/content checksum changes between Preview and Send.

Do not trust rendered recipients/body/fingerprint source material posted back from the frontend. The client may only return the opaque server-produced fingerprint.

The immutable queued message stores historical template version and the final rendered snapshot. The preview fingerprint itself may be stored for diagnostics but is not a security credential.

## Immutable rendered message

Do not enqueue `template_id + variables` and re-render later in the worker.

```text
template + typed values
        -> validate/render/resolve
        -> immutable rendered Message snapshot
        -> persist status=queued
        -> durable mail.send(message_id)
```

Template edits after queueing never modify an already queued message.

Persist enough historical template/origin metadata to understand old messages after template changes/deletion.

## Jobs and outbox

Mail delivery is background work and uses the existing job/outbox implementation.

The job payload is small and stable, normally only persisted `message_id` plus generic envelope metadata.

Persisting the queued rendered Message and durable outbox job must be atomic.

The mail job handler:

1. loads the immutable message;
2. atomically claims it / rejects terminal duplicates;
3. records a delivery attempt;
4. resolves/opens already-authorized attachments;
5. sends through the selected logical transport;
6. records transport result;
7. applies retry/terminal policy;
8. remains safe under at-least-once job delivery.

Do not hold a database transaction open across SMTP network I/O.

Do not claim cross-system exactly-once SMTP semantics. A crash after remote SMTP acceptance but before local persistence can cause retry. Use stable RFC Message-ID plus local claim/lease/idempotency controls and document the residual boundary.

## Retry and terminal failure

Retry behavior is explicit and bounded.

Configure a positive maximum attempts value such as `MAIL_SEND_MAX_ATTEMPTS`.

Classify transport failures where practical:

- network/timeout/transient transport failures -> retryable;
- SMTP 4xx -> retryable;
- clear SMTP 5xx permanent rejection -> terminal;
- missing/deleted immutable attachment reference -> terminal unless the concrete error is demonstrably transient;
- unknown transport alias/configuration error -> terminal;
- reaching max attempts -> terminal.

For retryable failure below max attempts:

- persist failed attempt and retryable message state/metadata;
- return the correct error so existing job/event-bus redelivery handles retry.

For terminal failure:

- persist terminal failed state/metadata;
- acknowledge/return success according to current runner semantics so the broker does not retry forever.

Do not make every `failed` Message automatically claimable forever.

## Message status vs delivery attempts

Separate message lifecycle from technical attempts.

Useful message states/metadata represent:

```text
queued
sending
accepted
failed (retryable or terminal explicitly distinguishable)
```

SMTP acceptance means remote SMTP accepted the message; it does not prove mailbox delivery/read.

Delivery attempts store useful safe diagnostics:

- attempt number;
- logical transport/driver;
- started/finished timestamps;
- result/status;
- SMTP/provider response code/message ID where available;
- safe error text;
- retryable/terminal information where useful.

Never persist SMTP credentials/tokens in history.

## Transport abstraction

Mail feature code depends on a transport contract, not SMTP directly.

Use logical transport aliases so templates select a route such as `default` while project config maps it to concrete drivers.

Initial drivers may include SMTP, null and log.

Credentials/host/TLS/auth belong to env/config/secrets, never database templates.

Support explicit timeout and secure TLS behavior. Do not silently downgrade required TLS.

## Sender policy

Rendered From is controlled by project/transport sender policy.

Prevent templates/user data from turning the CMS into an unrestricted spoofing relay.

Header values reject CR/LF/control injection.

No default From fallback is required: an invalid/empty required sender is a validation error.

## Attachment authorization: static vs variable

This distinction is a security invariant.

### Static template attachment

A static attachment is trusted template configuration.

When a user creates/updates a template and selects a static file, validate that file using the editing actor's normal file authorization and file constraints at template-write time.

After the template is validly saved, a different user who has Mail send permission but not generic `core.file.read` may send that template. They are using already-approved template configuration; they are not choosing an arbitrary file.

The send worker may later open that persisted static attachment with trusted/system delivery access because authorization was established when the template configuration was saved and the immutable Message was queued.

Do not require every send-only operator to inherit global file-read permission merely because the template has a static attachment.

### Manual file variable

A file variable supplied during manual preview/send is untrusted actor input.

Validate the selected file ID using the current manual actor's file authorization and declared field constraints. A user with `mail.message.create` but without permission to read the referenced file must not be able to exfiltrate it by posting its numeric ID directly to Mail API.

Never resolve manual file-variable values with `security.System()` before authorization.

### Automatic/system file variable

A trusted backend module calling the automatic Mail API may resolve file variables under trusted/system semantics when that module intentionally supplies the file reference. Preserve automatic origin metadata.

### Site file field

A compatible `site.field.<key>` file value is backend-authoritative site configuration. It does not come from manual user input. Treat its authorization according to the already-validated site configuration/runtime boundary, while still applying Mail's external URL/attachment safety rules.

### Worker

After queueing, the immutable Message contains only already-authorized attachment references/snapshots. The asynchronous worker has no browser actor and may use the trusted delivery/file-open path.

Add negative tests proving arbitrary manual `file_id` cannot bypass file permissions.

## Temporary/transient attachments from Forms and other backend modules

Future Forms may supply uploaded files that are intentionally temporary and must NOT become ordinary CMS `core.File` records, must NOT appear in FileExplorer, and must NOT be retained as permanent user-managed files.

Because Mail delivery is asynchronous, the bytes must outlive the original HTTP request. Therefore truly keeping the bytes nowhere is impossible. Do not work around this by rendering/sending SMTP synchronously and do not put large binary bodies into the job payload.

Use a Mail-owned temporary attachment spool abstraction.

Conceptually Mail accepts an automatic semantic file input such as:

```text
stored CMS file reference
OR
transient attachment stream + filename + MIME + size
```

Manual browser Mail APIs continue using authorized CMS file IDs for file fields; do not expose arbitrary spool object keys to clients.

For a transient attachment:

1. validate filename/MIME/size and Mail limits;
2. copy the stream into a private Mail spool before queueing;
3. persist only an opaque spool reference plus immutable metadata/checksum on the Message attachment snapshot;
4. create Message + Outbox transaction referencing that already-prepared spool object;
5. if the DB transaction fails, best-effort delete the newly created spool object;
6. worker opens the spool object through the Mail attachment-store abstraction;
7. keep it across retryable delivery failures;
8. delete it after terminal accepted/failed state;
9. independently TTL-clean abandoned/orphaned spool objects so process crashes do not leak data forever.

The spool is internal transport storage, not the CMS file catalog. Backing it with a module-bound private filesystem/object-storage adapter is acceptable; do not insert rows into `core.files` and do not expose it in FileExplorer.

Prefer a small Mail-owned contract such as a temporary `AttachmentStore`/`Spool` over teaching `core.File` about ephemeral mail/form files.

Use the existing module filesystem binding mechanism (`ModuleContext.Filesystems()`) where suitable so Mail depends on a logical spool alias, not a physical local/S3 implementation. Project composition maps that alias to infrastructure.

Do not store temporary attachment bytes in PostgreSQL JSON/bytea merely for convenience unless an explicit future requirement justifies that trade-off.

The automatic Mail API used by Forms must be able to provide transient file inputs without forcing Forms to first create permanent CMS Files.

## File scalar interpolation

A file variable used in text/HTML is different from an attachment.

Feature-specific Mail resolution may convert an authorized persistent/site file to a safe public URL only when it is actually suitable for external recipients.

Private/admin-only URLs must not be emitted into outbound email. Private files may still be real attachments after proper authorization.

A transient spool attachment has no public URL by default. Do not make `{{data.temp_file}}` emit an internal spool path/key. It may be used as an attachment but scalar interpolation must fail or be absent according to declared semantics unless an explicit safe URL policy is introduced later.

Generic templating never decides how file IDs/spool references become URLs.

## File/media distinction

Persistent static/manual file selection uses Files and reuses the existing file picker/upload machinery.

Transient automatic attachments are Mail spool objects, not Files and not Media.

Do not create a Media field solely for Mail.

A future Media field for semantic image use cases (e.g. SEO OG image) is separate.

## Manual sender actor snapshot

Manual message history must record immutable requester identity metadata.

Persist at least user ID and historical display name.

Do not trust `requested_by_name` / actor display name from the frontend. Resolve it on the backend from the authenticated actor/current user service before creating the immutable Message.

Automatic/system sends use explicit origin metadata instead.

## Limits and abuse protection

Validate configurable limits before queueing, including at least:

- maximum final recipient count across To/CC/BCC;
- maximum outgoing message/attachment size;
- maximum delivery attempts;
- transient spool attachment size within the same outgoing-message policy.

Do not rely on HTTP request body limits as the mail-message-size policy.

Return clear validation errors.

## Manual send admin workflow

The admin UI provides:

- template list/CRUD;
- manual send screen;
- choose template;
- dynamically render declared fields including required markers;
- preview through backend authoritative renderer;
- warnings for missing optional values;
- validation errors for missing required values;
- final Send enabled only for a current successful preview;
- 409 stale-template/site/dependency handling requiring new preview;
- history/message list;
- message detail/status/attempts.

For HTML templates use existing rich HTML editor; for text use textarea.

Reuse current file picker/upload UI for static attachments and manual file variables, but backend remains authoritative for file access.

The template editor should show available placeholders grouped by source, at least:

```text
Сайт
  {{site.id}}
  {{site.profile_code}}
  {{site.domain}}
  {{site.locale}}
  {{site.is_public}}
  {{site.field.<profile-param>}}

Данные шаблона
  {{data.<declared-variable>}}
```

Do not require administrators to redeclare site profile fields in every Mail template.

## Template lifecycle

Prefer enabled/disabled/archived behavior for integration templates.

Do not casually hard-delete a stable template code referenced by Forms or another module. If physical deletion remains supported, protect references by explicit rules and keep historical Message snapshots independent of template existence.

## Permissions

Use existing module/entity/action permission conventions plus site-scoped checks.

Expected capabilities include template read/create/update/delete-or-disable, message/history read, manual message create/send, and history delete only when intentionally exposed.

Backend authorization is authoritative. UI visibility is UX only.

Mail history can contain sensitive personal/form data and needs its own permission.

Automatic trusted sends do not impersonate arbitrary browser permissions; preserve origin metadata.

## History reads and retention

Message history list endpoints must be bounded and lightweight.

Use a summary projection for lists. Do not load full rendered text/HTML bodies and full attachment snapshots for every row when the UI only shows metadata.

Load the full immutable body on message detail only.

Support practical server-side history filters when implemented, including status, template, date range and recipient search.

Mail history can contain personal data and grow indefinitely. Provide configurable retention including an explicit retain-indefinitely mode. Cleanup is application/background work, bounded in batches, and never deletes active queued/sending/retryable messages.

Mail message retention and temporary spool retention must be coordinated but are not identical: terminal messages may retain attachment metadata after spool bytes are deleted.

## SMTP result semantics

Do not label SMTP `250` as guaranteed `delivered` or `read`. Use accepted/"Передано SMTP" semantics.

Provider delivery/bounce/open/click webhooks are future scope.

## Testing invariants

Add focused tests for at least:

- required variable missing -> validation error;
- optional variable missing -> empty/absent + warning;
- unknown/malformed placeholders fail;
- built-in `site.*` variables are backend-authoritative;
- profile `site.field.*` variables are derived from the current profile params/settings;
- HTML escaping and header injection rejection;
- final address/no-recipient validation;
- template version increments and stale preview/send -> conflict;
- site/attachment dependency change after Preview -> fingerprint conflict;
- static attachment requires editor file permission at template save;
- a send-only actor may send an already-approved static attachment without global file-read permission;
- manual file variable cannot bypass current actor file permission by posting an arbitrary ID;
- automatic trusted persistent file variable follows explicit system semantics;
- transient automatic attachment queues without creating `core.File`/FileExplorer entries;
- transient spool bytes survive until worker delivery/retry and are removed after terminal state;
- failed Message/outbox creation cleans newly spooled data best-effort and TTL cleanup handles orphan leftovers;
- transient spool key/path is never exposed as a public template scalar or browser-controlled reference;
- immutable Message unaffected by later template edits;
- queued Message + outbox atomicity;
- stable RFC Message-ID;
- duplicate terminal job does not resend locally;
- SMTP transient retry, SMTP permanent failure and max-attempt terminal behavior;
- manual requester ID/name resolved on backend;
- recipient/message-size limits;
- site and Mail permission isolation;
- lightweight history list vs full detail;
- retention excludes active messages;
- frontend preview/stale conflict/history critical paths.

## Scope control

Do not add unless explicitly requested:

- Forms module itself;
- newsletters/bulk campaigns/subscriber lists;
- SMS/Telegram/WhatsApp;
- conditional/loop template language;
- open/click tracking;
- provider bounce/delivery webhooks;
- inline CID images;
- generic notification workflow engine.

Mail should expose the automatic transient-attachment contract needed by future Forms, but do not implement Forms in a Mail task.

Build Mail as a clean consumer of generic templating + jobs/outbox so future Forms and other channels can reuse generic infrastructure without depending on Mail internals.
