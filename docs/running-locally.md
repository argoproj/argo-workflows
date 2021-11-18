# Running Locally

## Pre-requisites

* [Go 1.17](https://golang.org/dl/)
* [Yarn](https://classic.yarnpkg.com/en/docs/install/#mac-stable)
* [Docker](https://docs.docker.com/get-docker/)
* [protoc](http://google.github.io/proto-lens/installing-protoc.html) 
* [`jq`](https://stedolan.github.io/jq/download/)
* A local Kubernetes cluster

Code must be checked out into `$(GOPATH)/src/github.com/argoproj/argo-workflows`.

We recommend using [K3D](https://k3d.io/) to set up the local Kubernetes cluster since this will allow you to test RBAC
set-up and is fast. You can set-up K3D to be part of your default kube config as follows:

    k3d cluster start --wait

Alternatively, you can use [Minikube](https://github.com/kubernetes/minikube) to set up the local Kubernetes cluster.
Once a local Kubernetes cluster has started via `minikube start`, your kube config will use Minikube's context
automatically.

Add to /etc/hosts:

    127.0.0.1 dex
    127.0.0.1 minio
    127.0.0.1 postgres
    127.0.0.1 mysql

To install into the “argo” namespace of your cluster: Argo and MinIO (for saving artifacts and logs):

    make start

If you want the UI:

    make start UI=true

If you want MySQL for the workflow archive:

    make start PROFILE=mysql

For testing SSO integration, you can start a Argo with sso profile which will deploy a pre-configured dex instance in
argo namespace

    make start PROFILE=sso

You’ll now have:

* MinIO on http://localhost:9000 (use admin/password)
* Argo Server API on https://localhost:2746
* UI on http://localhost:8080
* Postgres on http://localhost:5432, run `make postgres-cli` to access.
* MySQL on http://localhost:3306, run `make mysql-cli` to access.

Before submitting/running workflows, build all the executor image:

    make argoexec-image

Before you commit, run:

    make pre-commit -B

Commits

* Sign-off your commits.
* Use [Conventional Commit messages](https://www.conventionalcommits.org/en/v1.0.0/).
* Suffix the issue number.

Example:

    git commit --signoff -m 'fix: Fixed broken thing. Fixes #1234'

## Troubleshooting Notes

* If you get a similar error when running one of the make pre-commit
  tests `make: *** [pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json] Error 1`, ensure you
  are working within your $GOPATH (YOUR-GOPATH/src/github.com/argoproj/argo-workflows).
* If you encounter out of heap issues when building UI through Docker, please validate resources allocated to Docker.
  Compilation may fail if allocated RAM is less than 4Gi

