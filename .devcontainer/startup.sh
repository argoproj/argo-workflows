#!/bin/bash
set -e

sudo apt update
sudo chown root:docker /var/run/docker.sock
sudo chown -R vscode:golang /go/src/

echo '127.0.0.1 dex\n127.0.0.1 minio\n127.0.0.1 postgres\n127.0.0.1 mysql' | sudo tee -a /etc/hosts

if k3d cluster list | grep k3s-default;
then
   echo "skip k3s creation, k3s-default cluster already exist"
else
    k3d cluster create
fi
