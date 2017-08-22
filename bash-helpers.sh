#!/usr/bin/env bash
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

#
# Source this file in your .bash_profile or .bashrc
# For even cooler bash prompts source bash-powerline.sh
#

# Set AX_NAMESPACE and AX_VERSION
# Usage: ksetv image-namespace image-version
ksetv()
{
    export AX_NAMESPACE=$1
    export AX_VERSION=$2
}

# Unset AX_NAMESPACE and AX_VERSION env, and kmanager will use default values
kunsetv()
{
    unset AX_NAMESPACE
    unset AX_VERSION
}

kcluster()
{
    kmanager bash
}

kctl()
{
    CLUSTER=$1
    NAMESPACE=$2
    shift 2
    COMMAND=$@
    eval ${KUBECTL} --kubeconfig=${HOME}/.kube/cluster_${CLUSTER}.conf --namespace ${NAMESPACE} ${COMMAND}
}

kdef()
{
    if [ -z "$1" ]
    then
        echo "Usage $0 clustername-prefix [namespace]"
        return 1
    fi

    if [[ -z "$(which kubectl-1.5.7)" || -z "$(which kubectl-1.6.7)" ]]
    then
        echo
        echo "You need to install two versions of kubectl client, 1.5.7 and 1.6.7."
        echo
    fi

    cluster=`ls -rt $HOME/.kube/cluster_$1* 2>/dev/null | tail -n 1 | sed 's#.*cluster_##g' | sed 's#\.conf##g'`
    if [ -z "$cluster" ]
    then
        echo "Cluster with prefix $1 not found. Assuming $1 is the full name."
        cluster=$1
    fi
    export DEFAULT_KCLUSTER=$cluster
    if [ -z "$2" ]
    then
        export DEFAULT_NAMESPACE=axsys
    else
        export DEFAULT_NAMESPACE=$2
    fi

    if grep -q "name: gcp" ${HOME}/.kube/cluster_${DEFAULT_KCLUSTER}.conf ; then
        echo "Target cluster ${DEFAULT_KCLUSTER} in GCP cloud."
        export KUBECTL=kubectl-1.5.7
    else
        echo "Target cluster ${DEFAULT_KCLUSTER} in AWS cloud."
        server=`kubectl-1.5.7 --kubeconfig=${HOME}/.kube/cluster_${DEFAULT_KCLUSTER}.conf version | grep -i server | cut -d ":" -f 5 2> /dev/null`
        if [[ $server == *"v1.6"* ]]; then
            export KUBECTL=kubectl-1.6.7
        else
            echo "Using default kubectl version 1.6.7"
            export KUBECTL=kubectl-1.6.7
        fi
    fi
}

kundef()
{
    unset DEFAULT_NAMESPACE
    unset DEFAULT_KCLUSTER
}

k()
{
    if [ -z "$DEFAULT_KCLUSTER" -o -z "$DEFAULT_NAMESPACE" ]
    then
        echo "Usage: Set default cluster using kdef command"
        return 1
    fi
    kctl $DEFAULT_KCLUSTER $DEFAULT_NAMESPACE $@
}

kp()
{
    k "get" "pods" $@
}

kdesc()
{
    k "describe" "pods" $@
}

kdp()
{
    pod=$1
    shift
    k "delete" "pod" $pod $@
}

kddp()
{
    deployment=$1
    shift
    k "delete" "deployment" $deployment $@
}

kdds()
{
    daemonset=$1
    shift
    k "delete" "daemonset" $daemonset $@
}

kn()
{
    k "get" "nodes" $@
}

ks()
{
    k "get" "svc" $@
}

kl()
{
    k "logs" $@
}

kj()
{
    k "get" "jobs" $@
}

