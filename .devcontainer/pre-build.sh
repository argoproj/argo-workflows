#!/usr/bin/env bash
set -eux

# install Tilt (used by `make start`) into $GOPATH/bin, which is on PATH
make tilt

# create cluster using the minimum tested Kubernetes version (k3d-up.sh also
# applies the kubelet config that stops images being GC'd during test runs)
K8S_VERSION=min ./hack/tilt/k3d-up.sh

# Make sure go path is owned by vscode
sudo chown vscode:vscode /home/vscode/go || true
sudo chown vscode:vscode /home/vscode/go/src || true
sudo chown vscode:vscode /home/vscode/go/src/github.com || true

# Patch CoreDNS to have host.docker.internal inside the cluster available
kubectl get cm coredns -n kube-system -o yaml | sed "s/  NodeHosts: |/  NodeHosts: |\n    `grep host.docker.internal /etc/hosts`/" | kubectl apply -f -
