#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Calculate parameters to configure resource requirement for
axsys component
"""


from ax.platform.resource.consts import EC2_PARAMS
from ax.platform.resource.resource_config import ResourceConfigBase
from ax.util.const import GiB


class AXSYSResourceConfig(ResourceConfigBase):
    """
    This class decides how to determine axsys resources
    """
    def __init__(self, usr_node_type, usr_node_max, ax_node_type, ax_node_max, cluster_type):
        super(AXSYSResourceConfig, self).__init__(usr_node_type, usr_node_max, ax_node_type, ax_node_max, cluster_type)
        self._cpu_multiplier = 1
        self._mem_multiplier = 1
        self._disk_multiplier = 1
        self._daemon_cpu_multiplier = 1
        self._daemon_mem_multiplier = 1
        self._calcualte_resource_multipliers()

    @property
    def cpu_multiplier(self):
        return self._cpu_multiplier

    @property
    def mem_multiplier(self):
        return self._mem_multiplier

    @property
    def disk_multiplier(self):
        return self._disk_multiplier

    @property
    def daemon_cpu_multiplier(self):
        return self._daemon_cpu_multiplier

    @property
    def daemon_mem_multiplier(self):
        return self._daemon_mem_multiplier

    # This is a hack for short term, we simply multiply the numbers
    # we use for "medium" cluster size with the multiplier
    # TODO: change axsys config algorithm when we have more data
    def _calcualte_resource_multipliers(self):
        # 2x m3.large nodes
        axsys_unit_cpu = 4000.0
        axsys_unit_mem = 15 * GiB

        self._cpu_multiplier = self.axsys_cpu / axsys_unit_cpu
        self._mem_multiplier = self.axsys_mem / axsys_unit_mem
        self._disk_multiplier = (self._cpu_multiplier + self._mem_multiplier) / 2

        # Use m3.large as unit node
        self._daemon_cpu_multiplier = EC2_PARAMS[self._usr_node_type]["cpu"] / 2000.0
        self._daemon_mem_multiplier = EC2_PARAMS[self._usr_node_type]["memory"] / (7.5 * GiB)


