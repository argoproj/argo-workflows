# How to setup your dev environment
## Pre-requisites:

* Dep. `brew install dep`
* Golang
* Yarn. `brew install yarn`
* Docker
* [Kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)
* [protoc](http://google.github.io/proto-lens/installing-protoc.html) `brew install protoc`
* `jq`
* [Swagger codegen](https://swagger.io/docs/open-source-tools/swagger-codegen/) `brew install swagger-codegen`
* Kubernetes Cluster (we recommend Docker for Desktop + K3D, as this will allow you to test RBAC set-up, and is also fast)

Useful:

* For a PS1 prompt showing your current kube context: kube-ps1 to help.  `brew install kube-ps1`
* For tailing logs: Stern. `brew install stern`

K3D tip: You can set-up K3D to be part of your default kube config as follows

    cp ~/.kube/config ~/.kube/config.bak
    cat $(k3d get-kubeconfig --name='k3s-default') >> ~/.kube/config

To install into the “argo” namespace of your cluster: Argo, MinIO (for saving artifacts and logs) and Postgres (for offloading or archiving):

    make start 

If you prefer MySQL:

	make start DB=mysql

To expose the services port forwards:

	make pf

You’ll now have

* Argo on http://localhost:2746 (see below)
* MinIO  http://localhost:9000 (use admin/password)

Either:

* Postgres on  http://localhost:5432, run `make postgres-cli` to access.
* MySQL on  http://localhost:3306, run `make mysql-cli` to access.

You need the token to access the CLI or UI:

    eval $(make env)

    ./dist/argo auth token

At this point you’ll have everything you need to use the CLI and UI.

Tip: If you want to make UI changes without a time-consuming build:

    cd ui
    yarn install
    yarn start

The UI will start up on http://localhost:8080.

If you want to run controller or argo-server in your IDE (e.g. so you can debug it):

Add to /etc/hosts:

    127.0.0.1 postgres
    127.0.0.1 mysql

Scale down the services you want to debug:

    kubectl -n argo scale deploy/workflow-controller --replicas 0
    kubectl -n argo scale deploy/argo-server --replicas 0

Restart the port forwarding:

    make pf

To find the command arguments you need to use, you’ll have to look at dist/postgres.yaml (or dist/mysql.yaml for MySQL aficionados).

## Clean

To clean-up everything:

    kubectl delete ns argo
    make clean
