#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module for minion upgrade.
Handles only launch config right now. Will add others later.
"""

import logging
import os
import zlib

from ax.cloud.aws import AMI
from ax.cloud.aws import ASG
from ax.cloud.aws import LaunchConfig
from ax.platform.cluster_config import AXClusterConfig
from ax.platform.kube_env_config import kube_env_update
from ax.platform.cluster_instance_profile import AXClusterInstanceProfile


logger = logging.getLogger(__name__)

AX_VOL_DISK = "/dev/xvdz"

# TODO (#254) clean up upgrade logics that are not needed.
class MinionUpgrade(object):
    def __init__(self,
                 new_kube_version,
                 new_cluster_install_version,
                 new_kube_server_hash,
                 new_kube_salt_hash,
                 profile,
                 region,
                 cluster_name_id,
                 ax_vol_disk_type):
        self._new_kube_version = new_kube_version
        self._new_cluster_install_version = new_cluster_install_version
        self._new_kube_server_hash = new_kube_server_hash
        self._new_kube_salt_hash = new_kube_salt_hash
        self._profile = profile
        self._region = region
        self._cluster_name_id = cluster_name_id
        self._ax_vol_disk_type = ax_vol_disk_type

    def update_all_launch_configs(self, retain_spot_price=False):
        """
        Upgrade all launch configurations for a cluster to new version.
        """
        asg_names = ASG(None, self._profile, self._region).get_asg_with_tags(tag_key="KubernetesCluster",
                                                                             tag_values=[self._cluster_name_id])
        for asg_name in asg_names:
            asg = ASG(asg_name, self._profile, self._region)
            new_lc_name = self._new_lc_name(asg_name)
            if new_lc_name:
                old_lc_name = asg.get_launch_config()
                if old_lc_name != new_lc_name:
                    self._update_launch_config(old_lc_name, new_lc_name, retain_spot_price=retain_spot_price)
                    asg.set_launch_config(new_lc_name)
                    LaunchConfig(old_lc_name, aws_profile=self._profile, aws_region=self._region).delete()

    def _update_launch_config(self, old_name, new_name, retain_spot_price=False):
        """
        Upgrade old_name launch config to new_name.
        Return OK if new launch config is already created.
        Raise error if both old and new don't exist.
        """
        logger.info("Converting launch config %s to %s ...", old_name, new_name)
        ami_name = os.getenv("AX_AWS_IMAGE_NAME")
        assert ami_name, "Fail to detect AMI name from environ"
        ami_id = AMI(aws_region=self._region, aws_profile=self._profile).get_ami_id_from_name(ami_name=ami_name)
        logger.info("Using ami %s for new minion launch configuration", ami_id)

        cluster_config = AXClusterConfig(cluster_name_id=self._cluster_name_id, aws_profile=self._profile)
        if LaunchConfig(new_name, aws_profile=self._profile, aws_region=self._region).get() is not None:
            # Launch config already updated, nop.
            logger.debug("New launch config %s already there. No creation.", new_name)
            return

        lc = LaunchConfig(old_name, aws_profile=self._profile, aws_region=self._region)
        config = lc.get()
        assert config is not None, "Empty old launch config and new launch config"
        user_data = config.pop("UserData")
        logger.debug("Existing launch config %s: %s", old_name, config)

        updates = {
            "new_kube_version": self._new_kube_version,
            "new_cluster_install_version": self._new_cluster_install_version,
            "new_kube_server_hash": self._new_kube_server_hash,
            "new_kube_salt_hash": self._new_kube_salt_hash,
        }

        # Replace ImageId and everything listed in default_kube_up_env.
        config["ImageId"] = ami_id
        config["IamInstanceProfile"] = AXClusterInstanceProfile(
            self._cluster_name_id, region_name=self._region, aws_profile=self._profile
        ).get_minion_instance_profile_name()
        user_data = zlib.decompressobj(32 + zlib.MAX_WBITS).decompress(user_data)
        user_data = kube_env_update(user_data, updates)
        comp = zlib.compressobj(9, zlib.DEFLATED, zlib.MAX_WBITS | 16)
        config["UserData"] = comp.compress(user_data) + comp.flush()

        # Add AX Volume device mappings.
        orig_block_devices = config.pop("BlockDeviceMappings")

        block_devices = []
        for device in orig_block_devices:
            if device["DeviceName"] != AX_VOL_DISK:
                block_devices.append(device)
        vol_device = {}
        vol_device["DeviceName"] = AX_VOL_DISK
        ebs_dict = {}
        ebs_dict["DeleteOnTermination"] = True
        ebs_dict["VolumeSize"] = cluster_config.get_ax_vol_size()
        ebs_dict["VolumeType"] = self._ax_vol_disk_type

        vol_device["Ebs"] = ebs_dict
        block_devices.append(vol_device)
        config["BlockDeviceMappings"] = block_devices
        logger.debug("New block device mappings: %s", config["BlockDeviceMappings"])

        lc.copy(new_name, config, retain_spot_price=retain_spot_price)
        logger.info("Converting launch config %s to %s ... DONE.", old_name, new_name)

    def _new_lc_name(self, asg_name):
        """
        Find out new launch config name based on cluster ASG name.
        """
        lc_name = ""
        if "minion-ax" in asg_name:
            template = "{name_id}-minion-ax-{region}-{kube_version}-{install_version}"
            lc_name = template.format(name_id=self._cluster_name_id,
                                      region=self._region,
                                      kube_version=self._new_kube_version,
                                      install_version=self._new_cluster_install_version)
        elif "minion-user-variable" in asg_name:
            template = "{name_id}-minion-user-variable-{kube_version}-{install_version}"
            lc_name = template.format(name_id=self._cluster_name_id,
                                      kube_version=self._new_kube_version,
                                      install_version=self._new_cluster_install_version)
        elif "minion-user" in asg_name and "on-demand" in asg_name:
            template = "{name_id}-minion-user-{region}-on-demand-{kube_version}-{install_version}"
            lc_name = template.format(name_id=self._cluster_name_id,
                                      region=self._region,
                                      kube_version=self._new_kube_version,
                                      install_version=self._new_cluster_install_version)
        elif "spot" in asg_name:
            logger.info("Ignoring old spot ASG %s", asg_name)
        else:
            assert 0, "Invalid ASG name {}".format(asg_name)
        return lc_name
