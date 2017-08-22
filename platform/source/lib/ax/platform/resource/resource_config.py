#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Resource config base case, which defines information commonly
needed for other resources
"""

from ax.platform.resource.consts import RESOURCE_PER_POD_STANDARD, RESOURCE_PER_POD_COMPUTE, DEFAULT_POD_PER_CU, EC2_PARAMS


class ResourceConfigBase(object):
    def __init__(self, usr_node_type, usr_node_max, ax_node_type, ax_node_max, cluster_type):
        self._cluster_type = cluster_type
        self._usr_node_type = usr_node_type
        self._usr_node_max = usr_node_max
        self._ax_node_type = ax_node_type
        self._ax_node_max = ax_node_max

        self._max_nodes = usr_node_max + ax_node_max
        self._axsys_cpu = 0
        self._axsys_mem = 0
        self._axuser_cpu = 0
        self._axuser_mem = 0
        self._cu_per_usr_node = 0
        self._cu_per_ax_node = 0

        self._calculate_total_cluster_resource()

    @property
    def max_node(self):
        return self._max_nodes

    @property
    def max_cu(self):
        return self._cluster_max_cu

    @property
    def cu_per_usr_node(self):
        return self._cu_per_usr_node

    @property
    def cu_per_ax_node(self):
        return self._cu_per_ax_node

    @property
    def axsys_cpu(self):
        return self._axsys_cpu

    @property
    def axsys_mem(self):
        return self._axsys_mem

    @property
    def axuser_cpu(self):
        return self._axuser_cpu

    @property
    def axuser_mem(self):
        return self._axuser_mem

    def _calculate_total_cluster_resource(self):
        user_node_cpu = EC2_PARAMS[self._usr_node_type]["cpu"]
        user_node_mem = EC2_PARAMS[self._usr_node_type]["memory"]
        axsys_node_cpu = EC2_PARAMS[self._ax_node_type]["cpu"]
        axsys_node_mem = EC2_PARAMS[self._ax_node_type]["memory"]

        self._axsys_cpu = axsys_node_cpu * self._ax_node_max
        self._axsys_mem = axsys_node_mem * self._ax_node_max
        self._axuser_cpu = user_node_cpu * self._usr_node_max
        self._axuser_mem = user_node_mem * self._usr_node_max

        # TODO: remove circular dependency with ax.platform.cluster_config
        # and use literals from AXClusterType
        if self._cluster_type == "standard":
            avg_cpu_per_pod = RESOURCE_PER_POD_STANDARD["cpu"]
            avg_mem_per_pod = RESOURCE_PER_POD_STANDARD["memory"]
        elif self._cluster_type == "compute":
            avg_cpu_per_pod = RESOURCE_PER_POD_COMPUTE["cpu"]
            avg_mem_per_pod = RESOURCE_PER_POD_COMPUTE["memory"]
        else:
            raise ValueError("Unknown cluster type: {}".format(self._cluster_type))

        # default compute unit to 1 if less than 1
        self._cu_per_ax_node = int((axsys_node_cpu / avg_cpu_per_pod + axsys_node_mem
                                              / avg_mem_per_pod) / 2 / DEFAULT_POD_PER_CU) or 1
        self._cu_per_usr_node = int((user_node_cpu / avg_cpu_per_pod + user_node_mem
                                              / avg_mem_per_pod) / 2 / DEFAULT_POD_PER_CU) or 1

        self._cluster_max_cu = self._cu_per_ax_node * self._ax_node_max + self._cu_per_usr_node * self._usr_node_max
