#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import unittest

from ax.platform.ax_asg import AXUserASGManager
import boto3
from bunch import bunchify
from moto import mock_autoscaling, mock_sts

class AXUserASGManagerTest(unittest.TestCase):
    """
    Tests for AXUserASGManager.
    """

    cluster_name = "cluster"
    cluster_id = "6883161e-d211-11e6-bc1d-c0ffeec0ffee"
    cluster_name_id = cluster_name + "-" + cluster_id

    session = boto3.Session(region_name="us-west-2")
    autoscaling = session.client("autoscaling")

    def create_mock_autoscaling(self, asg_name):
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

    def mock_setup(self):
        for asg in [self.cluster_name + "-user-on-demand", self.cluster_name + "-user-spot", self.cluster_name + "-user-variable"]:
            self.create_mock_autoscaling(asg)

    def ax_asg_helper(self):
        self.mock_setup()
        return AXUserASGManager(self.cluster_name_id, 'us-west-2')

    @mock_autoscaling
    @mock_sts
    def test_as_asg_singleton(self):
        asg_manager = self.ax_asg_helper()
        id_1 = id(asg_manager)

        asg_manager = self.ax_asg_helper()
        id_2 = id(asg_manager)

        asg_manager = self.ax_asg_helper()
        id_3 = id(asg_manager)

        assert id_1 == id_2 == id_3

    def ax_asg_checker(self, asg_manager):
        asg = bunchify(asg_manager.get_on_demand_asg())
        assert asg.AutoScalingGroupName == self.cluster_name + "-user-on-demand"

        asg = bunchify(asg_manager.get_spot_asg())
        assert asg.AutoScalingGroupName == self.cluster_name + "-user-spot"

        asg = bunchify(asg_manager.get_variable_asg())
        assert asg.AutoScalingGroupName == self.cluster_name + "-user-variable"
        assert len(asg_manager.asg_name_to_tags) == 3

    @mock_autoscaling
    @mock_sts
    def test_ax_asg(self):
        """
        Tests that the AXUserASGManager correctly returns the correct ASGs.
        """
        asg_manager = self.ax_asg_helper()
        self.ax_asg_checker(asg_manager)

    @mock_autoscaling
    @mock_sts
    def test_ax_asg_name(self):
        """
        Tests that the AXUserASGManager identifies ASGs correctly.
        """
        asg_manager = self.ax_asg_helper()

        # Create more ASGS where "spot", "on-demand" and "variable" strings are at the beginning.
        for asg in ["on-demand-" + self.cluster_name, "spot-" + self.cluster_name, "variable-" + self.cluster_name]:
            self.create_mock_autoscaling(asg)

        # Only ASGs with trailing "spot", "on-demand" and "variable" should be used.
        self.ax_asg_checker(asg_manager)
