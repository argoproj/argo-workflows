# Release Instructions

Allow 1h to do a release.

## Preparation

Cherry-pick your changes from master onto the release branch.

Mandatory: the release branch must be green [in CircleCI](https://app.circleci.com/github/argoproj/argo/pipelines).

It is a very good idea to clean up before you start:

    make clean
    kubectl delete ns argo

## Release

To generate new manifests and perform basic checks:

    make prepare-release VERSION=v2.5.0-rc6

Next, build everything:

    make build

Publish the images and local Git changes:

    make publish-release

Create [the release](https://github.com/argoproj/argo/releases) in Github. You can get some text for this using [Github Toolkit](https://github.com/alexec/github-toolkit):

    ght relnote v2.5.0-rc5..v2.5.0-rc6

    
## Validation

K3D tip: you'll need to import the images:

    k3d import-images argoproj/argocli:v2.5.0-rc6 argoproj/argoexec:v2.5.0-rc6 argoproj/workflow-controller:v2.5.0-rc6

Install Argo locally:

    kubectl create ns argo
    kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/v2.5.0-rc6/manifests/quick-start-postgres.yaml
    make pf-bg 

Maybe run e2e tests?

    make test-e2e
