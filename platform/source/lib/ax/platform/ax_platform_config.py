#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
AX platform object sets
"""
from ax.util.const import SECONDS_PER_MINUTE

START_SET = frozenset([
    "axmon-svc",
    "axdb-svc",
    "axopsbootstrap",
    "redis-svc",
    "kafka-zk-svc",
    "axnotification-svc",
    "fixturemanager-svc",
    "axworkflowadc-svc",
    "axamm-svc",
    "gateway-svc",
    "axops",
    "axstats",
    "cron",
    "axconsole-svc",
    "axscheduler-svc",
    "axartifactmanager-svc",
    "default-http-backend",
    "ingress-controller",
    "ingress-controller-int",
    "prometheus",
    "notification-center-svc",
    "platform-post-boot",
])

START_SET_KUBE_SYSTEM_EXT = frozenset([
    "kube-dns",
])

START_SET_KUBE_SYSTEM = frozenset([
    "node-exporter",
])


START_SET_KUBE_SYSTEM_AWS = frozenset([
    "autoscaler",
    "volume-mounts-fixer",
    "master-manager",
    "minion-manager",
])


POST_START_SET_KUBE_SYSTEM = frozenset([
    "applet",
    "fluentd"
])


CREATE_SET = frozenset([
    "registry-secrets",
    "redis-pvc",
    "kafka-zk-pvc",
    "gateway-pvc",
    "prometheus-pvc",
])


KUBERNETES_SET = frozenset([
    "axops-svc",
    "axops-internal-svc",
])

KUBERNETES_SET_AWS = frozenset([
    "applatix-svc",
])


# Create timeouts
OBJ_CREATE_WAIT_TIMEOUT = 25 * SECONDS_PER_MINUTE
OBJ_CREATE_POLLING_INTERVAL = 3
OBJ_CREATE_POLLING_MAX_RETRY = OBJ_CREATE_WAIT_TIMEOUT / OBJ_CREATE_POLLING_INTERVAL

# Create extra poll timeouts
OBJ_CREATE_EXTRA_POLL_TIMEOUT = 15 * SECONDS_PER_MINUTE
OBJ_CREATE_EXTRA_POLL_INTERVAL = 3
OBJ_CREATE_EXTRA_POLL_MAX_RETRY = OBJ_CREATE_EXTRA_POLL_TIMEOUT / OBJ_CREATE_EXTRA_POLL_INTERVAL

# Delete timeouts
OBJ_DELETE_WAIT_TIMEOUT = 2 * SECONDS_PER_MINUTE
OBJ_DELETE_POLLING_INTERVAL = 3
OBJ_DELETE_POLLING_MAX_RETRY = OBJ_DELETE_WAIT_TIMEOUT / OBJ_DELETE_POLLING_INTERVAL

# this should be longer as by the time we poll left ebs, kubernetes master
# might be busy deleting all volumes, which would increase chance of ExceedCallLimit
AWS_EBS_TIMEOUT = 6 * SECONDS_PER_MINUTE
AWS_EBS_POLLING_INTERVAL = 6
AWS_EBS_POLLING_MAX_RETRY = AWS_EBS_TIMEOUT / AWS_EBS_POLLING_INTERVAL

# All axsys pods will be tried to schedule on nodes with following tags
AXSYS_NODE_SELECTOR = "nodeSelector: {ax.tier: applatix}"
AXSYS_NODE_LABEL = "ax.tier=applatix"

# All axuser pods will be tried to schedule on nodes with following tags
AXUSER_NODE_LABEL = "ax.tier=user"

# axstats's resource consumption should be proportional to nodes
# CPU limit has unit `milicore`. Memory limit has unit `Mi`.
# In this case, for the largest cluster we would create (25 nodes, m3.2xlarge),
# axstats will have CPU request:limit = 187:375; Mem request:limit 450:900
# For "xlarge" size (20 nodes, m3.large), CPU: 60:120; Mem: 150:300
# For "large" size (10 nodes, m3.large), CPU: 40:80; Mem: 75:150
# For "medium" size (5 nodes, m3.large), CPU: 20:40; Mem: 45:90
# Requests are set to 1/2 of limit
AXSTATS_PER_NODE_CPU_LIMIT = {
    "m3.large": 8,
    "m3.xlarge": 10,
    "m3.2xlarge": 15
}
AXSTATS_PER_NODE_MEM_LIMIT = {
    "m3.large": 18,
    "m3.xlarge": 25,
    "m3.2xlarge": 36
}
