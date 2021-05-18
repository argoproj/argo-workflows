# Quick Start

## What's New?

Find out on [our blog](https://blog.argoproj.io).

## Argo CLI

### Mac

Available via `curl`

```sh
# Download the binary
curl -sLO https://github.com/argoproj/argo/releases/download/${version}/argo-darwin-amd64.gz

# Unzip
gunzip argo-darwin-amd64.gz

# Make binary executable
chmod +x argo-darwin-amd64

# Move binary to path
mv ./argo-darwin-amd64 /usr/local/bin/argo

# Test installation
argo version
```

### Linux

Available via `curl`

```sh
# Download the binary
curl -sLO https://github.com/argoproj/argo/releases/download/${version}/argo-linux-amd64.gz

# Unzip
gunzip argo-linux-amd64.gz

# Make binary executable
chmod +x argo-linux-amd64

# Move binary to path
mv ./argo-linux-amd64 /usr/local/bin/argo

# Test installation
argo version
```

## Install 

```sh
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/${version}/install.yaml
```
