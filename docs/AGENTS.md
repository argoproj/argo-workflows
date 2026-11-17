# Docs

- Follow `doc-changes.md` and the Kubernetes style guide it links: active voice, present tense, "you" for the reader, no statements about the future, and one sentence per line, also in the sources of generated pages.
- API kinds are plain text (Workflow, WorkflowTemplate); field names, values and commands go in back-ticks.
- Describe only what the release does, and name versions explicitly ("v4.2 or later", not "this release"); open proposals and planned features stay out.
- `upgrading.md` is replaced at each minor release, so don't link into or out of it.
- `.features/` notes become release notes: write for users, not reviewers, and remember featuregen drops blank lines, so no indented code blocks, unindented lists or separate paragraphs.
- Use Argo's terms only for what they name ("task" is a DAG task; likewise step, node, template, pod), and name the exact resource or tool (`workflow-controller-configmap` or a namespace's `artifact-repositories` configmap; `kubectl`).
- Be exact about behavior, checked against the code: whether a limit is per call or in total, whether a failure errors the node, when a boolean is true, what breaks on rollback or without the RBAC, the replacement for a deprecated field.
- Examples must work when copied as they are and follow the advice next to them; detail too long for a table cell goes in a section below the table that the row links to.
- New pages must be added to the `nav:` in `properdocs.yml` — the build is strict and fails otherwise.
- `make docs-serve` builds and serves on :8000 with the same checks as CI; `make docs` runs spellcheck, markdownlint, and the env-var documentation check.
- Generated pages — edit the source, not the markdown, then `make codegen -B`: `fields.md` (from swagger + `examples/`), `cli/` (from Cobra), `workflow-controller-configmap.md` (from doc comments on structs in `config/`), `metrics.md`/`tracing.md` (from `util/telemetry/builder/values.yaml`), `database-migrations.md` (from `persist/sqldb/migrate.go` and `util/sync/db/migrate.go`), `variable-flow/variables.md` (from `util/variables/`), `executor_swagger.md` (from doc comments in `pkg/plugins/executor/`), `docs/README.md` (copied from the root README); when rewording, also update the sample configmaps, `.features/` notes and code comments that repeat the text.
- `go-sdk-guide.md` is half-generated: prose is hand-edited, but code blocks between `<embed>` markers are injected from `sdks/go` sources by embeddoc — edit the source, not the block.
