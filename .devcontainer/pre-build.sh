#!/usr/bin/env bash
set -eux

# create cluster using the minimum tested Kubernetes version
. hack/k8s-versions.sh
k3d cluster get k3s-default || k3d cluster create --image "rancher/k3s:${K8S_VERSIONS[min]}-k3s1" --wait
k3d kubeconfig merge --kubeconfig-merge-default
kubectl cluster-info

# Make sure go path is owned by vscode
sudo chown vscode:vscode /home/vscode/go || true
sudo chown vscode:vscode /home/vscode/go/src || true
sudo chown vscode:vscode /home/vscode/go/src/github.com || true

# Patch CoreDNS to have host.docker.internal inside the cluster available
kubectl get cm coredns -n kube-system -o yaml | sed "s/  NodeHosts: |/  NodeHosts: |\n    `grep host.docker.internal /etc/hosts`/" | kubectl apply -f -
