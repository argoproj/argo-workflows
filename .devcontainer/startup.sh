#!/bin/bash
set -e

sudo apt update
sudo chown $USER:docker /var/run/docker.sock
sudo chown -fR $USER:golang $GOPATH

echo '127.0.0.1 dex\n127.0.0.1 minio\n127.0.0.1 postgres\n127.0.0.1 mysql\n127.0.0.1 azurite' | sudo tee -a /etc/hosts

if k3d cluster list | grep k3s-default;
then
   echo "skip k3s creation, k3s-default cluster already exist"
else
    k3d cluster create
fi

until k3d cluster start --wait ; do sleep 5 ; done
k3d kubeconfig merge k3s-default --kubeconfig-merge-default --kubeconfig-switch-context
