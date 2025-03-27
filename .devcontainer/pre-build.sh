#!/usr/bin/env bash
set -eux

# install kubernetes using the minimum tested version
. hack/k8s-versions.sh
wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
k3d cluster get k3s-default || k3d cluster create --image "rancher/k3s:${K8S_VERSIONS[min]}-k3s1" --wait
k3d kubeconfig merge --kubeconfig-merge-default

# install kubectl
curl -LO "https://dl.k8s.io/release/${K8S_VERSIONS[min]}/bin/linux/$(go env GOARCH)/kubectl"
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
kubectl cluster-info

# install protocol buffer compiler (protoc)
sudo apt update
sudo apt install -y protobuf-compiler

# Make sure go path is owned by vscode
sudo chown vscode:vscode /home/vscode/go || true
sudo chown vscode:vscode /home/vscode/go/src || true
sudo chown vscode:vscode /home/vscode/go/src/github.com || true

# install kit
make kit

# download dependencies and do first-pass compile
kit build

# Patch CoreDNS to have host.docker.internal inside the cluster available
kubectl get cm coredns -n kube-system -o yaml | sed "s/  NodeHosts: |/  NodeHosts: |\n    `grep host.docker.internal /etc/hosts`/" | kubectl apply -f -
