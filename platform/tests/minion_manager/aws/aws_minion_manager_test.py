"""The file has unit tests for the AWSMinionManager."""

import unittest
import mock
import pytest
from ax.platform.minion_manager.cloud_provider.aws.aws_minion_manager import AWSMinionManager
from ax.platform.minion_manager.cloud_provider.aws.aws_bid_advisor import AWSBidAdvisor
from moto import mock_autoscaling, mock_sts, mock_ec2
import boto3
from bunch import bunchify


class AWSMinionManagerTest(unittest.TestCase):
    """
    Tests for the AWSMinionManager.
    """
    cluster_name = "cluster"
    cluster_id = "abcd-c0ffeec0ffee"
    cluster_name_id = cluster_name + "-" + cluster_id
    asg_name = cluster_name_id + "-asg"
    lc_name = cluster_name_id + "-lc"

    session = boto3.Session(region_name="us-west-2")
    autoscaling = session.client("autoscaling")

    @mock_autoscaling
    @mock_sts
    def create_mock_asgs(self):
        """
        Creates mocked AWS resources.
        """
        response = self.autoscaling.create_launch_configuration(
            LaunchConfigurationName=self.lc_name, ImageId='ami-f00bad',
            SpotPrice="0.100", KeyName='kubernetes-some-key')
        resp = bunchify(response)
        assert resp.ResponseMetadata.HTTPStatusCode == 200

        response = self.autoscaling.create_auto_scaling_group(
            AutoScalingGroupName=self.asg_name,
            LaunchConfigurationName=self.lc_name, MinSize=3, MaxSize=3,
            DesiredCapacity=3,
            Tags=[{'ResourceId': self.cluster_name_id,
                   'Key': 'KubernetesCluster', 'Value': self.cluster_name_id}]
        )
        resp = bunchify(response)
        assert resp.ResponseMetadata.HTTPStatusCode == 200

    def basic_setup_and_test(self):
        """
        Creates the mock setup for tests, creates the aws_mm object and does
        some basic sanity tests before returning it.
        """
        self.create_mock_asgs()
        aws_mm = AWSMinionManager([self.asg_name], "us-west-2")
        assert len(aws_mm.get_asg_metas()) == 0, \
            "ASG Metadata already populated?"

        aws_mm.discover_asgs()
        assert aws_mm.get_asg_metas() is not None, "ASG Metadata not populated"

        for asg in aws_mm.get_asg_metas():
            assert asg.asg_info.AutoScalingGroupName == self.asg_name

        aws_mm.populate_current_config()
        return aws_mm

    @mock_autoscaling
    @mock_sts
    def test_discover_asgs(self):
        """
        Tests that the discover_asgs method works as expected.
        """
        self.basic_setup_and_test()

    @mock_autoscaling
    @mock_sts
    @mock_ec2
    def test_populate_instances(self):
        """
        Tests that existing info. about ASGs is populated correctly.
        """
        aws_mm = self.basic_setup_and_test()
        asg = aws_mm.get_asg_metas()[0]

        orig_instance_count = len(asg.get_instance_info())
        aws_mm.populate_instances(asg)
        assert len(asg.get_instance_info()) == orig_instance_count + 3

    @mock_autoscaling
    @mock_sts
    def test_populate_current_config(self):
        """
        Tests that existing instances are correctly populated by the
        populate_instances() method.
        """
        aws_mm = self.basic_setup_and_test()
        for asg_meta in aws_mm.get_asg_metas():
            assert asg_meta.get_lc_info().LaunchConfigurationName == \
                   self.lc_name
            assert asg_meta.get_bid_info()["type"] == "spot"
            assert asg_meta.get_bid_info()["price"] == "0.100"

    @mock_autoscaling
    @mock_sts
    @pytest.mark.skip(
        reason="Moto doesn't have some fields in it's LaunchConfig.")
    def test_update_cluster_spot(self):
        """
        Tests that the AWSMinionManager correctly creates launch-configs and
        updates the ASG.

        Note: Moto doesn't have the ClassicLinkVPCSecurityGroups and
        IamInstanceProfile fields in it's LaunchConfig. Running the test below
        required manually commenting out these fields in the call to
        create_launch_configuration :(
        """
        awsmm = self.basic_setup_and_test()
        bid_info = {}
        bid_info["type"] = "spot"
        bid_info["price"] = "10"
        awsmm.update_scaling_group(awsmm.get_asg_metas()[0], bid_info)

    @mock_autoscaling
    @mock_sts
    @pytest.mark.skip(
        reason="Moto doesn't have some fields in it's LaunchConfig.")
    def test_update_cluster_on_demand(self):
        """
        Tests that the AWSMinionManager correctly creates launch-configs and
        updates the ASG.

        Note: Moto doesn't have the ClassicLinkVPCSecurityGroups and
        IamInstanceProfile fields in it's LaunchConfig. Running the test below
        required manually commenting out these fields in the call to
        create_launch_configuration :(
        """
        awsmm = self.basic_setup_and_test()
        bid_info = {"type": "on-demand"}
        awsmm.update_scaling_group(awsmm.get_asg_metas()[0], bid_info)

    @mock_autoscaling
    @mock_sts
    @mock_ec2
    def test_update_needed(self):
        """
        Tests that the AWSMinionManager correctly checks if updates are needed.
        """
        awsmm = self.basic_setup_and_test()

        asg_meta = awsmm.get_asg_metas()[0]
        # Moto returns that all instances are running. No updates needed.
        assert awsmm.update_needed(asg_meta) is False
        bid_info = {"type": "on-demand"}
        asg_meta.set_bid_info(bid_info)

        assert awsmm.update_needed(asg_meta) is True

    @mock_autoscaling
    @mock_sts
    def test_bid_equality(self):
        """
        Tests that 2 bids are considered equal when their type and price match.
        Not equal otherwise.
        """
        a_bid = {}
        a_bid["type"] = "on-demand"
        b_bid = {}
        b_bid["type"] = "on-demand"
        b_bid["price"] = "100"
        awsmm = self.basic_setup_and_test()
        assert awsmm.are_bids_equal(a_bid, b_bid) is True

        # Change type of new bid to "spot".
        b_bid["type"] = "spot"
        assert awsmm.are_bids_equal(a_bid, b_bid) is False

        # Change the type of a_bid to "spot" but a different price.
        a_bid["type"] = "spot"
        a_bid["price"] = "90"
        assert awsmm.are_bids_equal(a_bid, b_bid) is False

        a_bid["price"] = "100"
        assert awsmm.are_bids_equal(a_bid, b_bid) is True

    @mock_autoscaling
    @mock_ec2
    @mock_sts
    def test_awsmm_instances(self):
        """
        Tests that the AWSMinionManager correctly tracks running instances.
        """
        awsmm = self.basic_setup_and_test()
        asg_meta = awsmm.get_asg_metas()[0]
        assert awsmm.check_scaling_group_instances(asg_meta)

        # Update the desired # of instances in the ASG. Verify that
        # minion-manager continues to account for the new instances.
        self.autoscaling.update_auto_scaling_group(
            AutoScalingGroupName=self.asg_name, MaxSize=4, DesiredCapacity=4)
        assert awsmm.check_scaling_group_instances(asg_meta)

    @mock_autoscaling
    @mock_ec2
    @mock_sts
    def test_instance_termination(self):
        """
        Tests that the AWSMinionManager schedules instance termination.
        """
        awsmm = self.basic_setup_and_test()
        assert len(awsmm.on_demand_kill_threads) == 0
        asg_meta = awsmm.get_asg_metas()[0]
        awsmm.populate_instances(asg_meta)

        assert len(asg_meta.get_instances()) == 3
        awsmm.schedule_instance_termination(asg_meta)
        assert len(awsmm.on_demand_kill_threads) == 3

        # For testing manually run the run_or_die method.
        instance_type = "m3.medium"
        zone = "us-west-2b"
        awsmm.bid_advisor.on_demand_price_dict[instance_type] = "100"
        awsmm.bid_advisor.spot_price_list = [{'InstanceType': instance_type,
                                              'SpotPrice': '80',
                                              'AvailabilityZone': zone}]
        awsmm.instance_type = instance_type
        dict_copy = awsmm.on_demand_kill_threads.copy()
        for key in dict_copy.keys():
            awsmm.run_or_die(key, zone, instance_type, asg_meta)
        assert len(awsmm.on_demand_kill_threads) == 0
        assert len(asg_meta.get_instances()) == 0

