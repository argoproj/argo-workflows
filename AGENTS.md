# Argo Workflows

Kubernetes-native workflow engine: workflows are CRDs, each step runs as a container. Components: `workflow-controller` (reconciles Workflows into pods), `argo-server` (gRPC+HTTP API, hosted inside the `argo` CLI binary), `argoexec` (runs inside workflow pods), and a React UI. Go module `github.com/argoproj/argo-workflows/v4` (imports need `/v4`), vendored deps.

The repo must be checked out at `$GOPATH/src/github.com/argoproj/argo-workflows` or codegen breaks.

Per-directory guidance lives in nested AGENTS.md files: `workflow/controller/AGENTS.md`, `ui/AGENTS.md`, `docs/AGENTS.md`. Read the relevant one before working in those areas — agents that only auto-load CLAUDE.md files must read these explicitly.

## Commands

- Build: `make cli` (`dist/argo`), `make controller`, `make dist/argoexec`.
- Unit tests: `make test`. Single test: `KUBECONFIG=/dev/null go test -run TestFoo ./workflow/controller/`.
- E2E: bring the local stack up first (`make start PROFILE=mysql`), then `make test-<tag>` where `<tag>` is the `//go:build` tag at the top of the test file. Single e2e test: `make TestArtifactServer` (any `Test*` name works as a make target).
- Local dev stack (k3d + Tilt, everything in-cluster with hot reload), ports, profiles, debugging: see docs/running-locally.md.
- Lint: `make lint` (golangci-lint `--fix` + UI lint — commit what `--fix` changes; CI runs `git diff --exit-code`).
- Codegen: `make codegen -B`. Pre-PR: `make pre-commit -B` (= codegen, lint, docs).

## Conventions

- Commits and PRs: follow the checklist in docs/pull-requests.md (DCO sign-off + Conventional Commits enforced by the commit-msg hook, `make pre-commit -B`, feature files for `feat:`, tests required, AI declaration). A bot checks PR descriptions against `.github/pull_request_template.md` and drafts non-conforming PRs.
- Logging is the repo's own `util/logging`, not logrus: level methods take ctx (`logger.Info(ctx, "msg")`); get the logger with `logging.RequireLoggerFromContext(ctx)`; tests use `logging.TestContext(t.Context())`.
- Errors: `errors/` for coded, user-visible `ArgoError`s; `util/errors.IsTransientErr(ctx, err)` for retry decisions.
- Every env var read via `os.Getenv`/`util/env` must be documented in docs/environment-variables.md — `make docs` fails on undocumented or stale entries.

### Generated files — never hand-edit, regenerate with `make codegen -B`

CI regenerates and fails on diff: `pkg/client/`, `*.pb.go`, `generated.proto`, `zz_generated.deepcopy.go`, `openapi_generated.go`, `*.swagger.json`, `api/openapi-spec/`, `api/jsonschema/`, `manifests/install*.yaml`, `manifests/quick-start-*.yaml`, CRDs under `manifests/base/crds/`, `**/mocks/` (mockery), `sdks/java/client`, `sdks/go` — plus several docs/ pages generated from Go source that carry no DO-NOT-EDIT header (see docs/AGENTS.md).

`examples/` are validated by a unit test (a malformed example fails `make test`) and feed the generated `docs/fields.md`.

## Architecture

Entrypoints: `cmd/workflow-controller` → `workflow/controller/` (reconciliation; see workflow/controller/AGENTS.md); `cmd/argo` → `cmd/argo/commands/` (includes `argo server` → `server/`); `cmd/argoexec` → `workflow/executor/`.

Cross-cutting invariants: the executor never writes the Workflow object — outputs travel through `WorkflowTaskResult` CRs merged by the controller. Large node status is compressed and, if still too big, offloaded to SQL (`workflow/hydrator/`, `persist/sqldb/`) — any reader outside the controller must hydrate first.

### Executor (`workflow/executor/`)

`argoexec emissary` is PID 1 wrapping the user command in every container: writes `/var/run/argo/ctr/<name>/exitcode`, waits on ContainerSet dependencies, reads the template from `/var/run/argo/template`. `init`/`wait` (legacy pod layout) or `supervisor` (init-less beta) stage input artifacts before main and run the shared `PostMain` capture sequence after (script result → parameters/artifacts/logs → report outputs). Output capture deliberately uses a background context so termination doesn't lose outputs.

### Server (`server/`)

`server/apiserver` serves gRPC and grpc-gateway HTTP on one port; artifact up/download endpoints and the embedded UI (`ui/embed.go`) are mounted directly. Auth (`server/auth`) is chosen per-request from the Authorization header: `client` (caller's k8s token, their RBAC), `server` (server's service account), `sso` (OIDC claims mapped to service accounts via RBAC labels); gatekeeper interceptors stash per-request clients in ctx (`auth.GetWfClient(ctx)`). Workflow archive lives in `server/workflowarchive` over `persist/sqldb` (Postgres/MySQL).

`pkg/apiclient` gives every CLI command four transports behind one interface: argo-server gRPC, HTTP1 fallback, offline (lint without a cluster), and direct-kube (no server — service logic runs in-process against the k8s API).

### Shared layers

`workflow/common` (label/annotation keys, container names, `/var/run/argo` paths), `workflow/validate` (lint/submit/controller all share it), `workflow/templateresolution` (resolves `templateRef` chains, caches in `Status.StoredTemplates`), `workflow/artifacts` (driver per storage backend: s3, gcs, azure, oss, hdfs, git, http, raw, plugin), `workflow/sync` (semaphores/mutexes, configmap- or database-backed), `workflow/util` (submit/retry/resubmit operations shared by CLI and server), `workflow/cron`.
