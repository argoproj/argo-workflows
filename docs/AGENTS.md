# Docs

- One sentence per line of markdown.
- New pages must be added to the `nav:` in `properdocs.yml` — the build is strict and fails otherwise.
- `make docs-serve` builds and serves on :8000 with the same checks as CI; `make docs` runs spellcheck, markdownlint, and the env-var documentation check.
- Generated pages — edit the source, not the markdown, then `make codegen -B`: `fields.md` (from swagger + `examples/`), `cli/` (from Cobra), `workflow-controller-configmap.md` (from doc comments on structs in `config/`), `metrics.md`/`tracing.md` (from `util/telemetry/builder/values.yaml`), `database-migrations.md` (from `persist/sqldb/migrate.go` and `util/sync/db/migrate.go`), `variable-flow/variables.md` (from `util/variables/`), `executor_swagger.md` (from doc comments in `pkg/plugins/executor/`), `docs/README.md` (copied from the root README).
- `go-sdk-guide.md` is half-generated: prose is hand-edited, but code blocks between `<embed>` markers are injected from `sdks/go` sources by embeddoc — edit the source, not the block.
