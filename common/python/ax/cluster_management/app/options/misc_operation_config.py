# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse

from ax.cloud import Cloud
from .common import add_common_flags, ClusterManagementOperationConfigBase


class ClusterMiscOperationConfig(ClusterManagementOperationConfigBase):
    def __init__(self, cfg):
        super(ClusterMiscOperationConfig, self).__init__(cfg)

    def validate(self):
        all_errs = []
        all_errs += self._validate_critical_directories()

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

        return all_errs


def add_misc_flags(parser):
    assert isinstance(parser, argparse.ArgumentParser)

    add_common_flags(parser)