kssh()
{
    if [ -z "$1" ]
    then 
        echo "Usage: $0 nodename from kn command"
        return 1
    fi
    NODE=$1
    if [ "${NODE:0:4}" = "gke-" ] ; then
        shift
        if [ -z "$*" ] ; then
            gcloud compute ssh $NODE
        else
            gcloud compute ssh $NODE --command "$*"
        fi
    else
        IP=`kn $NODE -o jsonpath="'{.status.addresses[2].address}'"`
        shift
        ssh -i $HOME/.ssh/kube_id_${CLUSTER} admin@$IP $@
    fi
}

km()
{
    ssh -o StrictHostKeyChecking=no -i $HOME/.ssh/kube_id_${DEFAULT_KCLUSTER} admin@$(grep "server:" $HOME/.kube/cluster_$DEFAULT_KCLUSTER.conf | sed 's#.*//##g') "$@"
}

kmdownload()
{
    scp -o StrictHostKeyChecking=no -i $HOME/.ssh/kube_id_${DEFAULT_KCLUSTER} admin@$(grep "server:" $HOME/.kube/cluster_$DEFAULT_KCLUSTER.conf | sed 's#.*//##g'):$1 $2
}

kmupload()
{
    temp=`mktemp`
    scp -o StrictHostKeyChecking=no -i $HOME/.ssh/kube_id_${DEFAULT_KCLUSTER} $1 admin@$(grep "server:" $HOME/.kube/cluster_$DEFAULT_KCLUSTER.conf | sed 's#.*//##g'):$temp
    ssh -o StrictHostKeyChecking=no -i $HOME/.ssh/kube_id_${DEFAULT_KCLUSTER} admin@$(grep "server:" $HOME/.kube/cluster_$DEFAULT_KCLUSTER.conf | sed 's#.*//##g') sudo mv $temp $2
}

kshell()
{
    if [ \( "$1" = "" \) -o \( "$2" = "" \) ]; then
        echo "Usage: kshell <pod> <shell> [<containername>]"
        return 1
    fi

    CONTAINERSHELL=""
    if [ "$3" != "" ]; then
        CONTAINERSHELL=" -c $3"
    fi
    COLUMNS=`tput cols`
    LINES=`tput lines`
    TERM=xterm
    k "exec" "-i" "-t" "$1" "env" "COLUMNS=$COLUMNS" "LINES=$LINES" "TERM=$TERM" "$2" "$CONTAINERSHELL"
}

kns()
{
    if [ -z "$1" ]
    then
        echo "Usage: kns <name-of-namespace>"
        echo "Current namespaces:"
        k get namespaces
        return 1
    fi
    local all_namespaces=`$KUBECTL --kubeconfig=${HOME}/.kube/cluster_${DEFAULT_KCLUSTER}.conf get namespaces | grep -v NAME | cut -d " " -f 1| tr '\r\n' ' '`
    if [[ ! " ${all_namespaces[@]} " =~ " $1 " ]]; then
        echo "No namespace named $1"
        return 1
    fi
    unset DEFAULT_NAMESPACE
    export DEFAULT_NAMESPACE=$1
}

kpassword()
{
    CLUSTER=$1
    if [ -z "$1" ]
    then
        CLUSTER=${DEFAULT_KCLUSTER}
    fi
    echo "ClusterID: ${CLUSTER}"
    echo "Access info in ~/.argo"
}

kui()
{
    # Open the Argo cluster UI in the default browser.
    elb=`k "get svc axops --namespace=axsys -o wide" | grep elb | cut  -d " " -f 9`
    python -m webbrowser "https://$elb"
}

kpf()
{
    if [ -z "$1" ]
    then
        echo "Usage: kpf pod-name-prefix"
        return 1
    fi
    pod=`kp | grep $1 | cut -d " " -f 1`
    echo "Using pod: " $pod
    k port-forward $pod $2
}

kall()
{
    ips=`kn | cut -d " " -f 1 | grep -v NAME`
    for ip in $ips; do
        echo "$(tput setaf 3)$ip $(tput setaf 7)"
        kssh $ip "$@"
    done
}

