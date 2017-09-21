#!/bin/bash
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
#
# This script upgrades Kubernetes for a cluster. It contains all reliable hacks
# we've been using for a long time. We have road map to clean these hacks up.

set -e
source bash-helpers.sh

AWSCLI=aws
KUBECONFIG_DOWNLOAD_PATH="/tmp/ax_kube"
DEFAULT_KUBECONFIG_DOWNLOAD_PREFIX="/tmp/ax_kube/cluster_"
KUBECONFIG_DIRECTORY="${HOME}/.kube"

# Flex volume disk type
AX_VOL_DISK_TYPE="${AX_VOL_DISK_TYPE:-gp2}"

# Environments variable required
REQUIRED_ENV="CLUSTER_NAME_ID AX_CUSTOMER_ID OLD_KUBE_VERSION"
REQUIRED_ENV="${REQUIRED_ENV} NEW_KUBE_VERSION NEW_CLUSTER_INSTALL_VERSION KUBERNETES_SERVER_HASH"
REQUIRED_ENV="${REQUIRED_ENV} ARGO_AWS_REGION"

declare -A asg_desired
declare -A asg_min
declare -A asg_max

# Master is also registered as minion, but is not included in ASG
total_desired=1


ensure-aws-envs () {
    unset AWS_DEFAULT_PROFILE
    unset AWS_PROFILE
    unset AWS_DEFAULT_OUTPUT

    if [[ -n "${ARGO_AWS_PROFILE}" ]]; then
        echo "Setting AWS profile to ${ARGO_AWS_PROFILE}"
        export AWS_DEFAULT_PROFILE="${ARGO_AWS_PROFILE}"
    else
        echo "Not setting AWS profile"
    fi
    echo "Setting AWS region to ${ARGO_AWS_REGION}"
    aws configure set region ${ARGO_AWS_REGION}
    export AWS_DEFAULT_OUTPUT=text
}


# Check necessary environment variables for customer
# Usage: ensure-env
ensure-env () {
    echo "Ensuring environment variables ..."

    for v in $1; do
        if [[ -z "${!v}" ]]; then
            echo "Upgrading Kubernetes. Env $v missing" >&2
            exit 1
        fi
    done

    echo "Generating Kubernetes sha1sum ..."
    NEW_KUBE_SALT_TAR=/kubernetes/server/kubernetes-salt.tar.gz

    export NEW_KUBE_SALT_SHA1=${AX_KUBE_SALT_HASH:-$(sha1sum ${NEW_KUBE_SALT_TAR} | awk '{ print $1; }')}
    export NEW_KUBE_SERVER_SHA1=${KUBERNETES_SERVER_HASH}
}


ensure-k-commands () {
    echo "Ensuring k commands"
    # `k` command assumes all config files are in $HOME/.kube
    # in case we are doing it for user, we won't mount a host path
    # to $HOME/.kube and axtool will download config to /tmp so we
    # need to hack it here

    mkdir -p ${KUBECONFIG_DIRECTORY}
    cp ${DEFAULT_KUBECONFIG_DOWNLOAD_PREFIX}${CLUSTER_NAME}*.conf ${KUBECONFIG_DIRECTORY}
    kdef ${CLUSTER_NAME_ID}
}


# Get auto scaling groups used by given cluster name id
# Usage get-autoscaling-groups ax-cluster-name-id
get-autoscaling-groups () {
    aws autoscaling describe-tags \
        --filters "Name=value,Values=$1" \
        --query 'Tags[?Key==`KubernetesCluster`].[ResourceId]'
}


# This function uploads Kubernetes binaries to S3. Nodes will download Kubernetes binaries and salts from S3
upload-kubernetes-server-binaries () {
    echo
    echo
    echo "=== Step 0. Upload binaries."
    echo

    local -r kube_tmp="/tmp/kubernetes/"
    local -r kube_installer_path="${kube_tmp}/installer/${NEW_CLUSTER_INSTALL_VERSION}/"
    mkdir -p "${kube_installer_path}"

    cp -a /kubernetes/server/bootstrap-script "${kube_installer_path}"
    cp -a /kubernetes/server/kubernetes-salt.tar.gz "${kube_installer_path}"

    ${AWSCLI} s3 sync ${kube_tmp} s3://applatix-cluster-${AX_CUSTOMER_ID}-0/kubernetes-staging/v${NEW_KUBE_VERSION} --acl public-read
}



