# Project development seeds

`Dev()` returns the source `sites_dev` with its existing history identity and
`dev` tag. It is explicitly attached to the Core adapter in the project's
`DatabaseDefinition.Seeds`. Core only supplies the shared system groups.

The SQL files create this project's development sites, demonstration settings
and development admin account. Run `console seeds up -tags=dev` to apply them.

The project integration scenario uses its own database configuration:
`CMS_TEST_PROJECT_POSTGRES_HOST`, `_PORT`, `_DB`, `_USER`, `_PASSWORD`, `_SSL_MODE`.
Use a fresh isolated database, separate from `CMS_TEST_POSTGRES_*` used by Core
adapter tests, so parallel `go test ./...` packages cannot reset each other's data.
