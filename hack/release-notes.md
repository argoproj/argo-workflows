# Quick Start

## What's New?

Find out on [our blog](https://blog.argoproj.io) and [changelog](https://github.com/argoproj/argo-workflows/blob/main/CHANGELOG.md).

## Breaking Changes and Known Issues

Check the [upgrading guide](https://argo-workflows.readthedocs.io/en/release-3.5/upgrading/) and search for [existing issues on GitHub](https://github.com/argoproj/argo-workflows/issues).

## Installation

### CLI

#### Mac / Linux

Available via `curl`

```bash
# Detect OS
ARGO_OS="darwin"
if [[ "$(uname -s)" != "Darwin" ]]; then
  ARGO_OS="linux"
fi

# Download the binary
curl -sLO "https://github.com/argoproj/argo-workflows/releases/download/$version/argo-$ARGO_OS-amd64.gz"

# Unzip
gunzip "argo-$ARGO_OS-amd64.gz"

# Make binary executable
chmod +x "argo-$ARGO_OS-amd64"

# Move binary to path
mv "./argo-$ARGO_OS-amd64" /usr/local/bin/argo

# Test installation
argo version
```

### Controller and Server

```bash
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/$version/install.yaml
```
