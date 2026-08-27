---
name: go-cms-mail
description: Use for GO CMS mail templates, rendered messages, SMTP/null/log transports, asynchronous mail jobs, previews/manual sending, delivery attempts/history, attachments, permissions and site-scoped mail administration.
---

# GO CMS Mail

Follow root `AGENTS.md`, `backend/AGENTS.md`, `go-cms-templating` for interpolation and `go-cms-events-jobs` for asynchronous delivery semantics.

## Module boundary

Mail is a feature module, not core infrastructure and not an SMTP wrapper embedded in `App`.

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

SMTP/provider-specific code stays in adapters/connectors at the infrastructure edge.

## Site scope

Mail templates and normal mail history are site-scoped unless the current architecture establishes an explicit global use case.

Use the existing SiteRuntime/module wiring and site-scoped authorization patterns. Do not create a mutable global active mail site/config.

## Template identity

Mail templates need both persistence identity and stable semantic code.

Conceptually:

```text
ID   -> database/admin identity
Code -> integration identity, e.g. feedback_notification
Name -> human label
```

Other modules should reference stable template codes rather than database IDs.

Template codes are unique in their intended scope and validated using the project's normal code conventions.

## Template variable schema

Reuse the existing core field-definition system for declared user/input variables where possible instead of inventing a second form-schema/type system.

Mail template variables normally become `data.<key>` placeholders.

For manual sending, declared variables are optional by Mail product semantics even if a generic field definition normally has required semantics. Missing values render as empty strings / empty references and produce preview warnings rather than blocking solely because they were omitted.

If a non-empty value is provided, normalize/validate it through the established field type/rules.

Keep semantic types intact long enough to resolve attachments/files correctly; do not eagerly convert every input into a string before the Mail renderer uses it.

## Template fields

A mail template should support at least:

- From display name/address;
- To recipients;
- CC recipients;
- BCC recipients;
- Reply-To;
- Subject;
- content mode (`text` or `html`);
- text or HTML body according to the selected mode;
- static/variable attachments;
- declared variables;
- enabled/disabled state;
- stable code/name.

Every string-capable field may use generic placeholders where appropriate.

Do not store all recipients as one unparsed comma-separated string. Keep address templates structurally separable into name/address entries so rendered addresses can be validated safely.

## Rendering and preview

Render all mail fields before a message is queued.

Use context-aware templating:

- body text -> plain text;
- HTML body -> HTML context with dynamic values escaped by default;
- Subject/address display names/address fields/Reply-To -> header-safe context with CR/LF injection protection.

After rendering, validate final email addresses with the appropriate standard parser/validation. Empty rendered recipient entries may be skipped; at least one final recipient must remain.

Preview must use the exact same rendering pipeline used to build the queued rendered message.

HTML preview in the admin frontend should use a sandboxed iframe or equivalently isolated rendering rather than injecting arbitrary message HTML into the CMS application's own DOM.

## Immutable rendered message

Do not enqueue `template_id + variables` and re-render later in the worker.

When a send is requested:

```text
template + typed input values
        -> validate/render/resolve
        -> immutable RenderedMessage snapshot
        -> persist message status=queued
        -> enqueue mail.send(message_id)
```

This guarantees that the worker sends exactly what the user previewed / what the business operation prepared, even if the template changes afterwards.

Persist historical template identity metadata (at least template ID if applicable, code and human name) on the message snapshot so history remains understandable after template edits/disablement.

## Jobs and outbox

Mail delivery is background work.

Use the existing job abstraction. The concrete job payload should be small and stable, normally only the persisted `message_id` plus schema metadata already supplied by the job envelope.

Creation of a queued rendered message and durable enqueue/outbox state must obey the current Jobs/Outbox reliability architecture. Do not use an HTTP request context as worker lifetime context.

A mail send job handler:

1. loads the immutable queued message;
2. safely claims it / rejects terminal duplicates;
3. records a delivery attempt;
4. resolves/open attachments required for this attempt;
5. sends through the selected logical transport alias;
6. records provider/SMTP result;
7. updates terminal/non-terminal message state according to retry policy;
8. remains safe under at-least-once job delivery.

Do not claim cross-system exactly-once SMTP semantics. A crash after remote SMTP acceptance but before local success persistence can cause a retry. Use a stable RFC Message-ID and local claim/idempotency controls to minimize duplicate side effects and document the residual boundary.

## Message status vs delivery attempts

Separate business message state from technical attempts.

Message states should represent the useful lifecycle, e.g.:

```text
queued
sending
sent/accepted
failed
```

Name/display the SMTP terminal success accurately: SMTP acceptance means the remote server accepted the message; it does not prove final recipient delivery/read.

Store delivery attempts separately with useful diagnostics such as:

- attempt number;
- transport alias/driver metadata safe to expose;
- started/finished timestamps;
- result/status;
- SMTP/provider response code/message ID when available;
- safe error text;
- next retry / terminal information when the current job system exposes it.

Never persist SMTP passwords/tokens in history.

## Transport abstraction

Mail feature code depends on a transport contract, not SMTP directly.

