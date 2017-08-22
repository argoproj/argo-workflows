#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Upgrades cluster_config in S3 so that it has all required fields.
"""

import logging
import os
import yaml

from ax.cloud.aws import AMI
from ax.platform.cluster_config import AXClusterConfig, AXClusterType
from ax.platform.ax_cluster_info import AXClusterInfo

logger = logging.getLogger(__name__)


class ClusterConfigUpgrade(object):
    def __init__(self, profile, region, cluster_name_id):
        self._profile = profile
        self._region = region
        self._cluster_name_id = cluster_name_id

    def update_cluster_config(self):
        """
        Upgrade the cluster config in S3 such that it has all required fields.
        """
        logger.info("Updating cluster config!")
        cluster_config = AXClusterConfig(cluster_name_id=self._cluster_name_id, aws_profile=self._profile)
        cluster_info = AXClusterInfo(cluster_name_id=self._cluster_name_id, aws_profile=self._profile)

        # Separate axsys / axuser config if needed
        update_node_config_key_needed = False
        try:
            # New cluster config is looking for "max_node_count" for this method and
            # should throw KeyError if the cluster config in s3 was the old one
            cluster_config.get_max_node_count()
        except KeyError:
            update_node_config_key_needed = True

        if update_node_config_key_needed:
            logger.info("Updating node config keys ...")
            # Parse old raw config directly
            minion_type = cluster_config._conf["cloud"]["configure"]["minion_type"]
            max_count = cluster_config._conf["cloud"]["configure"]["max_count"]
            min_count = cluster_config._conf["cloud"]["configure"]["min_count"]
            axsys_count = cluster_config._conf["cloud"]["configure"]["axsys_nodes"]

            # Remove all old keys
            for old_key in ["minion_type", "max_count", "min_count", "axsys_nodes"]:
                cluster_config._conf["cloud"]["configure"].pop(old_key, None)

            # Setting new keys
            cluster_config._conf["cloud"]["configure"]["axsys_node_count"] = axsys_count
            cluster_config._conf["cloud"]["configure"]["max_node_count"] = max_count
            cluster_config._conf["cloud"]["configure"]["min_node_count"] = min_count

            # All clusters that needs this upgrade has same node type for axsys and axuser
            cluster_config._conf["cloud"]["configure"]["axuser_node_type"] = minion_type
            cluster_config._conf["cloud"]["configure"]["axsys_node_type"] = minion_type
        else:
            logger.info("Node config keys are already up-to-date")

        # If cluster type is not set, default it to standard type
        if cluster_config.get_ax_cluster_type() == None:
            cluster_config._conf["cloud"]["configure"]["cluster_type"] = AXClusterType.STANDARD

        # Check and update Cluster user. Defaults to "customer"
        if cluster_config.get_ax_cluster_user() is None:
            cluster_config.set_ax_cluster_user('customer')

        # Check and update Cluster size. Defaults to "small"
        if cluster_config.get_ax_cluster_size() is None:
            max_count = cluster_config.get_max_node_count()
            if max_count == 5:
                cluster_size = "small"
            elif max_count == 10:
                cluster_size = "medium"
            elif max_count == 21:
                cluster_size = "large"
            elif max_count == 30:
                cluster_size = "xlarge"
            else:
                cluster_size = "small"
            cluster_config.set_ax_cluster_size(cluster_size)

        # Check and update AX Volume size. Note that this has to come *AFTER* the cluster_size is set.
        if cluster_config.get_ax_vol_size() is None:
            cluster_size = cluster_config.get_ax_cluster_size()
            if cluster_size in ("small", "medium"):
                vol_size = 100
            elif cluster_size == "large":
                vol_size = 200
            elif cluster_size == "xlarge":
                vol_size = 400
            else:
                vol_size = 100
            cluster_config.set_ax_vol_size(vol_size)

        # Ensure that we have 3 tiers now
        cluster_config.set_node_tiers("master/applatix/user")

        # set new ami id
        ami_name = os.getenv("AX_AWS_IMAGE_NAME")
        ami_id = AMI(aws_region=self._region, aws_profile=self._profile).get_ami_id_from_name(ami_name=ami_name)
        logger.info("Updating cluster config with ami %s", ami_id)
        cluster_config.set_ami_id(ami_id)

        cluster_config.save_config()
