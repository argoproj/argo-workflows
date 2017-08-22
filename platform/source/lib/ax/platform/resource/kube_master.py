#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
This object configures master resource requirement
"""
import logging
from ax.platform.resource.resource_config import ResourceConfigBase
from ax.platform.resource.consts import EC2_PARAMS
from ax.util.const import MiB

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


# Master component per CU config. CPU has unit "milicore", Memory has unit "MiB"
KUBE_MASTER_CPU_DEFAULT = {
    "API_SERVER_CPU_REQ": 15,
    "KUBE_CONTROLLER_CPU_REQ": 15,
    "KUBE_SCHED_CPU_REQ": 10,
    "KUBE_RESCHED_CPU_REQ": 2
}

KUBE_MASTER_MEM_DEFAULT = {
    "API_SERVER_MEM_REQ": 60,
    "DAEMONSET_CACHE": 8,
    "REPLICASET_CACHE": 30,
    "RC_CACHE": 30,
    "KUBE_SCHED_MEM_REQ": 30,
    "KUBE_RESCHED_MEM_REQ": 10
}

# Currently leave parallelism envs configured by kube-up
KUBE_MASTER_OTHER_DEFAULT = {
    "API_SERVER_THROTTLING": 10,
    "KUBE_SCHED_API_QPS": 3,
    "KUBE_SCHED_API_BURST": 6
}


class KubeMasterResourceConfig(ResourceConfigBase):
    """
    Based on cluster resource, estimate master resource config
    """
    def __init__(self, usr_node_type, usr_node_max, ax_node_type, ax_node_max, cluster_type):
        super(KubeMasterResourceConfig, self).__init__(usr_node_type, usr_node_max, ax_node_type,
                                                       ax_node_max, cluster_type)
        self._master_instance_type = None
        self._master_root_device_size = 0
        self._master_pd_size = 0

        self._calculate_total_cluster_resource()
        self._calculate_master_config()

    @property
    def master_instance_type(self):
        return self._master_instance_type

    @property
    def master_root_device_size(self):
        return self._master_root_device_size

    @property
    def master_pd_size(self):
        return self._master_pd_size

    @property
    def kube_up_env(self):
        rst = {}
        for item in KUBE_MASTER_CPU_DEFAULT:
            rst.update({
                item: str(KUBE_MASTER_CPU_DEFAULT[item])
            })
        for item in KUBE_MASTER_MEM_DEFAULT:
            rst.update({
                item: str(KUBE_MASTER_MEM_DEFAULT[item])
            })
        for item in KUBE_MASTER_OTHER_DEFAULT:
            rst.update({
                item: str(KUBE_MASTER_OTHER_DEFAULT[item])
            })
        rst["MASTER_DISK_SIZE"] = str(self._master_pd_size)
        rst["MASTER_ROOT_DISK_SIZE"] = str(self._master_root_device_size)
        rst["AX_COMPUTE_UNIT_MAX"] = str(self.max_cu)
        rst["AX_NUM_NODES_MAX"] = str(self._max_nodes)
        return rst

    def _calculate_master_config(self):
        cpu_req = 0
        mem_req = 0
        for item in KUBE_MASTER_CPU_DEFAULT:
            cpu_req += KUBE_MASTER_CPU_DEFAULT[item] * self.max_cu
        for item in KUBE_MASTER_MEM_DEFAULT:
            mem_req += KUBE_MASTER_MEM_DEFAULT[item] * self.max_cu

        logger.info("Master kube components requires %sm CPU, %s MiB memory.", cpu_req, mem_req)

        # Other components also needs resources
        cpu_req *= 1.5
        mem_req *= 2.5
        logger.info("Master requires %sm CPU and %s MiB memory", cpu_req, mem_req)

        if self.max_cu < 20:
            for node_type in ["m3.medium", "m3.large", "m3.xlarge"]:
                if EC2_PARAMS[node_type]["cpu"] >= cpu_req and EC2_PARAMS[node_type]["memory"] >= mem_req * MiB:
                    self._master_instance_type = node_type
                    break
        else:
            # Putting xlarge in this category is because Kubernetes API server has default
            # cache reservations, which adds up to 25 * cluster_max_cu + 4100 Mi
            # Since we already see OOM with xlarge (20 cu) case, we bump up memory availability
            # with r3 type instances here
            for node_type in ["r3.large", "r3.xlarge", "r3.2xlarge", "r3.4xlarge"]:
                if EC2_PARAMS[node_type]["cpu"] >= cpu_req and EC2_PARAMS[node_type]["memory"] >= mem_req * MiB:
                    self._master_instance_type = node_type
                    break
        assert self._master_instance_type, "Cannot find proper master instance type"

        if self.max_cu < 15:
            # small(5 cu), medium(10 cu)
            self._master_root_device_size = 16
            self._master_pd_size = 20
        elif self.max_cu < 50:
            # large (20 cu)
            self._master_root_device_size = 20
            self._master_pd_size = 40
        elif self.max_cu < 150:
            # xlarge (120 cu)
            self._master_root_device_size = 32
            self._master_pd_size = 60
        else:
            self._master_root_device_size = 40
            self._master_pd_size = 100
        logger.info("Master Config: instance %s, root device %s GB, pd size %s GB", self.master_instance_type,
                    self._master_root_device_size, self._master_pd_size)
