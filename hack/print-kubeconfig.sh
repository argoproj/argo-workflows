#!/usr/bin/env sh
set -eu
# This is a utility script that creates a KUBECONFIG based on a service account in another cluster.

context=${1:-$(kubectl config current-context)}
namespace=${2:-default}
serviceAccount=${3:-default}

server=$(kubectl config view --context $context --minify --raw -o jsonpath='{.clusters[0].cluster.server}')
secretName=$(kubectl --context $context -n $namespace get sa $serviceAccount -o jsonpath='{.secrets[0].name}')
ca=$(kubectl --context $context -n $namespace get secret $secretName -o jsonpath='{.data.ca\.crt}')
token=$(kubectl --context $context -n $namespace get secret $secretName -o jsonpath='{.data.token}' | base64 --decode)

# keep on one line to make it work with kubectl create secret ... --from-literal
cat <<EOF
{"apiVersion":"v1","kind":"Config","clusters":[{"name":"default","cluster":{"certificate-authority-data":"${ca}","server":"${server}"}}],"contexts":[{"name":"default","context":{"cluster":"default","namespace":"${namespace}","user":"${serviceAccount}"}}],"users":[{"name":"${serviceAccount}","user":{"token":"${token}"}}],"current-context":"default"}
EOF