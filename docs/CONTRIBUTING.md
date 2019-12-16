# How to contribute (Work in progress)

## How to report a bug

Open an issue at https://github.com/argoproj/
* What did you do? (how to reproduce)
* What did you see? (include logs and screenshots as appropriate)
* What did you expect?

## How to contribute a bug fix

Go to https://github.com/argoproj/
* Open an issue and discuss it.
* Create a pull request for your fix.

## How to suggest a new feature

Go to https://github.com/argoproj/
* Open an issue and discuss it.

## How to setup your dev environment

### Requirements
* Golang 1.11
* Docker
* dep v0.5
   * Mac Install: `brew install dep`
* golangci-lint v1.16.0

### Quickstart
```
$ go get github.com/argoproj/argo
$ cd $(go env GOPATH)/src/github.com/argoproj/argo
$ dep ensure -vendor-only
$ make
```

### Build workflow-controller and executor images
The following will build the release versions of workflow-controller and executor images tagged
with the `latest` tag, then push to a personal dockerhub repository, `mydockerrepo`:
```
$ make controller-image executor-image IMAGE_TAG=latest IMAGE_NAMESPACE=mydockerrepo DOCKER_PUSH=true
```
Building release versions of the images will be slow during development, since the build happens
inside a docker build context, which cannot re-use the golang build cache between builds. To build
images quicker (for development purposes), images can be built by adding DEV_IMAGE=true.
```
$ make controller-image executor-image IMAGE_TAG=latest IMAGE_NAMESPACE=mydockerrepo DOCKER_PUSH=true DEV_IMAGE=true
```

### Build argo cli
```
$ make cli
$ ./dist/argo version
```

### Deploying controller with alternative controller/executor images
```
$ helm install argo/argo --set images.namespace=mydockerrepo --set
images.controller workflow-controller:latest
```
