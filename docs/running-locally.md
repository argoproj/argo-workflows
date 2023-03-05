# Running Locally

You have two options:

1. If you're using VSCode, you use the [Dev-Container](#development-container). This takes about 7 minutes.
1. Install the [requirements](#requirements) on your computer manually. This takes about 1 hour.

## Git Clone

Clone the Git repo into: `$GOPATH/src/github.com/argoproj/argo-workflows`. Any other path will mean the code
generation does not work.

## Development Container

A development container is a running Docker container with a well-defined tool/runtime stack and its prerequisites.
[The Visual Studio Code Remote - Containers](https://code.visualstudio.com/docs/remote/containers)  extension lets you use a Docker container as a full-featured development environment.

System requirements can be found [here](https://code.visualstudio.com/docs/remote/containers#_system-requirements)

Note:

* `GOPATH` must be `$HOME/go`.
* for **Apple Silicon**
    * This platform can spend 3 times the indicated time
    * Configure Docker Desktop to use BuildKit:

    ```json
    "features": {
      "buildkit": true
    },
    ```

* For **Windows WSL2**
    * Configure [`.wslconfig`](https://docs.microsoft.com/en-us/windows/wsl/wsl-config#configuration-setting-for-wslconfig) to limit memory usage by the WSL2 to prevent VSCode OOM.

* For **Linux**
    * Use [Docker Desktop](https://docs.docker.com/desktop/linux/install/) instead of [Docker Engine](https://docs.docker.com/engine/install/) to prevent incorrect network configuration by k3d

## Requirements

* [Go](https://golang.org/dl/)
* [Yarn](https://classic.yarnpkg.com/en/docs/install/#mac-stable)
* [Docker](https://docs.docker.com/get-docker/)
* [`protoc`](http://google.github.io/proto-lens/installing-protoc.html)
* [`node`](https://nodejs.org/download/release/latest-v16.x/) for running the UI
* A local Kubernetes cluster ([`k3d`](https://k3d.io/), [`kind`](https://kind.sigs.k8s.io/docs/user/quick-start/#installation), or [`minikube`](https://minikube.sigs.k8s.io/docs/start/))

We recommend using [K3D](https://k3d.io/) to set up the local Kubernetes cluster since this will allow you to test RBAC
set-up and is fast. You can set-up K3D to be part of your default kube config as follows:

```bash
k3d cluster start --wait
```

Alternatively, you can use [Minikube](https://github.com/kubernetes/minikube) to set up the local Kubernetes cluster.
Once a local Kubernetes cluster has started via `minikube start`, your kube config will use Minikube's context
automatically.

⚠️ Do not use Docker for Desktop with its embedded Kubernetes, it does not support Kubernetes RBAC (i.e. `kubectl auth can-i` always
returns `allowed`).

## Developing locally

Add the following to your `/etc/hosts`:

```text
127.0.0.1 dex
127.0.0.1 minio
127.0.0.1 postgres
127.0.0.1 mysql
127.0.0.1 azurite
```

To start:

* The controller, so you can run workflows.
* MinIO (<http://localhost:9000>, use admin/password) so you can use artifacts:

Run:

```bash
make start
```

Make sure you don't see any errors in your terminal. This runs the Workflow Controller locally on your machine (not in Docker/Kubernetes).

You can submit a workflow for testing using `kubectl`:

```bash
kubectl create -f examples/hello-world.yaml
```

We recommend running `make clean` before `make start` to ensure recompilation.

If you made changes to the executor, you need to build the image:

```bash
make argoexec-image
```

To also start the API on <http://localhost:2746>:

```bash
make start API=true
```

This runs the Argo Server (in addition to the Workflow Controller) locally on your machine.

To also start the UI on <http://localhost:8080> (`UI=true` implies `API=true`):

```bash
make start UI=true
```

![diagram](assets/make-start-UI-true.png)

If you are making change to the CLI (i.e. Argo Server), you can build it separately if you want:

```bash
make cli
./dist/argo submit examples/hello-world.yaml ;# new CLI is created as `./dist/argo`
```

Although, note that this will be built automatically if you do: `make start API=true`.

To test the workflow archive, use `PROFILE=mysql` or `PROFILE=postgres`:

```bash
make start PROFILE=mysql
```

You'll have, either:

* Postgres on <http://localhost:5432>, run `make postgres-cli` to access.
* MySQL on <http://localhost:3306>, run `make mysql-cli` to access.

To test SSO integration, use `PROFILE=sso`:

```bash
make start UI=true PROFILE=sso
```

### Running E2E tests locally

Start up Argo Workflows using the following:

```bash
make start PROFILE=mysql AUTH_MODE=client STATIC_FILES=false API=true
```

If you want to run Azure tests against a local Azurite:

```bash
kubectl -n $KUBE_NAMESPACE apply -f test/e2e/azure/deploy-azurite.yaml
make start
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
* Check the controller and argo-server logs. These are printed to the console you ran `make start` in. Is anything
  logged at `level=error`?

If tests run slowly or time out, factory reset your Kubernetes cluster.

## Committing

Before you commit code and raise a PR, always run:

```bash
make pre-commit -B
```

Please do the following when creating your PR:

* Sign-off your commits.
* Use [Conventional Commit messages](https://www.conventionalcommits.org/en/v1.0.0/).
* Suffix the issue number.

Examples:

```bash
git commit --signoff -m 'fix: Fixed broken thing. Fixes #1234'
```

```bash
git commit --signoff -m 'feat: Added a new feature. Fixes #1234'
```

## Troubleshooting

* When running `make pre-commit -B`, if you encounter errors like
  `make: *** [pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json] Error 1`, ensure that you
  have checked out your code into `$GOPATH/src/github.com/argoproj/argo-workflows`.
* If you encounter "out of heap" issues when building UI through Docker, please validate resources allocated to Docker.
  Compilation may fail if allocated RAM is less than 4Gi.

## Using Multiple Terminals

I run the controller in one terminal, and the UI in another. I like the UI: it is much faster to debug workflows than
the terminal. This allows you to make changes to the controller and re-start it, without restarting the UI (which I
think takes too long to start-up).

As a convenience, `CTRL=false` implies `UI=true`, so just run:

```bash
make start CTRL=false
```
