#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


class AXClusterType:
    STANDARD = "standard"
    COMPUTE = "compute"
    VALID_CLUSTER_TYPES = [STANDARD, COMPUTE]


class AXClusterSize:
    CLUSTER_MVC = "mvc"
    CLUSTER_SMALL = "small"
    CLUSTER_MEDIUM = "medium"
    CLUSTER_LARGE = "large"
    CLUSTER_XLARGE = "xlarge"
    CLUSTER_USER_PROVIDED = "user_provided"

    # TODO (#36): mvc is currently broken so mark it as not valid
    VALID_CLUSTER_SIZES = [CLUSTER_SMALL, CLUSTER_MEDIUM, CLUSTER_LARGE, CLUSTER_XLARGE]


class SpotInstanceOption:
    NO_SPOT = "none"
    PARTIAL_SPOT = "partial"
    ALL_SPOT = "all"
    VALID_SPOT_INSTANCE_OPTIONS = [NO_SPOT, PARTIAL_SPOT, ALL_SPOT]

class ClusterProvider:
    ARGO = "argo"
    USER = "user"
    VALID_CLUSTER_PROVIDERS = [ARGO, USER]
