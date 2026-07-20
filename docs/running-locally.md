# Running Locally

## Development Environment Setup

You have two options for setting up your development environment:

1. Use the [Dev Container](#development-container), either locally or via [GitHub Codespaces](https://github.com/codespaces). This is usually the fastest and easiest way to get started.
1. [Manual installation](#manual-installation) of the necessary tooling. This requires a basic understanding of administering Kubernetes and package management for your OS.

### Initial Local Setup

Unless you're using GitHub Codespaces, the first step is cloning the Git repo into `$GOPATH/src/github.com/argoproj/argo-workflows`. Any other path will break the code generation.

### Development Container

Prebuilt [development container](https://containers.dev/) images are provided for both `amd64` and `arm64` containing all you need to develop Argo Workflows, without installing tools on your local machine. Provisioning a dev container is fully automated and typically takes ~1 minute.

You can use the development container in a few different ways:

1. [Visual Studio Code](https://code.visualstudio.com/) with [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers). Open your `argo-workflows` folder in VSCode and it should offer to use the development container automatically. VSCode will allow you to forward ports to allow your external browser to access the running components.
1. [`devcontainer` CLI](https://github.com/devcontainers/cli). In your `argo-workflows` folder, run `make devcontainer-up`, which will automatically install the CLI and start the container. Then, use `devcontainer exec --workspace-folder . /bin/bash` to get a shell where you can build the code. You can use any editor outside the container to edit code; any changes will be mirrored inside the container. Unlike the VS Code extension, the CLI does not forward ports to your host. The dev stack binds its services (UI `8080`, server `2746`, metrics `9090`, Tilt UI `10350`) to `0.0.0.0`, so reach them via the container's IP — `docker inspect <container> --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'`, then e.g. `http://<ip>:8080`.
1. [GitHub Codespaces](https://github.com/codespaces). You can start editing as soon as VSCode is open, though you may want to wait for `pre-build.sh` to finish installing dependencies, building binaries, and setting up the cluster before running any commands in the terminal. Once you start running services (see next steps below), you can click on the "PORTS" tab in the VSCode terminal to see all forwarded ports. You can open the Web UI in a new tab from there.

Once you have entered the container, continue to [Developing Locally](#developing-locally).

The container runs [k3d](https://k3d.io/) via [docker-in-docker](https://github.com/devcontainers/features/tree/main/src/docker-in-docker) so you have a cluster to test against. To communicate with services running either in other development containers or directly on the local machine (e.g. a database), the following URL can be used in the workflow spec: `host.docker.internal:<PORT>`. This facilitates the implementation of workflows which need to connect to a database or an API server.

Note for Windows: configure [`.wslconfig`](https://docs.microsoft.com/en-us/windows/wsl/wsl-config#configuration-setting-for-wslconfig) to limit memory usage by the WSL2 to prevent VSCode OOM.

### Manual Installation

To build on your own machine without using the Dev Container you will need:

* [Go](https://golang.org/dl/)
* [Yarn](https://classic.yarnpkg.com/en/docs/install/#mac-stable)
* [Docker](https://docs.docker.com/get-docker/)
* [`protoc`](http://google.github.io/proto-lens/installing-protoc.html)
* [`node`](https://nodejs.org/download/release/latest-v16.x/) for running the UI
* [`k3d`](https://k3d.io/) to run a local Kubernetes cluster
* [Tilt](https://tilt.dev/) to build images and run Argo in that cluster
* The following entries in your `/etc/hosts` file:

    ```text
    127.0.0.1 dex
    127.0.0.1 minio
    127.0.0.1 postgres
    127.0.0.1 mysql
    127.0.0.1 azurite
    ```

We use [k3d](https://k3d.io/) for the local Kubernetes cluster since it is fast and lets you test RBAC set-up. You don't
need to create the cluster by hand — `make start` (below) runs `make k3d-up`, which creates it if needed (pinned to a
supported Kubernetes version from `hack/k8s-versions.sh`) and wires up your kube config. No image registry is needed:
Tilt delivers images straight to the cluster with `k3d image import`. To create or delete the cluster directly:

```bash
make k3d-up    # create the cluster
make k3d-down  # delete the cluster
```

!!! Note
    If your `KUBECONFIG` lists multiple files, `make k3d-up` writes the cluster's kube config to a dedicated file
    (`~/.kube/configs/k3d-k3s-default.yaml`) and prints how to add it to your `KUBECONFIG`.

!!! Warning
    Do not use Docker Desktop's embedded Kubernetes, it does not support Kubernetes RBAC (i.e. `kubectl auth can-i` always returns `allowed`).

## Developing locally

Everything runs in your local k3d cluster via [Tilt](https://tilt.dev/). To start:

```bash
make start
```

This ensures the k3d cluster exists (`make k3d-up`) and runs `tilt up`. Tilt then:

* Builds the controller, server and executor images and runs them **in-cluster** (not as host processes).
* Runs the UI (`yarn start`) with hot-reload on <http://localhost:8080>.
* Port-forwards the Argo Server to <http://localhost:2746> and the controller metrics to <http://localhost:9090>.
* Port-forwards the backing services for the profile: MinIO (<http://localhost:9000>, use `admin`/`password`) so you
  can use artifacts, plus the database (`mysql`/`postgres` profiles) and Dex (`sso` profile).

Tilt prints a web UI (default <http://localhost:10350>) where you can watch each resource and tail its logs.

You can submit a workflow for testing using `kubectl` (the cluster's current namespace is `argo`):

```bash
kubectl create -f examples/hello-world.yaml
```

### Inner loop

When you edit Go source, Tilt recompiles the affected binary on the host, rebuilds its (small) image and recreates the
pod — typically around ten seconds. UI edits hot-reload via webpack. There is no separate build step to run, and you do
not need to build the executor image by hand (`make argoexec-image`) — Tilt builds and imports it for you.

!!! Note "Error `expected 'package', found signal_darwin`"
    You may see this error if symlinks are not configured for your `git` installation.
    Run `git config core.symlinks true` to correct this.

### Profiles

Use `PROFILE` to choose what gets deployed; it is passed to Tilt as `--profile`. The default is `minimal`.

To test the workflow archive, use `PROFILE=mysql` or `PROFILE=postgres`:

```bash
make start PROFILE=mysql
```

You'll have, either:

* Postgres on <http://localhost:5432>, run `make postgres-cli` to access.
* MySQL on <http://localhost:3306>, run `make mysql-cli` to access.

To back up the database, use `make postgres-dump` or `make mysql-dump`, which will generate a SQL dump in the `db-dumps/` directory.

```console
make postgres-dump
```

To restore the backup, use `make postgres-cli` or `make mysql-cli`, piping in the file from the `db-dumps/` directory.

Note that this is destructive and will delete any data you have stored.

```console
make postgres-cli < db-dumps/2024-10-16T17:11:58Z.sql
```

To test SSO integration, use `PROFILE=sso`:

```bash
make start PROFILE=sso
```

Other profiles include `plugins` (executor plugins) and `telemetry` (OpenTelemetry tracing to an in-cluster collector).

Other `make start` options, passed through to the in-cluster manifests: `API=false` (skip the Argo Server),
`SECURE=true` (serve the API over TLS) and `POD_STATUS_CAPTURE_FINALIZER=false` (disable the pod status capture
finalizer on the controller).

### Running E2E tests locally

Start up Argo Workflows using the following:

```bash
make start PROFILE=mysql
```

The E2E tests run on your machine and reach the in-cluster services (Argo Server, MinIO, the database, Dex) over
`localhost` — Tilt port-forwards all of them, so no extra port-forwarding is needed.

If you want to run Azure tests against a local Azurite:

```bash
kubectl -n argo apply -f test/e2e/azure/deploy-azurite.yaml
kubectl -n argo port-forward deploy/azurite 10000:10000
```

#### Running One Test

In most cases, you want to run the test that relates to your changes locally. You should not run all the tests suites.
Our CI will run those concurrently when you create a PR, which will give you feedback much faster.

Find the test that you want to run in `test/e2e`

```bash
make TestArtifactServer
```

#### Running A Set Of Tests

You can find the build tag at the top of the test file.

```go
//go:build api
```

You need to run `make test-{buildTag}`, so for `api` that would be:

```bash
make test-api
```

#### Diagnosing Test Failure

Tests often fail: that's good. To diagnose failure:

* Run `kubectl get pods`, are pods in the state you expect?
* Run `kubectl get wf`, is your workflow in the state you expect?
* What do the pod logs say? I.e. `kubectl logs`.
* Check the controller and argo-server logs via `kubectl -n argo logs deploy/workflow-controller` and
  `kubectl -n argo logs deploy/argo-server`, or in the Tilt web UI (<http://localhost:10350>). Is anything
  logged at `level=error`?

If tests run slowly or time out, factory reset your Kubernetes cluster.

### Running UI E2E tests locally

The UI has browser-level end-to-end tests written with [Playwright](https://playwright.dev), separate from the
Jest unit tests (`yarn --cwd ui test`). They drive a real browser against a running stack.

Start the dev stack in one terminal — Tilt serves the UI on <http://localhost:8080>:

```bash
make start
```

Then run the tests in another terminal:

```bash
yarn --cwd ui install
yarn --cwd ui playwright install --with-deps chromium   # first run only
make test-ui-e2e                            # or: yarn --cwd ui e2e
```

Useful extras:

* `yarn --cwd ui e2e:ui` — interactive/headed runner for debugging.
* `yarn --cwd ui playwright show-report` — open the HTML report (including traces and videos for failures).

The tests authenticate by reading the `argo-server` service-account token (the same secret the Go e2e suite uses)
and log workflows in via the REST API, waiting for them to complete before making UI assertions. See
`ui/e2e/README.md` for details, and `ARGO_TOKEN` / `ARGO_UI_BASE_URL` for pointing at a different cluster or URL
(CI, for example, sets `ARGO_UI_BASE_URL=http://localhost:2746` to test the production UI bundle served in-cluster).

### Database Tooling

The `go run ./hack/db` CLI provides a few useful commands for working with the DB locally:

```console
$ go run ./hack/db
CLI for developers to use when working on the DB locally

Usage:
  db [command]

Available Commands:
  completion              Generate the autocompletion script for the specified shell
  fake-archived-workflows Insert randomly-generated workflows into argo_archived_workflows, for testing purposes
  help                    Help about any command
  migrate                 Force DB migration for given cluster/table

Flags:
  -c, --dsn string   DSN connection string. For MySQL, use 'mysql:password@tcp/argo'. (default "postgres://postgres@localhost:5432/postgres")
  -h, --help         help for db

Use "db [command] --help" for more information about a command.
```

### Debugging under Tilt

The controller and server run in-cluster, so debugging means running them under
[Delve](https://github.com/go-delve/delve) and attaching from your IDE over a forwarded
port. Start the dev stack naming the component(s) you want to debug:

```bash
make start DEBUG=controller          # controller on :2345
make start DEBUG=controller,server   # also server on :2346
```

(equivalently, `tilt up -- --profile=minimal --debug=controller,server`). The `--debug`
flag is dev-only (it is ignored under `tilt ci`). It does three things for each named
component: rebuilds the binary with `-gcflags='all=-N -l'` (no optimisation/inlining, so the
debugger can map source lines and locals), wraps its in-cluster command with
`dlv exec --headless --listen=:<port> --continue --accept-multiclient`, and adds a port
forward for the Delve port. `--continue` means the program starts immediately, so the pod
stays Ready and you can attach whenever you like.

**Attach from your IDE** to `127.0.0.1:2345` (controller) or `127.0.0.1:2346` (server):

* **VS Code** — use the `Attach to workflow controller` / `Attach to argo server` launch
  configurations (shipped via `.devcontainer/devcontainer.json`; the same configs work
  outside the dev container if copied into a local `.vscode/launch.json`).
* **GoLand / JetBrains** — create a *Go Remote* run configuration pointing at the port, and
  uncheck *Shutdown remote debugger on disconnect* so detaching leaves Delve running.

Because the binaries are compiled on the host, their debug info already points at your local
source tree — no remote path mapping is needed.

Two things to know:

* While `--debug` is set, the component's compile step switches to **manual trigger mode**.
  Editing source no longer recreates the pod out from under your session; Tilt flags the
  change and you click the resource's update button when you are ready to rebuild and
  re-attach. Non-debugged components (and the UI dev server) keep rebuilding automatically.
* Debugging the controller sets `LEADER_ELECTION_DISABLE=true` (you will see it log
  *"Running in single-instance mode"*). Without this, pausing at a breakpoint stalls lease
  renewal and the controller exits when it loses leadership.
* In dev the server pod serves the **API only** — the UI is a separate webpack dev server
  (`yarn start`, on <http://localhost:8080>) that proxies API calls to the server on `:2746`.
  So while the server is paused at a breakpoint, in-flight UI requests will hang until you
  continue; you can use the UI to trigger an API call and hit a server breakpoint.

## Committing

Before you commit code and raise a PR, always run:

```bash
make pre-commit -B
```

Please do the following when creating your PR:

* [Sign-off](https://probot.github.io/apps/dco) your commits.
* Use [Conventional Commit messages](https://www.conventionalcommits.org/en/v1.0.0/).
* Suffix the issue number.

Examples:

```bash
git commit --signoff -m 'fix: Fixed broken thing. Fixes #1234'
```

```bash
git commit --signoff -m 'feat: Added a new feature. Fixes #1234'
```

### Creating Feature Descriptions

When adding a new feature, you must create a feature description file that will be used to generate new feature information when we do a feature release:

```bash
make feature-new
```

This will create a new feature description file in the `.features` directory which you must then edit to describe your feature.
By default, it uses your current branch name as the file name.
The name of the file doesn't get used by the tooling, it just needs to be unique to your feature so as not to collide on merge.
You can also specify a custom file name:

```bash
make feature-new FEATURE_FILENAME=my-awesome-feature
```

You must have an issue number to associate with your PR for features, and that must be placed in this file.
It seems reasonable that all new features are discussed in an issue before being developed.
There is a `Component` field which must match one of the fields in `hack/featuregen/components.go`

The feature file should be included in your PR to document your changes.
Before submitting, you can validate your feature file:

```bash
make features-validate
```

The `pre-commit` target will also do that.

You can also preview how your feature will appear in the release notes:

```bash
make features-preview
```

This command runs a dry-run of the release notes generation process, showing you how your feature will appear in the markdown file that will be used to generate the release notes.

## Troubleshooting

* When running `make pre-commit -B`, if you encounter errors like
  `make: *** [pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json] Error 1`, ensure that you
  have checked out your code into `$GOPATH/src/github.com/argoproj/argo-workflows`.
* If you encounter "out of heap" issues when building UI through Docker, please validate resources allocated to Docker.
  Compilation may fail if allocated RAM is less than 4Gi.
* To start profiling with [`pprof`](https://go.dev/blog/pprof), pass `ARGO_PPROF=true` when starting the controller locally.
  Then run the following:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile   # 30-second CPU profile
go tool pprof http://localhost:6060/debug/pprof/heap      # heap profile
go tool pprof http://localhost:6060/debug/pprof/block     # goroutine blocking profile
```
