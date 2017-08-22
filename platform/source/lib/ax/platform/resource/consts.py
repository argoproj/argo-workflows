#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
This file defines default constants and heuristics for us to infer cluster size
"""

from ax.util.const import GiB, MiB

# We assume 25 pods per compute unit. This is based on limited documentation
# and code comment from kubernetes about estimating master resource usage:
#
# http://blog.kubernetes.io/2016/07/kubernetes-updates-to-performance-and-scalability-in-1.3.html
# They proposed 30 pods per node, but we want to be a little bit more aggressive
DEFAULT_POD_PER_CU = 25

# Define per pod average resource
RESOURCE_PER_POD_STANDARD = {
    "cpu": 100,
    "memory": 250 * MiB
}

RESOURCE_PER_POD_COMPUTE = {
    "cpu": 200,
    "memory": 250 * MiB
}

CPU_UNIT_MILLICORE = "Millicore"
MEM_UNIT_BYTES = "Bytes"

# EC2 parameters we will probably use
EC2_PARAMS = {
    "m3.medium": {
        "cpu": 1000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 3.75 * GiB
    },
    "m3.large": {
        "cpu": 2000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 7.5 * GiB
    },
    "m3.xlarge": {
        "cpu": 4000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 15 * GiB
    },
    "m3.2xlarge": {
        "cpu": 8000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 30 * GiB
    },
    "r3.large": {
        "cpu": 2000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 15.25 * GiB
    },
    "r3.xlarge": {
        "cpu": 4000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 30.5 * GiB
    },
    "r3.2xlarge": {
        "cpu": 8000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 61 * GiB
    },
    "r3.4xlarge": {
        "cpu": 16000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 122 * GiB
    },
    "m4.large": {
        "cpu": 2000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 8 * GiB
    },
    "m4.xlarge": {
        "cpu": 4000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 16 * GiB
    },
    "m4.2xlarge": {
        "cpu": 8000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 32 * GiB
    },
    "m4.4xlarge": {
        "cpu": 16000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 64 * GiB
    },
    "c3.large": {
        "cpu": 2000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 3.75 * GiB
    },
    "c3.xlarge": {
        "cpu": 4000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 7.5 * GiB
    },
    "c3.2xlarge": {
        "cpu": 8000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 15 * GiB
    },
    "c3.4xlarge": {
        "cpu": 16000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 30 * GiB
    },
    "c3.8xlarge": {
        "cpu": 32000,
        "cpu_unit": CPU_UNIT_MILLICORE,
        "mem_unit": MEM_UNIT_BYTES,
        "memory": 60 * GiB
    },
}
