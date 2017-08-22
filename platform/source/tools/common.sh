#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

#
# Functions commonly used for Applatix scripts
#

# AWS configuration
AWS_CONFIG_PATH=${AWS_CONFIG:-"$HOME/.aws"}
CUSTOMER_AWS_REGION=${AWS_REGION:-"us-west-2"}

# Kubernetes configurations
KUBECONFIG_DOWNLOAD_PATH="/tmp/ax_kube"
DEFAULT_KUBECONFIG_DOWNLOAD_PREFIX="/tmp/ax_kube/cluster_"
KUBECONFIG_DIRECTORY="${HOME}/.kube"
KUBESSH_DIRECTORY="${HOME}/.ssh"
GKE_CONFIG_DIRECTORY="${HOME}/.config"

# New Kubernetes version information
# We get these information from axclustermanager as this is part
# of Applatix's Kubernetes release
NEW_KUBE_SALT_TAR=/kubernetes/server/kubernetes-salt.tar.gz
NEW_KUBE_SERVER_TAR=/kubernetes/server/kubernetes-server-linux-amd64.tar.gz

echo "Generating Kubernetes sha1sum ..."
export NEW_KUBE_VERSION=${AX_KUBE_VERSION:-}
export NEW_KUBE_SALT_SHA1=${AX_KUBE_SALT_HASH:-$(sha1sum ${NEW_KUBE_SALT_TAR} | awk '{ print $1; }')}
export NEW_KUBE_SERVER_SHA1=${AX_KUBE_SERVER_HASH:-$(sha1sum ${NEW_KUBE_SERVER_TAR} | awk '{ print $1; }')}

export NEW_AX_AWS_IMAGE_NAME=${AX_AWS_IMAGE_NAME:-}

# New AX version information
# Currently whenever we do Applatix upgrade, AX software version
# and namespace are environment variables of axclustermanager
export NEW_AX_NAMESPACE=${AX_NAMESPACE:-}
export NEW_AX_VERSION=${AX_VERSION:-}

export CLUSTER_INSTALL_VERSION=`cat /kubernetes/cluster/version.txt`
export NEW_CLUSTER_INSTALL_VERSION=$CLUSTER_INSTALL_VERSION

# Portal url to report upgrade
export PORTAL_URL=${AX_PORTAL_URL:-"https://portal.applatix.com"}

# Cluster configurations
KUBE_ENV_MASTER_PATH=/etc/kubernetes/kube_env.yaml



# Check necessary environment variables for customer
# Usage: ensure-env
ensure-env () {
    for v in $1; do
        if [[ -z "${!v}" ]]; then
            echo "Upgrading Kubernetes for customer, env $v missing" >&2
            exit 1
        fi
    done
}



# Get temp credentials from cross-account arn. This assumes the instance you are running
# on has the permission to do `aws sts assume-role`
#
# Usage: get-aws-credentials customer-cluster-name cross-account-arn customer-external-id
# Returns: access-key-id secret-access-key session-token
get-aws-credentials () {
    aws sts assume-role \
        --role-session-name $1 \
        --role-arn $2 \
        --external-id $3 \
        --duration-seconds 3600 \
        --query 'Credentials.[AccessKeyId, SecretAccessKey, SessionToken]' \
        --output text
}


# Create config and credential files. Note this should ONLY be called for customer
# We write these to files for debugging purposes
# Usage: write-aws-profile access-key-id secret-access-key session-token
write-aws-profile () {
    if [[ -d ${AWS_CONFIG_PATH} ]]; then
        rm -rf ${AWS_CONFIG_PATH}
    fi
    mkdir ${AWS_CONFIG_PATH}
    cat <<EOF > ${AWS_CONFIG_PATH}/credentials
[default]
aws_access_key_id = $1
aws_secret_access_key = $2
aws_session_token = $3
EOF

    cat <<EOF > ${AWS_CONFIG_PATH}/config
[default]
output = text
region = ${CUSTOMER_AWS_REGION}
EOF
}


get-creator-ip () {
    local ok=1
    curl -s --connect-timeout 5 http://169.254.169.254/latest/meta-data/public-ipv4
    if [[ $? != "0" ]]; then
        curl -s --connect-timeout 10 http://ipinfo.io/ip || ok=0
        if [[ ${ok} == 0 ]]; then
            echo ""
        fi
    fi
}


# Usage: get-ax-security-groups ax-cluster-name-id
get-ax-cluster-master-sg () {
    aws ec2 describe-security-groups \
        --filters Name=group-name,Values="kubernetes-master-$1" \
        --query SecurityGroups[].GroupId
}


# Authorize ingress to a security group.
# Usage authorize-security-group-ingress group-id proto port cidr
authorize-security-group-ingress () {
  local ok=1
  local output=""
  output=$(aws ec2 authorize-security-group-ingress \
                --group-id $1 \
                --protocol $2 \
                --port $3 \
                --cidr $4 \
                2>&1) || ok=0
  if [[ ${ok} == 0 ]]; then
    if [[ "${output}" != *"InvalidPermission.Duplicate"* ]]; then
      echo "Error creating security group ingress rule" >&2
      echo "Output: ${output}" >&2
      exit 1
    fi
  fi
}


# Revoke ingress to a security group.
# Usage authorize-security-group-ingress group-id proto port cidr
revoke-security-group-ingress () {
  local ok=1
  local output=""
  output=$(aws ec2 revoke-security-group-ingress \
                --group-id $1 \
                --protocol $2 \
                --port $3 \
                --cidr $4 \
                2>&1) || ok=0
  if [[ ${ok} == 0 ]]; then
    if [[ "${output}" != *"InvalidPermission.NotFound"* ]]; then
      echo "Error creating security group ingress rule" >&2
      echo "Output: ${output}" >&2
      exit 1
    fi
  fi
}


# Get auto scaling groups used by given cluster name id
# Usage get-autoscaling-groups ax-cluster-name-id
get-autoscaling-groups () {
    aws autoscaling describe-tags \
        --filters "Name=value,Values=$1" \
        --query 'Tags[?Key==`KubernetesCluster`].[ResourceId]'
}
