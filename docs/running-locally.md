# Running Locally

## Pre-requisites

* [Go](https://golang.org/dl/) (The project currently uses version 1.13)
* [Yarn](https://classic.yarnpkg.com/en/docs/install/#mac-stable)
* [Docker](https://docs.docker.com/get-docker/)
* [Kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)
* [protoc](http://google.github.io/proto-lens/installing-protoc.html) `brew install protobuf`
* [`jq`](https://stedolan.github.io/jq/download/)
* A local Kubernetes cluster

We recommend using [K3D](https://k3d.io/) to set up the local Kubernetes cluster since this will allow you to test RBAC set-up and is fast. You can set-up K3D to be part of your default kube config as follows:

    cp ~/.kube/config ~/.kube/config.bak
    cat $(k3d kubeconfig get k3s-default) >> ~/.kube/config
    
Alternatively, you can use [Minikube](https://github.com/kubernetes/minikube) to set up the local Kubernetes cluster. Once a local Kubernetes cluster has started via `minikube start`, your kube config will use Minikube's context automatically.

Add to /etc/hosts:

    127.0.0.1 dex
    127.0.0.1 minio
    127.0.0.1 postgres
    127.0.0.1 mysql

To install into the “argo” namespace of your cluster: Argo, MinIO (for saving artifacts and logs) and Postgres (for offloading or archiving):

    make start 

If you prefer MySQL:

	make start DB=mysql

You’ll now have

* Argo on https://localhost:2746
* MinIO  http://localhost:9000 (use admin/password)

Either:

* Postgres on  http://localhost:5432, run `make postgres-cli` to access.
* MySQL on  http://localhost:3306, run `make mysql-cli` to access.

You need the token to access the CLI or UI:

    eval $(make env)

    ./dist/argo auth token

At this point you’ll have everything you need to use the CLI and UI.

## User Interface

Tip: If you want to make UI changes without a time-consuming build:

    cd ui
    yarn install
    yarn start

The UI will start up on http://localhost:8080.

## Debugging

If you want to run controller or argo-server in your IDE (e.g. so you can debug it):


Start with only components you don't want to debug;

    make start COMPONENTS=controller
    
Or

    make start COMPONENTS=argo-server
    
To find the command arguments you need to use, you’ll have to look at the `start` target in the `Makefile`.`
 
## Clean

To clean-up everything:

    make clean
    kubectl delete ns argo
    docker system prune -af
