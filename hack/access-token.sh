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
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: jenkins.service-account-token
  annotations:
    kubernetes.io/service-account.name: jenkins
type: kubernetes.io/service-account-token
EOF
    ;;
  get)
    while true; do
      TOKEN=$(kubectl get secret jenkins.service-account-token -o=jsonpath='{.data.token}' | base64 --decode)
      if [ "$TOKEN" != "" ]; then
        echo "Bearer $TOKEN"
        exit
      fi
      sleep 1
    done
    ;;
  *)
    exit 1
    ;;
esac
