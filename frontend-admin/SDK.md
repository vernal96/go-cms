# Admin extension SDK

Package with Node 24+: `npm ci && npm pack`. The public package entry is
`@go-cms/admin/sdk`; component styles are exported as `@go-cms/admin/sdk.css`.
`prepack` rebuilds the SDK and validates its JavaScript, declarations and CSS
before creating the archive. Only `dist-sdk`, this guide and package metadata
are packaged. The package is publishable; these commands do not publish it.
`npm run build:sdk` remains available for local development.

From the repository root, `python3 scripts/check-admin-package.py` verifies a
clean source copy: installs locked build dependencies, packs without a manual
build, checks the actual archive exports, installs it into the independent
example plugin, and builds both the plugin/types and its demo host. Vue and the
shared UI libraries remain peer dependencies. Browser/API behavior is checked
separately as described in `../examples/README.md`.

The SDK exports:

- `AdminPlugin`, `AdminRouteDefinition`, `AdminPluginRegistry`;
- the registry, access-token and permission injection keys;
- field/configuration metadata types, `DynamicField`, `DynamicFieldsForm`,
  `ConfigurationEditor`;
- `useFieldValidation`, value initialization/validation helpers;
- `useSelectedSite`, `adminRequest`, `adminBlob`, `AdminAPIError`.

A plugin exports an `AdminPlugin` and uses its namespace for routes and editors.
Install it, import it in `src/admin-plugins.ts`, and add it to `adminPlugins`.
Backend modules contribute semantic route/editor codes; installing Go code does
not install Vue components. Rebuild the admin application for new UI packages.

Libraries must externalize `vue` and `@go-cms/admin/sdk` and declare them as peer
dependencies. The SDK itself externalizes its UI libraries. Do not bundle a
second SDK or Vue instance: injection keys, selected-site state and reactivity
must be shared. The host Vite config resolves the public SDK entry to its own
source entry; independently installed consumers use the built package exports.

The host provides `adminPluginRegistryKey`, `adminAccessTokenKey` and
`adminPermissionsKey`. A field editor accepts `field`, `v-model`, `siteId`,
`accessToken`, and `resourceTemplates`. Disable attribute fallthrough or declare
these props so credentials never become HTML attributes. A configuration editor
accepts `fields`, an object `v-model`, `siteId`, `accessToken`, and `context`, and
may expose a synchronous `validate()` that throws on invalid input.

Use `useFieldValidation()` inside components; it binds validation to the injected
registry used for rendering. Outside components pass the registry explicitly to
`validateFieldValues(fields, values, registry)` and
`unsupportedFieldTypes(fields, registry)`. An absent custom editor fails
validation, including children of empty repeaters, without changing values.
Backend validation and authorization remain authoritative.

Pass shell routes as the second argument to `AdminPluginRegistry` so plugin
paths cannot collide with the shell. Structurally identical parameter routes
are rejected; static and parameter routes may coexist. This follows the host's
default case-insensitive, non-strict Vue Router configuration.

The independent package in `../examples/admin-plugin` demonstrates a page, field
editor and Forms configuration editor using only this public entrypoint.

`options.multiple: true` opts a scalar field into the shared ordered-list editor,
including custom semantic types. The registered editor receives one scalar value
per item; list bounds and indexed errors are handled by the SDK. The backend
compiler must support the same list options and validate the resulting array.
