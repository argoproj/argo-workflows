#!/usr/bin/env bash
set -eux -o pipefail

kubectl -n local delete wf --all

kubectl -n argo delete secret argo-kubeconfig --ignore-not-found
kubectl config view --raw --minify | sed 's/k3d-k3s-default/cluster-1/'| sed 's/namespace: .*/namespace: remote/' > argo-kubeconfig.yaml
kubectl -n argo create secret generic argo-kubeconfig --from-file=value=argo-kubeconfig.yaml

kubectl delete ns remote --ignore-not-found
kubectl create ns remote
kubectl -n remote create role remote --verb=create --resource=pods
kubectl -n remote create sa remote
kubectl -n remote create rolebinding remote --role=remote --serviceaccount=remote:remote
kubectl -n remote apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/minio/my-minio-cred-secret.yaml
kubectl -n remote apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/workflow-role.yaml
kubectl -n remote create sa workflow
kubectl -n remote create rolebinding workflow --role=workflow-role --serviceaccount=remote:workflow

kubectl delete ns local --ignore-not-found
kubectl create ns local
SECRET=$(kubectl -n remote get sa remote -o=jsonpath='{.secrets[0].name}')
TOKEN=$(kubectl get -n remote secret $SECRET -o=jsonpath='{.data.token}' | base64 --decode)

sed "s/TOKEN/$TOKEN/" > workflow-kubeconfig.yaml <<END
apiVersion: v1
contexts:
  - context:
      cluster: cluster-1
      namespace: remote
      user: cluster-1
    name: cluster-1
current-context: cluster-1
kind: Config
preferences: { }
users:
  - name: cluster-1
    user:
      token: TOKEN
END

kubectl -n local create secret generic workflow-kubeconfig --from-file=value=workflow-kubeconfig.yaml

kubectl -n local apply -f ../examples/multi-cluster/multi-cluster-workflow.yaml

kubectl -n local wait wf/multi-cluster --for=condition=Completed
