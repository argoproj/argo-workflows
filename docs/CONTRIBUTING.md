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

Requirements:

* Golang 1.11
* Docker
* Dep v0.5
   * Mac Install: `brew install dep`
* Yarn
* Kubernetes locally installed.

Then you can install the development version into the cluster in your kube config:

```
$ go get github.com/argoproj/argo
$ cd $(go env GOPATH)/src/github.com/argoproj/argo
$ make start
```

### E2E Testing

See [test/e2e/README.md](../test/e2e/README.md).
