# Running Locally

## Requirements

* [Go 1.17](https://golang.org/dl/)
* [Yarn](https://classic.yarnpkg.com/en/docs/install/#mac-stable)
* [Docker](https://docs.docker.com/get-docker/)
* [protoc](http://google.github.io/proto-lens/installing-protoc.html) 
* [jq](https://stedolan.github.io/jq/download/)
* A local Kubernetes cluster (`k3d`, `kind`, or `minikube`)

We recommend using [K3D](https://k3d.io/) to set up the local Kubernetes cluster since this will allow you to test RBAC
set-up and is fast. You can set-up K3D to be part of your default kube config as follows:

```shell
k3d cluster start --wait
```

Alternatively, you can use [Minikube](https://github.com/kubernetes/minikube) to set up the local Kubernetes cluster.
Once a local Kubernetes cluster has started via `minikube start`, your kube config will use Minikube's context
automatically.

## Developing locally

!!! Warning
    The git repo must be checked out into: `$(GOPATH)/src/github.com/argoproj/argo-workflows`

Add the following to your `/etc/hosts`:

```
127.0.0.1 dex
127.0.0.1 minio
127.0.0.1 postgres
127.0.0.1 mysql
```

To run the controller and argo-server API locally, with MinIO inside the "argo" namespace of your cluster:

```shell
make start API=true
```
    
To start the UI, use `UI=true`:

```shell
make start API=true UI=true
```

To test the workflow archive, use `PROFILE=mysql`:

```shell
make start API=true UI=true PROFILE=mysql
```
    
To test SSO integration, use `PROFILE=sso`:

```shell
make start API=true UI=true PROFILE=sso
```

You’ll now have:

* Argo UI on http://localhost:8080
* Argo Server API on https://localhost:2746
* MinIO on http://localhost:9000 (use admin/password)
* Postgres on http://localhost:5432, run `make postgres-cli` to access.
* MySQL on http://localhost:3306, run `make mysql-cli` to access.

Before submitting/running workflows, build the executor images with this command:

```shell
make argoexec-image
```

### Running E2E tests locally

1. Configure your IDE to set the `KUBECONFIG` environment variable to your k3d kubeconfig file
2. Find an e2e test that you want to run in `test/e2e`
3. Determine which profile the e2e test is using by inspecting the go build flag at the top of the file and referring to [ci-build.yaml](https://github.com/argoproj/argo-workflows/blob/master/.github/workflows/ci-build.yaml)

    For example `TestArchiveStrategies` in `test/e2e/functional_test.go` has the following build flags

    ```go
    //go:build functional
    // +build functional
    ```

    In [ci-build.yaml](https://github.com/argoproj/argo-workflows/blob/master/.github/workflows/ci-build.yaml) the functional test suite is using the `minimal` profile

4. Run the profile in a terminal window

    ```shell
    make start PROFILE=minimal E2E_EXECUTOR=emissary AUTH_MODE=client STATIC_FILES=false LOG_LEVEL=info API=true UI=false
    ```

5. Run the test in your IDE

## Committing

Before you commit code and raise a PR, always run:

```shell
make pre-commit -B
```

Please adhere to the following when creating your commits:

* Sign-off your commits.
* Use [Conventional Commit messages](https://www.conventionalcommits.org/en/v1.0.0/).
* Suffix the issue number.

Example:

```shell
git commit --signoff -m 'fix: Fixed broken thing. Fixes #1234'
```

Troubleshooting:

* When running `make pre-commit -B`, if you encounter errors like
  `make: *** [pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json] Error 1`,
  ensure that you have checked out your code into `$(GOPATH)/src/github.com/argoproj/argo-workflows`.
* If you encounter "out of heap" issues when building UI through Docker, please validate resources allocated to Docker.
  Compilation may fail if allocated RAM is less than 4Gi.