fdp()
{
    if [ -z "$1" ]
    then
        echo "Usage: fdp pod-name-prefix"
        return 1
    fi
    pod=`kp | grep $1 | cut -d " " -f 1`
    kdp $pod
}

fl()
{
    if [ -z "$1" ]
    then
        echo "Usage: fl pod-name-prefix [-f]"
        return 1
    fi
    pod=`kp | grep $1 | cut -d " " -f 1`
    kl $pod "$@"
}

export AWS_OUTPUT_FORMAT=table
export ACMD_AWS_PROFILE=default

aformat()
{
    export AWS_OUTPUT_FORMAT=$1
    echo "Output format for a commands set to $1"
}

aprofile()
{
    export ACMD_AWS_PROFILE=$1
    echo "Using aws profile $ACMD_AWS_PROFILE"
}

an()
{
    if [ -z "$DEFAULT_KCLUSTER" ]; then
        echo "Lists all ec2 instances in given k8s cluster. Run kdef first"
        return 1
    fi
    aws --profile $ACMD_AWS_PROFILE ec2 describe-instances --output $AWS_OUTPUT_FORMAT --filters Name=tag:Name,Values="$DEFAULT_KCLUSTER*" --query 'Reservations[].Instances[].[Tags[?Key==`Name`] | [0].Value, InstanceId, State.Name, PublicIpAddress, PrivateDnsName, InstanceLifecycle]'
}

atags()
{
    if [ -z "$1" ]; then
        echo "Lists all tags of given EC2 instance. Usage $FUNCNAME instance-id"
        return 1
    fi
    aws --profile $ACMD_AWS_PROFILE ec2 describe-tags --filters "Name=resource-id,Values=$1" --output $AWS_OUTPUT_FORMAT
}

avs()
{
    if [ -z "$DEFAULT_KCLUSTER" ]; then
        echo "Lists all volumes in given k8s cluster. Run kdef first"
        return 1
    fi
    aws --profile $ACMD_AWS_PROFILE ec2 describe-volumes --output $AWS_OUTPUT_FORMAT --filters Name=tag:Name,Values="$DEFAULT_KCLUSTER*" --query 'Volumes[].[Tags[?Key==`Name`] | [0].Value, VolumeId, AvailabilityZone, Size, Attachments[0].InstanceId]'
}

avtags()
{
    if [ -z "$1" ]; then
        echo "Lists all tags of given EBS volume. Usage $FUNCNAME volume-id"
        return 1
    fi
    aws --profile $ACMD_AWS_PROFILE ec2 describe-tags --filters "Name=resource-type,Values=volume,Name=resource-id,Values=$1" --output $AWS_OUTPUT_FORMAT
}

avpcs()
{
    # Lists all VPCs
    aws --profile $ACMD_AWS_PROFILE ec2 describe-vpcs --query 'Vpcs[].[Tags[?Key==`KubernetesCluster`] | [0].Value, VpcId, CidrBlock]' --output $AWS_OUTPUT_FORMAT
}

avpc()
{
    if [ -z "$DEFAULT_KCLUSTER" ]; then
        echo "Lists all resources in given K8S cluster's VPC. Run kdef first"
        return 1
    fi
    aws --profile $ACMD_AWS_PROFILE ec2 describe-tags --filters "Name=resource-type,Values=vpc,Name=value,Values=$DEFAULT_KCLUSTER" --output $AWS_OUTPUT_FORMAT
}

aasgs()
{
    if [ -z "$DEFAULT_KCLUSTER" ]; then
        echo "Lists all autoscaling groups in given K8S cluster. Run kdef first"
        return 1
    fi
    aws --profile $ACMD_AWS_PROFILE autoscaling describe-tags --filters "Name=value,Values=$DEFAULT_KCLUSTER" --query 'Tags[?Key==`KubernetesCluster`].[ResourceId, Value]' --output $AWS_OUTPUT_FORMAT
}
