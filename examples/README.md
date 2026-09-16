# Independent CMS consumers

These fixtures verify package boundaries with import paths outside the CMS
module. They do not publish anything or import the CMS project's `internal`.

## Go application

`external-app` is a separate Go module (`example.org/cms-consumer`). Its own
`internal/notice` module contributes a persistent string field, a Forms element
and site navigation. Two profiles demonstrate enabling/disabling the extension.
The application uses public Core/Mail/Forms PostgreSQL adapters and the public
HTTP server. Project seeds are declared on the database binding.

Run `python3 scripts/check-external-packages.py` from the repository root. It
copies the consumer into a temporary directory, compiles it, and independently
relocates and builds every connector under `example.org/connector-*`. Without
integration environment variables the persistence test explicitly skips.

For integration tests create a **fresh isolated database**, with a name beginning
`cms_extension_`, then export `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`,
`EXAMPLE_DATABASE` and an arbitrary temporary `EXAMPLE_TOKEN`. Run the same script
or `go -C examples/external-app test -count=1 -v ./...`. The test covers metadata,
save/read of the custom field and Forms element, site isolation, and rejection
of an element on a site without the extension.

`go -C examples/external-app run .` serves the fixture at `127.0.0.1:18081`.
`EXAMPLE_ADDRESS` overrides the listener. This fixture uses a static test token
for its seeded user and a discarding event bus; it is not a production auth or
message-delivery example. Never point it at an application database.

## Independent admin plugin

Use Node 24+. From `frontend-admin`, run:

```sh
npm ci
npm run build:sdk
npm pack --pack-destination /tmp
```

From `examples/admin-plugin`, install the resulting archive and build:

```sh
npm install --no-save --package-lock=false /tmp/go-cms-admin-0.1.0.tgz
npm run build
npm run demo
```

The plugin imports only `@go-cms/admin/sdk` and Vue. Its demo host uses the
**installed archive**, not an alias into the CMS checkout, and proxies API calls
to the external Go application. In the browser set
`sessionStorage.setItem('example-token', '<EXAMPLE_TOKEN>')`, then reload
`http://127.0.0.1:18082/admin/example`. The page edits the custom site setting and
Forms element through real CMS APIs. Create the example form if it does not yet
exist. `?missing=1` omits the field editor to exercise blocked saving.

With Playwright available to Node, run
`node examples/admin-plugin/tests/browser.cjs` from the repository root, passing
`EXAMPLE_TOKEN`. Optional `BROWSER_BASE` and `BROWSER_CHROME` configure the server
and browser executable. It checks save/read after reload, site switching and
that a missing editor sends no mutations. The fixture must use a fresh database
(or one initialized by this fixture only).

For the normal CMS admin host, install the built plugin package and register its
`noticePlugin` in `frontend-admin/src/admin-plugins.ts`. New frontend packages
require rebuilding the admin. Runtime remote JavaScript loading is not used.
