#!/bin/bash
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

AXSYS_NUM_NODES=${AXSYS_NUM_NODES:-2}

# Number of on-demand instances for "axuser".
AXUSER_ON_DEMAND_NUM_NODES=${AXUSER_ON_DEMAND_NUM_NODES:-0}

# Global flag to indicate whether to use spot-instances.
AX_USE_SPOT_INSTANCES=${AX_USE_SPOT_INSTANCES:-false}

AX_ENABLE_MASTER_KUBE_PROXY=${AX_ENABLE_MASTER_KUBE_PROXY:-true}
AX_ENABLE_MASTER_FLUENTD=${AX_ENABLE_MASTER_FLUENTD:-true}
AX_TCP_KEEPALIVE=${AX_TCP_KEEPALIVE:-true}
AX_INSTALLER_VERSION_PATH=${AX_INSTALLER_VERSION_PATH:-"/kubernetes/cluster/version.txt"}

# More dynamic resource configuration for master. These envs should be
# provided by axinstaller during installation time

# (harry) Based on limited documentation such as
# http://blog.kubernetes.io/2016/07/kubernetes-updates-to-performance-and-scalability-in-1.3.html
# and comment in Kubernetes source code, Kubernetes API server should be 
# allocated with 60 MB memory for ~30 Pods according to some benchmarks.
#
# Let's be more aggressive here:
#   - 1 Compute Unit can run 25 Pods in cluster
#   - On average 1 Pod uses 100m CPU and 250 Mi Memory
#
# For medium sized cluster, we have 5x m3.large minion nodes, and therefore
# each node can run ~25 Pods (20 by CPU and 30 by Memory) so 5 Compute Units
# for such cluster we have 5 Compute Units
AX_COMPUTE_UNIT_MAX=${AX_COMPUTE_UNIT_MAX:-5}

# Cluster node related
# Note we still leave this number as kubernetes might use pillar['num_nodes']
# for some other uses, though not now.
AX_NUM_NODES_MAX=${AX_NUM_NODES_MAX:-5}

# kube-apiserver related (per compute unit)
API_SERVER_CPU_REQ=${API_SERVER_CPU_REQ:-15}
API_SERVER_MEM_REQ=${API_SERVER_MEM_REQ:-60}
API_SERVER_THROTTLING=${API_SERVER_THROTTLING:-15}

# kube-controller-manager related (per compute unit)
KUBE_CONTROLLER_CPU_REQ=${KUBE_CONTROLLER_CPU_REQ:-20}
DAEMONSET_CACHE=${DAEMONSET_CACHE:-8}
REPLICASET_CACHE=${REPLICASET_CACHE:-30}
RC_CACHE=${RC_CACHE:-30}

# Dynamically determine concurrency of kube-controller (cluster value)
# Default values from kubernetes documentation are referenced
# See http://kubernetes.io/docs/admin/kube-controller-manager/
#
# TODO: More data needed for these heuristics
if [[ ${AX_COMPUTE_UNIT_MAX} -lt 15 ]]; then
    # medium(5 cu), large(10 cu)
    DEPLOYMENT_SYNC_BATCH=${DEPLOYMENT_SYNC_BATCH:-3}
    ENDPOINT_SYNC_BATCH=${ENDPOINT_SYNC_BATCH:-3}
    GARBAGE_COLLECTOR_SYNC_BATCH=${GARBAGE_COLLECTOR_SYNC_BATCH:-5}
    NAMESPACE_SYNC_BATCH=${NAMESPACE_SYNC_BATCH:-2}
    REPLICASET_SYNC_BATCH=${REPLICASET_SYNC_BATCH:-3}
    RESOURCE_QUOTA_SYNC_BATCH=${RESOURCE_QUOTA_SYNC_BATCH:-3}
    SERVICE_SYNC_BATCH=${SERVICE_SYNC_BATCH:-2}
    SA_TOKEN_SYNC_BATCH=${SA_TOKEN_SYNC_BATCH:-3}
    RC_SYNC_BATCH=${RC_SYNC_BATCH:-3}
