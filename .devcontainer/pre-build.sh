#!/usr/bin/env sh
set -eux

# install kubernetes
wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
k3d cluster get k3s-default || k3d cluster create --image rancher/k3s:v1.27.3-k3s1 --wait
k3d kubeconfig merge --kubeconfig-merge-default

# install kubectl
curl -LO https://dl.k8s.io/release/v1.27.3/bin/linux/$(go env GOARCH)/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
kubectl cluster-info

# install kit
curl -q https://raw.githubusercontent.com/kitproj/kit/main/install.sh | sh

# install protocol buffer compiler (protoc)
sudo apt update
sudo apt install -y protobuf-compiler

# Make sure go path is owned by vscode
sudo chown -R vscode:vscode /home/vscode/go

# download dependencies and do first-pass compile
CI=1 kit pre-up

# Patch CoreDNS to have host.docker.internal inside the cluster available
kubectl get cm coredns -n kube-system -o yaml | sed "s/  NodeHosts: |/  NodeHosts: |\n    `grep host.docker.internal /etc/hosts`/" | kubectl apply -f -
