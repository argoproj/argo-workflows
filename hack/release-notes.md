# Quick Start

## What's New?

Find out on [our blog](https://blog.argoproj.io) and [changelog](https://github.com/argoproj/argo-workflows/blob/main/CHANGELOG.md).

## Breaking Changes and Known Issues

Check the [upgrading guide](https://argo-workflows.readthedocs.io/en/latest/upgrading/) and search for [existing issues on GitHub](https://github.com/argoproj/argo-workflows/issues).

## Installation

### CLI

#### Mac / Linux

Available via `curl`

```bash
IS_MAC=$([[ $(uname -s) == "Darwin" ]] && echo "darwin")  # if it's a mac, sets the variable as darwin
ARGO_OS=${IS_MAC:-"linux"}  # if IS_MAC is not set, defaults to linux
ARGO_ARCH=$(uname -m)

# Download the binary
curl -sLO "https://github.com/argoproj/argo-workflows/releases/download/$version/argo-$ARGO_OS-$ARGO_ARCH.gz"

# Unzip
gunzip "argo-$ARGO_OS-$ARGO_ARCH.gz"

# Make binary executable
chmod +x "argo-$ARGO_OS-$ARGO_ARCH"

# Move binary to path
mv "./argo-$ARGO_OS-$ARGO_ARCH" /usr/local/bin/argo

# Test installation
argo version
```

### Controller and Server

```bash
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/$version/install.yaml
```
