#!/usr/bin/env sh
set -eu

case $1 in
  init)
    kubectl delete role jenkins --ignore-not-found
    kubectl create role jenkins --verb=create,list,watch --resource=workflows.argoproj.io
    kubectl delete sa jenkins --ignore-not-found
    kubectl create sa jenkins
    kubectl delete rolebinding jenkins --ignore-not-found
    kubectl create rolebinding jenkins --role=jenkins --serviceaccount=argo:jenkins
    ;;
  get)
    SECRET=$(kubectl get sa jenkins -o=jsonpath='{.secrets[0].name}')
    ARGO_TOKEN="Bearer $(kubectl get secret $SECRET -o=jsonpath='{.data.token}' | base64 --decode)"

    curl -s http://localhost:2746/api/v1/workflows/argo -H "Authorization: $ARGO_TOKEN" > /dev/null

    echo "$ARGO_TOKEN"
    ;;
  *)
    exit 1
    ;;
esac
