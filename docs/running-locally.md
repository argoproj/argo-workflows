# Running Locally

## Pre-requisites

* [Go](https://golang.org/dl/) (The project currently uses version 1.15)
* [Yarn](https://classic.yarnpkg.com/en/docs/install/#mac-stable)
* [Docker](https://docs.docker.com/get-docker/)
* [Kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/)
* [protoc](http://google.github.io/proto-lens/installing-protoc.html) `brew install protobuf`
* [`jq`](https://stedolan.github.io/jq/download/)
* A local Kubernetes cluster

### 1. Set up a local cluster

We recommend using [K3D](https://k3d.io/) to set up the local Kubernetes cluster since this will allow you to test RBAC set-up and is fast. You can set-up K3D to be part of your default kube config as follows:

    k3d cluster start --wait
    
Alternatively, you can use [Minikube](https://github.com/kubernetes/minikube) to set up the local Kubernetes cluster. Once a local Kubernetes cluster has started via `minikube start`, your kube config will use Minikube's context automatically.


### 2. Set up local service aliases

Add to /etc/hosts:

    127.0.0.1 dex
    127.0.0.1 minio
    127.0.0.1 postgres
    127.0.0.1 mysql

### 3. Install Argo and start the controller

To install into the “argo” namespace of your cluster: Argo and MinIO (for saving artifacts and logs):

    make start 

### 4. (Optional) Set up a DB for the Workflow archive

If you want MySQL for the workflow archive:

    make start PROFILE=mysql

### 5. Check out the Argo services running locally

You’ll now have

* Argo Server API on https://localhost:2746
* UI on http://localhost:8080
* MinIO  http://localhost:9000 (use admin/password)

Either:

* Postgres on  http://localhost:5432, run `make postgres-cli` to access.
* MySQL on  http://localhost:3306, run `make mysql-cli` to access.

At this point you’ll have everything you need to use the CLI and UI.


### 6. Build Argo images

Before submitting/running workflows, build all Argo images, so they're available for the workflow.

    make build

## Troubleshooting Notes

If you get a similar error when running one of the make pre-commit tests `make: *** [pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json] Error 1`, ensure you are working within your $GOPATH (YOUR-GOPATH/src/github.com/argoproj/argo-workflows).

## Clean

To clean-up everything:

    make clean
    kubectl delete ns argo
    docker system prune -af
