#!/usr/bin/env bash
set -eu

app=argo-server
container=$(docker ps --format="{{.Names}}" | grep ${app})

host=$(docker inspect ${container} | grep -o 'KUBERNETES_SERVICE_HOST=[^"]*' | cut -c 25-)
port=$(docker inspect ${container} | grep -o 'KUBERNETES_SERVICE_PORT=[^"]*' | cut -c 25-)

server=https://${host}:${port}
file=test/e2e/kubeconfig.$(whoami)

cat > $file <<EOF
# Automatically created by hack/export-in-cluster-config.sh
apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: ${server}
    certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
users:
- name: service-account
  user:
    tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
contexts:
- context:
    cluster: local
    user: service-account
EOF

echo "created/updated $file"