elif [[ ${AX_COMPUTE_UNIT_MAX} -lt 50 ]]; then
    # xlarge (20 cu)
    DEPLOYMENT_SYNC_BATCH=${DEPLOYMENT_SYNC_BATCH:-8}
    ENDPOINT_SYNC_BATCH=${ENDPOINT_SYNC_BATCH:-8}
    GARBAGE_COLLECTOR_SYNC_BATCH=${GARBAGE_COLLECTOR_SYNC_BATCH:-15}
    NAMESPACE_SYNC_BATCH=${NAMESPACE_SYNC_BATCH:-6}
    REPLICASET_SYNC_BATCH=${REPLICASET_SYNC_BATCH:-8}
    RESOURCE_QUOTA_SYNC_BATCH=${RESOURCE_QUOTA_SYNC_BATCH:-8}
    SERVICE_SYNC_BATCH=${SERVICE_SYNC_BATCH:-8}
    SA_TOKEN_SYNC_BATCH=${SA_TOKEN_SYNC_BATCH:-8}
    RC_SYNC_BATCH=${RC_SYNC_BATCH:-8}
elif [[ ${AX_COMPUTE_UNIT_MAX} -lt 150 ]]; then
    # 2xlarge (120 cu)
    DEPLOYMENT_SYNC_BATCH=${DEPLOYMENT_SYNC_BATCH:-20}
    ENDPOINT_SYNC_BATCH=${ENDPOINT_SYNC_BATCH:-20}
    GARBAGE_COLLECTOR_SYNC_BATCH=${GARBAGE_COLLECTOR_SYNC_BATCH:-30}
    NAMESPACE_SYNC_BATCH=${NAMESPACE_SYNC_BATCH:-10}
    REPLICASET_SYNC_BATCH=${REPLICASET_SYNC_BATCH:-20}
    RESOURCE_QUOTA_SYNC_BATCH=${RESOURCE_QUOTA_SYNC_BATCH:-15}
    SERVICE_SYNC_BATCH=${SERVICE_SYNC_BATCH:-15}
    SA_TOKEN_SYNC_BATCH=${SA_TOKEN_SYNC_BATCH:-15}
    RC_SYNC_BATCH=${RC_SYNC_BATCH:-20}
else
    DEPLOYMENT_SYNC_BATCH=${DEPLOYMENT_SYNC_BATCH:-40}
    ENDPOINT_SYNC_BATCH=${ENDPOINT_SYNC_BATCH:-40}
    GARBAGE_COLLECTOR_SYNC_BATCH=${GARBAGE_COLLECTOR_SYNC_BATCH:-60}
    NAMESPACE_SYNC_BATCH=${NAMESPACE_SYNC_BATCH:-20}
    REPLICASET_SYNC_BATCH=${REPLICASET_SYNC_BATCH:-40}
    RESOURCE_QUOTA_SYNC_BATCH=${RESOURCE_QUOTA_SYNC_BATCH:-30}
    SERVICE_SYNC_BATCH=${SERVICE_SYNC_BATCH:-30}
    SA_TOKEN_SYNC_BATCH=${SA_TOKEN_SYNC_BATCH:-20}
    RC_SYNC_BATCH=${RC_SYNC_BATCH:-40}
fi


# kube-scheduler related (per compute unit)
KUBE_SCHED_CPU_REQ=${KUBE_SCHED_CPU_REQ:-10}
KUBE_SCHED_MEM_REQ=${KUBE_SCHED_MEM_REQ:-30}
KUBE_SCHED_API_QPS=${KUBE_SCHED_API_QPS:-3}
KUBE_SCHED_API_BURST=${KUBE_SCHED_API_BURST:-6}


# rescheduler related (per compute unit)
KUBE_RESCHED_CPU_REQ=${KUBE_RESCHED_CPU_REQ:-2}
KUBE_RESCHED_MEM_REQ=${KUBE_RESCHED_MEM_REQ:-10}


# We also need to configure kubelet before we create cluster
# This interface is constructed for kubelet, for tuning the
# following flags:
#   --kube-api-burst
#   --kube-api-qps
#   --event-qps
#   --event-burst
#   --registry-qps
#   --registry-burst
#   --max-pods
#   --max-open-files
#   --system-reserved
#
# Currently are only tuning the following flags on Minion ONLY
#   --max-pods
#   --system-reserved