detect-asg () {
    echo "Detecting auto scaling groups"
    ASG_NAMES=$(get-autoscaling-groups ${CLUSTER_NAME_ID})
    if [[ -z "${ASG_NAMES}" ]]; then
        echo "No minion auto scaling groups detected for cluster ${CLUSTER_NAME_ID}" >&2
        exit 1
    else
        echo "Updating autoscaling groups:"
        echo "${ASG_NAMES}"
    fi
}


scale-down-asgs() {
    echo
    echo
    echo "=== Step 1. Scaling down auto scaling groups."
    echo

    detect-asg

    # Retain asg config
    for asg_name in ${ASG_NAMES} ; do
        get-instance-counts ${asg_name}
    done
    echo "Cluster has totally ${total_desired} nodes"

    # Scale down asgs
    for asg_name in ${ASG_NAMES} ; do
        ${AWSCLI} autoscaling update-auto-scaling-group --auto-scaling-group-name $asg_name --min-size 0 --desired-capacity 0
    done

    # Wait for instances to not be InService anymore
    for asg_name in ${ASG_NAMES}; do
        # Scale down to 0 and come back to current instance count. Have to wait until scaling down is started to scale back up.
        while ${AWSCLI} autoscaling describe-auto-scaling-groups --auto-scaling-group-name $asg_name --query AutoScalingGroups[0].Instances[].LifecycleState | grep -q InService ; do
            echo "Waiting for auto scaling group ${asg_name} to scale down ..."
            sleep 10
        done
    done
    echo "All auto scaling groups scaled down to 0"

    # Wait for minions to de-register from Master
    # The only instance left should be Master (also registered as minion)
    while true; do
        local remaining_kube_nodes=$(kn | grep -v "NAME" | wc -l)
        echo "Waiting for all minions to de-register from master: ${remaining_kube_nodes} remaining"
        if [[ ! ${remaining_kube_nodes} -eq 1 ]]; then
            sleep 10
        else
            break
        fi
    done
    echo "All minions de-registered"
}


scale-up-asgs() {
    echo "Scaling up auto scaling groups ..."
    for asg_name in ${ASG_NAMES}; do
        echo "Fixing:" $asg_name
        if [[ ${asg_min[$asg_name]} -eq -1 ]]; then
            echo "Deleting ASG: " $asg_name
            lc=$(${AWSCLI} autoscaling describe-auto-scaling-groups --auto-scaling-group $asg_name \
                                                                    --query 'AutoScalingGroups[].LaunchConfigurationName' \
                                                                    --output text)
            # Wait for the ASG to drain any outstanding activity
            sleep 5
            ${AWSCLI} autoscaling delete-auto-scaling-group --auto-scaling-group-name $asg_name || true
            ${AWSCLI} autoscaling delete-launch-configuration --launch-configuration-name $lc || true
        else
            local min=${asg_min[$asg_name]}
            local desired=${asg_desired[$asg_name]}
            local max=${asg_max[$asg_name]}
            echo "Updating ASG $asg_name to min: ${min}, desired: ${desired}, max: ${max}"
            ${AWSCLI} autoscaling update-auto-scaling-group --auto-scaling-group-name $asg_name \
                                                            --min-size ${min} \
                                                            --desired-capacity ${desired} \
                                                            --max-size ${max}
        fi
    done
}


get-instance-counts() {
    local asg_name=$1
    size=$($AWSCLI autoscaling describe-auto-scaling-groups --auto-scaling-group-name ${asg_name} \
                                                            --query AutoScalingGroups[0].[MinSize,DesiredCapacity,MaxSize])
    local min=$(echo $size | awk '{print $1;}')
    local desired=$(echo $size | awk '{print $2;}')
    local max=$(echo $size | awk '{print $3;}')
    asg_min[$asg_name]=${min}
    asg_desired[$asg_name]=${desired}
    asg_max[$asg_name]=${max}
    echo "Auto scaling group ${asg_name} has minimum ${min}, desired ${desired}, max ${max} nodes"
    total_desired=$((total_desired + asg_desired[$asg_name]))
}


