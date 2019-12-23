#!/usr/bin/env bash
set -eux -o pipefail

app=argo-server
container=$(docker ps --format="{{.Names}}" | grep ${app} | head -n1 || true)

if [ "$container" = "" ]; then
  echo "cannot find container to determine KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT" >&2
  exit 1
fi

docker inspect "${container}" >&2

host=$(docker inspect "${container}" | grep -o 'KUBERNETES_SERVICE_HOST=[^"]*' | cut -c 25-)
port=$(docker inspect "${container}" | grep -o 'KUBERNETES_SERVICE_PORT=[^"]*' | cut -c 25-)

if [ "$host" = "" ]; then
  echo "host not found" >&2
  exit 1
fi

if [ "$port" = "" ]; then
  echo "port not found" >&2
  exit 1
fi

server=https://${host}:${port}
file=test/e2e/kubeconfig

cat > $file <<EOF
# Automatically created by hack/update-in-cluster-config.sh
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