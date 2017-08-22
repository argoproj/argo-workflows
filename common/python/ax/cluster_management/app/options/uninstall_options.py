# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse

from ax.cloud import Cloud
from ax.cloud.aws import EC2
from .common import add_common_flags, ClusterManagementOperationConfigBase


class ClusterUninstallConfig(ClusterManagementOperationConfigBase):
    def __init__(self, cfg):
        super(ClusterUninstallConfig, self).__init__(cfg)
        self.cloud_region = cfg.cloud_region
        self.cloud_placement = cfg.cloud_placement
        self.force_uninstall = cfg.force_uninstall

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
            if self.force_uninstall and self.cloud_region and self.cloud_placement:
                try:
                    # Validate placement only for AWS
                    c = Cloud(target_cloud=self.cloud_provider)
                    if c.target_cloud_aws():
                        ec2 = EC2(profile=self.cloud_profile, region=self.cloud_region)
                        zones = ec2.get_availability_zones()
                        if self.cloud_placement not in zones:
                            all_errs.append("Invalid cloud placement {}. Please choose from {}".format(
                                self.cloud_placement, zones
                            ))
                except Exception as e:
                    all_errs.append("Cloud provider validation error: {}".format(e))

        return all_errs


def add_uninstall_flags(parser):
    assert isinstance(parser, argparse.ArgumentParser)

    add_common_flags(parser)
    parser.add_argument("--cloud-region", default=None, help="Cluster's region, used only when force uninstall")
    parser.add_argument("--cloud-placement", default=None, help="Cluster's placement, used only when force uninstall")
    parser.add_argument("--force-uninstall", default=False, action="store_true", help="Uninstall the cluster without first gracefully shutting down the cluster.")
