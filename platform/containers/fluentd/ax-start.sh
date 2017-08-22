#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.

# Update td-agent configuration file and start it.

set -ex

[[ -n "$AX_CUSTOMER_ID" ]] || exit 1
[[ -n "$AX_CLUSTER_NAME_ID" ]] || exit 1
[[ -n "$AX_NODE_NAME" ]] || exit 1

# Set default to cluster bucket if it's not defined or defined as empty.
ARGO_LOG_BUCKET_NAME=${ARGO_LOG_BUCKET_NAME:-applatix-cluster-${AX_CUSTOMER_ID}-0}
[[ -n "$ARGO_LOG_BUCKET_NAME" ]] || ARGO_LOG_BUCKET_NAME=applatix-cluster-${AX_CUSTOMER_ID}-0
ARGO_CLUSTER_ID=${AX_CLUSTER_NAME_ID:(-36)}
ARGO_CLUSTER_NAME=${AX_CLUSTER_NAME_ID/-${ARGO_CLUSTER_ID}/}

sed -i 's#$(ARGO_LOG_BUCKET_NAME)#'$ARGO_LOG_BUCKET_NAME'#g' /etc/td-agent/td-agent.conf
sed -i 's#$(ARGO_CLUSTER_NAME)#'$ARGO_CLUSTER_NAME'#g' /etc/td-agent/td-agent.conf
sed -i 's#$(ARGO_CLUSTER_ID)#'$ARGO_CLUSTER_ID'#g' /etc/td-agent/td-agent.conf
sed -i 's#$(ARGO_NODE_NAME)#'$ARGO_NODE_NAME'#g' /etc/td-agent/td-agent.conf
exec td-agent
sleep 3600
