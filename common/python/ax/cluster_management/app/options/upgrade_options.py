# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse

from ax.cloud import Cloud
from ax.platform.component_config import SoftwareInfo
from .common import add_common_flags, add_software_info_flags, validate_software_info, \
    ClusterManagementOperationConfigBase


class ClusterUpgradeConfig(ClusterManagementOperationConfigBase):
    def __init__(self, cfg):
        super(ClusterUpgradeConfig, self).__init__(cfg)

        self.manifest_root = cfg.service_manifest_root
        self.bootstrap_config = cfg.platform_bootstrap_config
        self.force_upgrade = cfg.force_upgrade
        if cfg.software_version_info:
            # Read software info from config file
            self.target_software_info = SoftwareInfo(info_file=cfg.software_version_info)
        else:
            # Read software info from envs
            self.target_software_info = SoftwareInfo()

    def validate(self):
        all_errs = []
        all_errs += self._validate_critical_directories()

        # Because we have strict validation during installation, so we can assume
        # cluster has a valid name and cluster config
        if not self.cluster_name:
            all_errs.append("Please provide cluster name to pause the cluster")

        if self.cloud_provider not in Cloud.VALID_TARGET_CLOUD_INPUT:
            all_errs.append("Cloud provider {} not supported. Please choose from {}".format(
                self.cloud_provider, Cloud.VALID_TARGET_CLOUD_INPUT
            ))
        else:
            # Cloud singleton should be instantiated during validation stage so
            # we can ensure customer ID
            Cloud(target_cloud=self.cloud_provider)

        all_errs += validate_software_info(self.target_software_info)

        return all_errs


def add_upgrade_flags(parser):
    assert isinstance(parser, argparse.ArgumentParser)

    add_common_flags(parser)
    add_software_info_flags(parser)

    parser.add_argument("--force-upgrade", action="store_true", default=False, help="Always upgrade cluster, even if all software information are the same")