upgrade-launch-config() {
    echo "Upgrading launch configurations ..."
    local aws_profile_arg=""
    if [[ ! -z ${AWS_DEFAULT_PROFILE+x} ]]; then
        aws_profile_arg="--profile ${AWS_DEFAULT_PROFILE}"
    fi
    /ax/bin/minion_upgrade --new-kube-version ${NEW_KUBE_VERSION} \
                           --new-kube-server-hash ${NEW_KUBE_SERVER_SHA1} \
                           --new-cluster-install-version ${NEW_CLUSTER_INSTALL_VERSION} \
                           --new-kube-salt-hash ${NEW_KUBE_SALT_SHA1} \
                           --region ${ARGO_AWS_REGION} \
                           --ax-vol-disk-type ${AX_VOL_DISK_TYPE} \
                           --cluster-name-id ${CLUSTER_NAME_ID} \
                           ${aws_profile_arg}
}


upgrade-master () {
    # Step 2. Configure Kubernetes master.
    echo
    echo
    echo "=== Step 2. Configure Kubernetes master."
    echo

    local aws_profile_arg=""
    if [[ ! -z ${AWS_DEFAULT_PROFILE+x} ]]; then
        aws_profile_arg="--profile ${AWS_DEFAULT_PROFILE}"
    fi

    /ax/bin/master_manager ${CLUSTER_NAME_ID} upgrade --region ${ARGO_AWS_REGION} ${aws_profile_arg}
    rm -f ~/.ssh/known_hosts
}


upgrade-minion () {
    # Step 3. Configure Kubernetes minions.
    echo
    echo
    echo "=== Step 3. Configure Kubernetes minions."
    echo

    upgrade-launch-config

    scale-up-asgs
}


wait-for-master () {
    # Step 4. Wait for kubernetes to come up.
    echo
    echo
    echo "=== Step 4. Wait for kubernetes to come up."
    echo

    # Give master totally 10 min to bootstrap
    local attempt=0
    while ! k version | grep ${NEW_KUBE_VERSION} ; do
        echo "Waiting for master to initialize ..."
        sleep 20
        attempt=$(($attempt+1))
        if [[ ${attempt} -gt 30 ]]; then
            echo
            echo "Master fail to boot up, or is unlikely to be healthy"
            echo
            exit 1
        fi
    done
    echo "Master setup done"
}


wait-for-minion () {
    echo
    echo
    echo "=== Step 5. Wait for minions."
    echo
    # We give minions 15 min for them to come up
    local attempt=0

    # Wait for all minions to be in service
    for asg_name in ${ASG_NAMES} ; do
        while true; do
            local current_count=$(${AWSCLI} autoscaling describe-auto-scaling-groups --auto-scaling-group-name $asg_name \
                                                                                     --query AutoScalingGroups[0].Instances[].LifecycleState \
                                                                                     --output json | grep -c InService)
            local desired=${asg_desired[$asg_name]}
            echo "Waiting for ${desired} minions in auto scaling group ${asg_name} to come up, currently ${current_count} ..."
            if [[ ${current_count} -eq ${desired} ]]; then
                break
            fi
            sleep 10
            attempt=$(($attempt+1))
            if [[ ${attempt} -gt 90 ]]; then
                echo
                echo "Failed to provision all minions"
                echo
                exit 1
            fi
        done
    done
    echo "All minions came up"

    # Wait for all minions to be registered to master
    while true; do
        local current_count=$(kn | grep -v Not | grep -c Ready)
        echo "Waiting for ${total_desired} minions to register in master, currently ${current_count} ..."
        if [[ ${current_count} -eq ${total_desired} ]]; then
            break
        fi
        sleep 10
        attempt=$(($attempt+1))
        if [[ ${attempt} -gt 90 ]]; then
            echo
            echo "Not all minions are healthy"
            echo
            exit 1
        fi
    done
    echo "All minions registered"
}


#
# Main upgrade routine
#

echo
echo
echo
echo "=====> Start upgrading Kubernetes"
echo
echo

ensure-env ${REQUIRED_ENV}

ensure-k-commands

ensure-aws-envs

upload-kubernetes-server-binaries

# TODO (#253): use cluster pauser / restarter to scale down, scale up, and retain information

scale-down-asgs

upgrade-master

upgrade-minion

wait-for-master

wait-for-minion

echo
echo
echo
echo "=== Upgraded Kubernetes of cluster ${CLUSTER_NAME_ID} from ${OLD_KUBE_VERSION} to ${NEW_KUBE_VERSION}"
echo
echo

