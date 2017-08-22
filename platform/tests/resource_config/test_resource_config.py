#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import math
import os
import random
import uuid

from ax.util.const import GiB
from ax.platform.cluster_config import AXClusterType
from ax.platform.resource.resource_config import ResourceConfigBase
from ax.platform.resource import AXSYSResourceConfig, KubeMasterResourceConfig
from ax.util import ResourceValueConverter
from ax.util.resource import literal_to_var, valid_storage_units, valid_cpu_units


os.environ["AX_CUSTOMER_ID"] = str(uuid.uuid4())
os.environ["AX_CLUSTER_NAME_ID"] = "test-cluster-" + str(uuid.uuid4())


def test_resource_config_base_medium():
    # Medium size cluster
    rc = ResourceConfigBase(usr_node_type="m3.large",
                            usr_node_max=3,
                            ax_node_type="m3.large",
                            ax_node_max=2,
                            cluster_type=AXClusterType.STANDARD)
    assert rc.max_node == 5
    assert rc.max_cu == 5
    assert rc.cu_per_ax_node == 1
    assert rc.cu_per_usr_node == 1
    assert rc.axsys_cpu == 4000
    assert rc.axuser_cpu == 6000
    assert rc.axsys_mem == 2 * 7.5 * GiB
    assert rc.axuser_mem == 3 * 7.5 * GiB


def test_resource_config_base_large():
    # Large size cluster
    rc = ResourceConfigBase(usr_node_type="m3.large",
                            usr_node_max=8,
                            ax_node_type="m3.large",
                            ax_node_max=2,
                            cluster_type=AXClusterType.STANDARD)
    assert rc.max_node == 10
    assert rc.max_cu == 10
    assert rc.cu_per_ax_node == 1
    assert rc.cu_per_usr_node == 1
    assert rc.axsys_cpu == 2 * 2000
    assert rc.axuser_cpu == 8 * 2000
    assert rc.axsys_mem == 2 * 7.5 * GiB
    assert rc.axuser_mem == 8 * 7.5 * GiB


def test_resource_config_base_xlarge():
    # xLarge size cluster
    rc = ResourceConfigBase(usr_node_type="m3.large",
                            usr_node_max=18,
                            ax_node_type="m3.large",
                            ax_node_max=3,
                            cluster_type=AXClusterType.STANDARD)
    assert rc.max_node == 21
    assert rc.max_cu == 21
    assert rc.cu_per_ax_node == 1
    assert rc.cu_per_usr_node == 1
    assert rc.axsys_cpu == 3 * 2000
    assert rc.axuser_cpu == 18 * 2000
    assert rc.axsys_mem == 3 * 7.5 * GiB
    assert rc.axuser_mem == 18 * 7.5 * GiB


def test_resource_config_base_2xlarge():
    # 2xLarge size cluster
    rc = ResourceConfigBase(usr_node_type="m3.2xlarge",
                            usr_node_max=28,
                            ax_node_type="m3.2xlarge",
                            ax_node_max=2,
                            cluster_type=AXClusterType.STANDARD)
    assert rc.max_node == 30
    assert rc.max_cu == 120
    assert rc.cu_per_ax_node == 4
    assert rc.cu_per_usr_node == 4
    assert rc.axsys_cpu == 2 * 8000
    assert rc.axuser_cpu == 28 * 8000
    assert rc.axsys_mem == 2 * 30 * GiB
    assert rc.axuser_mem == 28 * 30 * GiB


# For testing master / minion config, we just make sure our dummy
# algorithm produces same results as the already-tested values
# we are using now
def test_kube_master_config_medium():
    km = KubeMasterResourceConfig(usr_node_type="m3.large",
                                  usr_node_max=3,
                                  ax_node_type="m3.large",
                                  ax_node_max=2,
                                  cluster_type=AXClusterType.STANDARD)
    assert km.max_cu == 5
    assert km.master_instance_type == "m3.medium"
    assert km.master_root_device_size == 16
    assert km.master_pd_size == 20


def test_kube_master_config_large():
    km = KubeMasterResourceConfig(usr_node_type="m3.large",
                                  usr_node_max=8,
                                  ax_node_type="m3.large",
                                  ax_node_max=2,
                                  cluster_type=AXClusterType.STANDARD)
    assert km.max_cu == 10
    assert km.master_instance_type == "m3.large"
    assert km.master_root_device_size == 16
    assert km.master_pd_size == 20


