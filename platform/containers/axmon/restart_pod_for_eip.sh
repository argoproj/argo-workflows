#!/bin/bash

set -x

echo "Setting hostname to $1"

CLUSTER_DNS_NAME=$1
NAMESPACE=axsys
KUBECTL=kubectl
DEPLOYS="axops-deployment gateway-deployment"
CONFIGMAP=cluster-dns-name

$KUBECTL delete configmap $CONFIGMAP --namespace $NAMESPACE
while $KUBECTL get configmap $CONFIGMAP --namespace $NAMESPACE ; do
    sleep 1
done
$KUBECTL create configmap $CONFIGMAP --namespace $NAMESPACE --from-literal=cluster-external-dns-name=$CLUSTER_DNS_NAME
while ! $KUBECTL get configmap $CONFIGMAP --namespace $NAMESPACE ; do
    sleep 1
done

for deploy in $DEPLOYS ; do
    config_file=`mktemp`
    $KUBECTL get deployment $deploy -o yaml --namespace $NAMESPACE > $config_file || continue
    old_name=`$KUBECTL get deployment $deploy -o yaml --namespace $NAMESPACE | grep -C 1 AXOPS_EXT_DNS | tail -1 | awk '{print $2}'`
    [[ "$CLUSTER_DNS_NAME" = "$old_name" ]] && continue
    $KUBECTL delete deployment $deploy --namespace $NAMESPACE
    while $KUBECTL get po -l "app=$deploy" --namespace $NAMESPACE | grep -q NAME ; do
        sleep 1
    done
    cat $config_file | sed "s/value: $old_name/value: $CLUSTER_DNS_NAME/g" | $KUBECTL create --namespace $NAMESPACE -f -
    while ! $KUBECTL get po -l "app=$deploy" --namespace $NAMESPACE | grep -q NAME ; do
        sleep 1
    done
    rm $config_file
done
