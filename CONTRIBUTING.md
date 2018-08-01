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

Go to https://groups.google.com/forum/#!forum/argoproj
* Create a new topic to discuss your feature.

## How to setup your dev environment

### Requirements
* Golang 1.10
* Docker
* dep
   * Mac Install: `brew install dep`
   * Mac/Linux Install: `go get -u github.com/golang/dep/cmd/dep`

### Quickstart
```
$ go get github.com/argoproj/argo
$ cd $(go env GOPATH)/src/github.com/argoproj/argo
$ dep ensure -vendor-only
$ make
```

### Build workflow-controller and executor images
The following will build the workflow-controller and executor images tagged with the `latest` tag, then push to a personal dockerhub repository:
```
$ make controller-image executor-image IMAGE_TAG=latest IMAGE_NAMESPACE=jessesuen DOCKER_PUSH=true
```

### Build argo cli
```
$ make cli
$ ./dist/argo version
```

### Deploying controller with alternative controller/executor images
```
$ argo install --controller-image jessesuen/workflow-controller:latest --executor-image jessesuen/argoexec:latest
```

## Most needed contributions

* TBD
