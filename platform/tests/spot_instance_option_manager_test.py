# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""The file has unit tests for the SpotInstanceOptionManager."""

import mock
import pytest
import unittest

from ax.platform.cluster_config import SpotInstanceOption
from ax.platform.minion_manager import SpotInstanceOptionManager
import boto3
from bunch import bunchify
from moto import mock_autoscaling, mock_sts


class SpotInstanceOptionManagerTest(unittest.TestCase):
    """
    Tests for the SpotInstanceOptionManager.
    """
    cluster_name = "cluster"
    cluster_id = "6883161e-d211-11e6-bc1d-c0ffeec0ffee"
    cluster_name_id = cluster_name + "-" + cluster_id
    asg_names = []
    for asg in [cluster_name + "-user-on-demand", cluster_name + "-user-spot", cluster_name + "-user-variable"]:
        asg_names.append(asg)

    session = boto3.Session(region_name="us-west-2")
    autoscaling = session.client("autoscaling")

    def create_mock_asg(self, asg_name):
        lc_name = asg_name + "-lc"
        response = self.autoscaling.create_launch_configuration(
            LaunchConfigurationName=lc_name, ImageId='ami-f00bad',
            KeyName='kubernetes-some-key')
        r = bunchify(response)
        assert r.ResponseMetadata.HTTPStatusCode == 200

        response = self.autoscaling.create_auto_scaling_group(
            AutoScalingGroupName=asg_name, LaunchConfigurationName=lc_name,
            MinSize=3, MaxSize=3, DesiredCapacity=3,
            Tags=[{'ResourceId': self.cluster_name_id, 'Key': 'KubernetesCluster', 'Value': self.cluster_name_id},
                  {'ResourceId': self.cluster_name_id, 'Key': 'AXTier', 'Value': 'user'}]
        )
        r = bunchify(response)
        assert r.ResponseMetadata.HTTPStatusCode == 200

    def setup_test(self):
        for asg in self.asg_names:
            self.create_mock_asg(asg)

        return SpotInstanceOptionManager(self.cluster_name_id, "us-west-2")

    @mock_autoscaling
    @mock_sts
    def test_option_to_asgs(self):
        spot_option_mgr = self.setup_test()
        assert set(spot_option_mgr.option_to_asgs(SpotInstanceOption.ALL_SPOT)) == set(self.asg_names)

        partial_asg = spot_option_mgr.option_to_asgs(SpotInstanceOption.PARTIAL_SPOT)
        assert len(partial_asg) == 1, "Too many ASGs returned for PARTIAL_SPOT"
        assert partial_asg[0] == self.cluster_name + "-user-variable"

        no_spot = spot_option_mgr.option_to_asgs(SpotInstanceOption.NO_SPOT)
        assert len(no_spot) == 0

    @mock_autoscaling
    @mock_sts
    def test_asgs_to_option(self):
        spot_option_mgr = self.setup_test()
        assert spot_option_mgr.asgs_to_option(self.asg_names) == SpotInstanceOption.ALL_SPOT
        assert spot_option_mgr.asgs_to_option([self.cluster_name + "-user-variable"]) == SpotInstanceOption.PARTIAL_SPOT
        assert spot_option_mgr.asgs_to_option([]) == SpotInstanceOption.NO_SPOT