AX_CONFIG_MASTER_KUBELET=${AX_CONFIG_MASTER_KUBELET:-false}
AX_CONFIG_MINION_KUBELET=${AX_CONFIG_MINION_KUBELET:-true}

# QPSs (Query per second)
MASTER_KUBELET_API_QPS=${MASTER_KUBELET_API_QPS:-}
MASTER_KUBELET_EVENT_QPS=${MASTER_KUBELET_EVENT_QPS:-}
MASTER_KUBELET_REGISTRY_QPS=${MASTER_KUBELET_REGISTRY_QPS:-}
MINION_KUBELET_API_QPS=${MINION_KUBELET_API_QPS:-}
MINION_KUBELET_EVENT_QPS=${MINION_KUBELET_EVENT_QPS:-}
MINION_KUBELET_REGISTRY_QPS=${MINION_KUBELET_REGISTRY_QPS:-}

# Bursts
MASTER_KUBELET_API_BURST=${MASTER_KUBELET_API_BURST:-}
MASTER_KUBELET_EVENT_BURST=${MASTER_KUBELET_EVENT_BURST:-}
MASTER_KUBELET_REGISTRY_BURST=${MASTER_KUBELET_REGISTRY_BURST:-}
MINION_KUBELET_API_BURST=${MINION_KUBELET_API_BURST:-}
MINION_KUBELET_EVENT_BURST=${MINION_KUBELET_EVENT_BURST:-}
MINION_KUBELET_REGISTRY_BURST=${MINION_KUBELET_REGISTRY_BURST:-}

# Misc.
MASTER_MAX_POD=${MASTER_MAX_POD:-}
MASTER_KUBELET_MAX_OPEN_FILE=${MASTER_KUBELET_MAX_OPEN_FILE:-}
MASTER_KUBE_SYS_RESERVED=${MASTER_KUBE_SYS_RESERVED:-}  # Should be something such as "cpu=100m,memory=150Mi"
MINION_MAX_POD=${MINION_MAX_POD:-80}
MINION_KUBELET_MAX_OPEN_FILE=${MINION_KUBELET_MAX_OPEN_FILE:-}
MINION_KUBE_SYS_RESERVED=${MINION_KUBE_SYS_RESERVED:-}


# Container lifecycle related configurations for kubelet
# These are deprecated but are still effective as of 1.4.3
# See documentation at
# https://github.com/kubernetes/community/blob/master/contributors/design-proposals/kubelet-eviction.md

# For configuring --minimum-container-ttl-duration
MIN_CONTAINER_GC_TTL=${MIN_CONTAINER_GC_TTL:-180s}

# For configuring --maximum-dead-containers, kubernetes default: 100
MAX_DEAD_CONTAINERS=${MAX_DEAD_CONTAINERS:-}

# For configuring --maximum-dead-containers-per-container, kubernetes default: 2
MAX_DEAD_CONTAINERS_PER_CONTAINER=${MAX_DEAD_CONTAINERS_PER_CONTAINER:-}


# Docker options
DOCKER_LOG_DRIVER=${DOCKER_LOG_DRIVER:-"json-file"}
if [[ "${DOCKER_LOG_DRIVER}" == "json-file" ]]; then
    DOCKER_LOG_MAX_FILE=${DOCKER_LOG_MAX_FILE:-5}
    DOCKER_LOG_MAX_SIZE=${DOCKER_LOG_MAX_SIZE:-10m}
    DOCKER_LOG_OPTS="--log-driver ${DOCKER_LOG_DRIVER}"
    DOCKER_LOG_OPTS="${DOCKER_LOG_OPTS} --log-opt max-size=${DOCKER_LOG_MAX_SIZE}"
    DOCKER_LOG_OPTS="${DOCKER_LOG_OPTS} --log-opt max-file=${DOCKER_LOG_MAX_FILE}"
else
    echo "Docker log driver ${DOCKER_LOG_DRIVER} is not supported by Applatix yet. Sorry!" >&2
    exit 1
fi

ENABLE_DOCKER_DEBUG=${ENABLE_DOCKER_DEBUG:-true}
