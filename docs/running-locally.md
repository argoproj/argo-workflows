# Running Locally
## Pre-requisites:

* Golang
* Yarn. `brew install yarn` 
* Docker
* [Kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)
* [protoc](http://google.github.io/proto-lens/installing-protoc.html) `brew install protobuf`
* `jq`
* Kubernetes Cluster (we recommend Docker for Desktop + K3D, as this will allow you to test RBAC set-up, and is also fast)

Useful:

* For a PS1 prompt showing your current kube context: kube-ps1 to help.  `brew install kube-ps1`

K3D tip: You can set-up K3D to be part of your default kube config as follows

    cp ~/.kube/config ~/.kube/config.bak
    cat $(k3d get-kubeconfig --name='k3s-default') >> ~/.kube/config

Add to /etc/hosts:

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

### Running Sonar Locally

This can only be done if you have already created a pull request.

Install the scanner:

```
brew install sonar-scanner
```

Run the tests:

```
make test CI=true
make test-reports/test-report.out
```

Perform a scan:

```
# the key is PR number (e.g. "2666"), the branch is the CI branch, e.g. "pull/2666"
SONAR_TOKEN=... sonar-scanner -Dsonar.pullrequest.key=... -Dsonar.pullrequest.branch=... 
```
 
## Clean

To clean-up everything:

    make clean
    kubectl delete ns argo
    docker system prune -af