def test_kube_master_config_xlarge():
    km = KubeMasterResourceConfig(usr_node_type="m3.large",
                                  usr_node_max=18,
                                  ax_node_type="m3.large",
                                  ax_node_max=3,
                                  cluster_type=AXClusterType.STANDARD)
    assert km.max_cu == 21
    assert km.master_instance_type == "r3.large"
    assert km.master_root_device_size == 20
    assert km.master_pd_size == 40


def test_kube_master_config_2xlarge():
    km = KubeMasterResourceConfig(usr_node_type="m3.2xlarge",
                                  usr_node_max=28,
                                  ax_node_type="m3.2xlarge",
                                  ax_node_max=2,
                                  cluster_type=AXClusterType.STANDARD)
    assert km.max_cu == 120
    assert km.master_instance_type == "r3.2xlarge"
    assert km.master_root_device_size == 32
    assert km.master_pd_size == 60


def test_axsys_config_medium():
    ar = AXSYSResourceConfig("m3.large", 3, "m3.large", 2, AXClusterType.STANDARD)
    assert ar.cpu_multiplier == 1
    assert ar.mem_multiplier == 1
    assert ar.disk_multiplier == 1
    assert ar.daemon_cpu_multiplier == 1
    assert ar.daemon_mem_multiplier == 1


def test_axsys_config_large():
    ar = AXSYSResourceConfig("m3.large", 8, "m3.large", 2, AXClusterType.STANDARD)
    assert ar.cpu_multiplier == 1
    assert ar.mem_multiplier == 1
    assert ar.disk_multiplier == 1
    assert ar.daemon_cpu_multiplier == 1
    assert ar.daemon_mem_multiplier == 1


def test_axsys_config_xlarge():
    ar = AXSYSResourceConfig("m3.large", 18, "m3.large", 3, AXClusterType.STANDARD)
    assert ar.cpu_multiplier == 1.5
    assert ar.mem_multiplier == 1.5
    assert ar.disk_multiplier == 1.5
    assert ar.daemon_cpu_multiplier == 1
    assert ar.daemon_mem_multiplier == 1


def test_axsys_config_2xlarge():
    ar = AXSYSResourceConfig("m3.2xlarge", 28, "m3.2xlarge", 2, AXClusterType.STANDARD)
    assert ar.cpu_multiplier == 4
    assert ar.mem_multiplier == 4
    assert ar.disk_multiplier == 4
    assert ar.daemon_cpu_multiplier == 4
    assert ar.daemon_mem_multiplier == 4


def test_resource_converter_cpu_milicores():
    for _ in range(10):
        for u in valid_cpu_units:
            raw = random.randint(0, 10000)
            raw_value = "{}{}".format(raw, u)
            rvc = ResourceValueConverter(raw_value, random.choice(["CPU", "cpu"]))
            assert rvc.raw == raw
            # Converting from mili-core to mili-core should have same value as raw
            assert rvc.convert(random.choice(valid_cpu_units)) == raw

            # Converting from mili-core to core should have raw / 1000 with roundup to 2 digits
            assert rvc.convert("") == round(float(raw) / 1000, 3)


def test_resource_converter_cpu_cores():
    for _ in range(20):
        for digit in range(3):
            raw = round(random.uniform(0.0, 10.0), digit)
            rvc = ResourceValueConverter(raw, random.choice(["CPU", "cpu"]))

            # Fine... this is documented https://docs.python.org/2/tutorial/floatingpoint.html
            assert math.fabs(rvc.raw - raw * 1000) <= 1
            for u in valid_cpu_units:
                # CPU milicore converion removes fractions
                assert rvc.convert(u) == int(raw * 1000)
            assert math.fabs(rvc.convert("") - round(float(raw), 3)) <= 0.001


def test_resource_converter_storage():
    storage_names = ["memory", "mem", "disk"]
    # This is for testing converting between any of 2 valid storage units
    for u in valid_storage_units:
        raw = random.randint(0, 1000)
        raw_value = "{}{}".format(raw, u)
        rvc = ResourceValueConverter(raw_value, random.choice(storage_names))
        assert rvc.raw == raw * literal_to_var[u], "{}, {}".format(rvc.raw, raw * literal_to_var[u])
        for u_conv in valid_storage_units:
            print("Converting {} to {}: {}".format(raw_value, u_conv, rvc.convert(u_conv)))
            assert rvc.convert(u_conv) == float(int(raw * literal_to_var[u])) / literal_to_var[u_conv]
