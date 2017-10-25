#!/bin/bash

# Shell script to process a kubeconfig and embed the certificate and token
# for the purposes of an argo install

set -e

kubeconfig=$1
output_config=$2

if [[ -z "${kubeconfig}" || -z "${output_config}" ]] ; then
    echo "Usage ${0} ~/.kube/config ~/.kube/processed_config"
    exit 1
fi

# read necessary information about current context
current_context=`kubectl --kubeconfig=$1 config current-context`
token=`kubectl --kubeconfig=$1 get secrets --namespace default -o custom-columns=:data.token | base64 --decode`
server_ip=`kubectl --kubeconfig=$1 config view -o jsonpath="{.clusters[?(@.name == \"${current_context}\")].cluster.server}"`
temp_crt_file="/tmp/${current_context}_ca.crt"
rm -f ${temp_crt_file}
kubectl --kubeconfig=$1 get secrets --namespace default -o custom-columns=:data."ca\.crt" | base64 --decode > ${temp_crt_file}

# write the new kubeconfig
kubectl config --kubeconfig=$2 set-cluster ${current_context} --server=${server_ip} --embed-certs=true --certificate-authority=${temp_crt_file}
kubectl config --kubeconfig=$2 --server=${server_ip} set-credentials ${current_context} --token ${token}
kubectl config --kubeconfig=$2 --server=${server_ip} set-context --cluster ${current_context} --user ${current_context} ${current_context}
kubectl config --kubeconfig=$2 --server=${server_ip} use-context ${current_context}
kubectl config --kubeconfig=$2 --server=${server_ip} set-cluster ${current_context}

rm -f ${temp_crt_file}