Use logical transport aliases/purpose names so templates can select a logical route (e.g. `default`, `transactional`) while project configuration maps those aliases to concrete adapters/drivers.

Initial useful drivers:

- SMTP for real delivery;
- null for discard/testing;
- log for development diagnostics if it can be implemented without leaking sensitive content by default.

Credentials and physical SMTP host/TLS/auth settings belong to project env/config/secrets, not database templates.

Support explicit timeouts and secure TLS behavior. Do not silently fall back to insecure SMTP.

## Sender policy

Template From headers may be templated, but the transport/project policy remains authoritative over allowed envelope sender/from domains.

Prevent templates/user data from turning the CMS into an unrestricted sender/spoofing relay.

Header values must reject CR/LF injection.

## Attachments

Reuse the existing filesystem/file picker and upload machinery instead of creating a second upload subsystem.

Attachment template source modes should support at least:

```text
static   -> a selected file reference
variable -> a declared compatible file variable
```

Only variables with a compatible semantic field type may be selectable for file attachments.

Keep file references typed until send-time resolution. Generic templating must not open files.

Snapshot enough attachment metadata onto the rendered message/history to explain what was sent (filename, MIME, size, source/reference metadata as appropriate), but do not duplicate large binary bodies into PostgreSQL merely for mail history.

The send worker may resolve/read the current referenced file when sending. If product requirements later demand immutable binary preservation after source-file deletion, solve that explicitly with a retention/copy policy rather than accidentally storing MIME blobs in the message table.

Inline/CID attachments are out of first scope unless explicitly requested.

## File/media distinction

Attachments use Files.

Do not create a new Media field solely for Mail if the existing `file` type/picker satisfies attachment needs.

A future Media field may be appropriate for semantic image/media use cases such as SEO OpenGraph images, but it is independent of this Mail feature.

## Manual send admin workflow

The admin UI should provide:

- template list/CRUD;
- manual send screen;
- choose template;
- dynamically render the declared input fields;
- optional input values;
- preview using backend authoritative render endpoint;
- clear warnings for empty/missing variables;
- final Send action that queues the exact rendered snapshot;
- history/message list;
- message detail with recipients/content metadata/status/attempts.

For `html` templates use the existing rich HTML editor; for `text` use a textarea.

Reuse current filesystem picker/upload UI for static file attachments and manual file-variable values.

## Template lifecycle

Prefer enabled/disabled/archived behavior for templates already used by integrations. Do not casually hard-delete a stable template code referenced by Forms or another module.

If physical deletion is supported, protect referenced templates according to explicit repository/domain rules and keep message-history snapshots independent of template existence.

## Permissions

Use the existing module/entity/action permission conventions and site-scoped checks.

Expected capabilities include the equivalent of:

- template read/create/update/delete or disable;
- message/history read;
- manual message create/send;
- message/history delete only if product requirements expose it.

Backend authorization is authoritative. Hiding buttons/routes is UX only.

Mail history may contain sensitive personal/form data, so do not expose it merely because an actor can view normal site content.

Automatic/system sends initiated by trusted module jobs/events do not impersonate a browser user's arbitrary permissions; preserve actor/origin metadata separately where useful.

## Manual vs automatic origin

Persist enough origin metadata on the rendered message to distinguish at least:

- manual admin send with requesting user ID/name where appropriate;
- automatic/system/module-triggered send.

This is mail history metadata, not a substitute for the future global Audit module.

## History retention

Mail history can grow and can contain personal data.

Provide an explicit configurable retention policy, including a way to retain indefinitely when intentionally configured.

Cleanup is operational/background work and must be bounded/safe. Do not delete active queued/sending messages as ordinary retention cleanup.

Keep cleanup rules separate from outbox retention and resource revisions.

## SMTP result semantics

Do not label an SMTP `250`/accepted result as guaranteed "delivered" or "read".

Future provider webhooks may add richer states such as delivered/bounced/opened/clicked, but those are not part of the initial SMTP implementation.

## Testing invariants

Add focused tests for at least:

- template variable schema and rendering across subject/from/to/cc/bcc/reply-to/body;
- omitted declared variables become empty + preview warning;
- malformed/unknown variables fail validation;
- HTML values are escaped;
- header CR/LF injection is rejected;
- final recipient validation and no-recipient failure;
- text vs HTML editor/API contract;
- static and variable attachments;
- immutable rendered message is unaffected by later template edits;
- queued message + durable job/outbox path is reliable;
- duplicate job delivery does not resend a locally terminal message;
- attempt history records failure and eventual acceptance;
- SMTP/null/log adapter behavior as applicable;
- stable Message-ID;
- manual sender actor metadata;
- permission and cross-site isolation;
- history retention excludes active messages;
- frontend template/manual-send/preview/history critical paths.

## Scope control

Do not add in the initial implementation unless explicitly requested:

- newsletter/bulk marketing campaign engine;
- subscriber lists;
- SMS/Telegram/WhatsApp transports;
- conditional/loop template language;
- open/click tracking;
- provider webhooks for bounce/delivery;
- inline CID images;
- generic notification workflow engine.

Build Mail as a clean first consumer of templating + jobs/outbox so later channels can reuse the generic pieces without depending on the Mail module.
