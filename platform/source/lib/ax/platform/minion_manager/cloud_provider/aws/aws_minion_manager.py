#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""Minion Manager implementation for AWS."""

import base64
import copy
from datetime import datetime
import logging
import os
from random import randint
import sys
from threading import Lock, Timer, Thread
import time

from ax.kubernetes.client import KubernetesApiClient
from ax.platform.minion_manager.cloud_provider.aws.aws_bid_advisor import AWSBidAdvisor
from ax.platform.minion_manager.cloud_provider.aws.minion_monitor import AWSMinionMonitor
from ax.platform.minion_manager.cloud_provider.aws.price_reporter import AWSPriceReporter
from ax.util.const import SECONDS_PER_MINUTE, SECONDS_PER_HOUR
import boto3
from botocore.exceptions import ClientError
from bunch import bunchify
from flask import Flask, jsonify, request
from future.utils import with_metaclass
import pytz
from retrying import retry

from ..base import MinionManagerBase
from .asg_mm import AWSAutoscalinGroupMM


logger = logging.getLogger("aws.minion-manager")
logging.getLogger('boto3').setLevel(logging.WARNING)
logging.getLogger('botocore').setLevel(logging.WARNING)
logging.getLogger('requests').setLevel(logging.WARNING)

MM_CONFIG_MAP_NAME = "minion-manager-config"
MM_CONFIG_MAP_NAMESPACE = "kube-system"

class AWSMinionManager(MinionManagerBase):
    """
    This class implements the minion-manager functionality for AWS.
    """

    def __init__(self, scaling_groups, region, **kwargs):
        super(AWSMinionManager, self).__init__(scaling_groups, region)
        aws_profile = kwargs.get("aws_profile", None)
        if aws_profile:
            boto_session = boto3.Session(region_name=region,
                                         profile_name=aws_profile)
        else:
            boto_session = boto3.Session(region_name=region)
        self._ac_client = boto_session.client('autoscaling')
        self._ec2_client = boto_session.client('ec2')

        self._asg_lock = Lock()
        self._asg_metas = []
        self.instance_type = None

        self.on_demand_kill_threads = {}

        self.bid_advisor = AWSBidAdvisor(
            on_demand_refresh_interval=4 * SECONDS_PER_HOUR,
            spot_refresh_interval=15 * SECONDS_PER_MINUTE, region=region)

        self.price_reporter = AWSPriceReporter(self._ec2_client,
                                               self.bid_advisor, self)

        monitor_minions = kwargs.get("monitor_minions", True)
        if monitor_minions:
            self.minion_monitor = AWSMinionMonitor(self._ec2_client)
        else:
            self.minion_monitor = None

        # The rest_thread is the Thread that responds to REST endpoints.
        self.rest_thread = Thread(target=self.rest_api,
                                  name="MinionManagerRestAPI")
        self.rest_thread.setDaemon(True)

    @staticmethod
    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def describe_asg_with_retries(ac_client, asgs):
        """
        AWS describe_auto_scaling_groups with retries.
        """
        response = ac_client.describe_auto_scaling_groups(
            AutoScalingGroupNames=asgs)
        return bunchify(response)

    @staticmethod
    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_instances_with_retries(ec2_client, instance_ids):
        """
        AWS describe_instances with retries.
        """
        response = ec2_client.describe_instances(
            InstanceIds=instance_ids)
        return bunchify(response)

    def discover_asgs(self):
        """ Query AWS and get metadata about all required ASGs. """
        response = AWSMinionManager.describe_asg_with_retries(
            self._ac_client, self._scaling_groups)
        with self._asg_lock:
            for asg in response.AutoScalingGroups:
                asg_mm = AWSAutoscalinGroupMM()
                asg_mm.set_asg_info(asg)
                self._asg_metas.append(asg_mm)
                logger.info("Added %s", asg.AutoScalingGroupName)

    def populate_current_config(self):
        """
        Queries AWS to get current bid_price for all ASGs and stores it
        in AWSAutoscalinGroupMM.
        """
        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
        def _describe_launch_configuration():
            response = self._ac_client.describe_launch_configurations(
                LaunchConfigurationNames=[asg.LaunchConfigurationName])
            assert len(response["LaunchConfigurations"]) == 1
            return bunchify(response).LaunchConfigurations[0]

        with self._asg_lock:
            for asg_meta in self._asg_metas:
                asg = asg_meta.asg_info

                # Get current launch configuration.
                launch_config = _describe_launch_configuration()
                asg_meta.set_lc_info(launch_config)
                bid_info = {}
                if "SpotPrice" in launch_config.keys():
                    bid_info["type"] = "spot"
                    bid_info["price"] = launch_config.SpotPrice
                else:
                    bid_info["type"] = "on-demand"
                asg_meta.set_bid_info(bid_info)
                logger.info("ASG %s using launch-config %s with bid-info %s",
                            asg.AutoScalingGroupName,
                            launch_config.LaunchConfigurationName, bid_info)

    def start(self):
        try:
            # Discover and populate the correct ASGs.
            self.discover_asgs()
            self.populate_current_config()
        except Exception as ex:
            raise Exception("Failed to discover/populate current ASG info: " +
                            str(ex))

    def update_needed(self, asg_meta):
        """ Checks if an ASG needs to be updated to use spot-instances. """
        try:
            bid_info = asg_meta.get_bid_info()
            if bid_info["type"] == "on-demand":
                logger.info("ASG %s needs to be updated", asg_meta.get_name())
                return True

            assert bid_info["type"] == "spot"
            if self.check_scaling_group_instances(asg_meta):
                # Desired # of instances running. No updates needed.
                logger.info("ASG %s does not need to be updated",
                            asg_meta.get_name())
                return False
            else:
                # Desired # of instances are not running.
                logger.info("ASG %s needs to be updated", asg_meta.get_name())
                return True
        except Exception as ex:
            logger.error("Failed while checking minions in %s: %s",
                         asg_meta.get_name(), str(ex))
            return False

    def are_bids_equal(self, cur_bid_info, new_bid_info):
        """
        Returns True if the new bid_info is the same as the current one.
        False otherwise.
        """
        if cur_bid_info["type"] != new_bid_info["type"]:
            return False
        # If you're here, it means that the bid types are equal.
        if cur_bid_info["type"] == "on-demand":
            return True

        if cur_bid_info["price"] == new_bid_info["price"]:
            return True

        return False

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def create_lc_with_spot(self, new_lc_name, launch_config, spot_price):
        """ Creates a launch-config for using spot-instances. """
        try:
            response = self._ac_client.create_launch_configuration(
                LaunchConfigurationName=new_lc_name,
                ImageId=launch_config.ImageId,
                KeyName=launch_config.KeyName,
                SecurityGroups=launch_config.SecurityGroups,
                ClassicLinkVPCSecurityGroups=launch_config.
                ClassicLinkVPCSecurityGroups,
                UserData=base64.b64decode(launch_config.UserData),
                InstanceType=launch_config.InstanceType,
                BlockDeviceMappings=launch_config.BlockDeviceMappings,
                InstanceMonitoring=launch_config.InstanceMonitoring,
                SpotPrice=spot_price,
                IamInstanceProfile=launch_config.IamInstanceProfile,
                EbsOptimized=launch_config.EbsOptimized,
                AssociatePublicIpAddress=launch_config.
                AssociatePublicIpAddress)
            assert response is not None, \
                "Failed to create launch-config {}".format(new_lc_name)
            assert response["HTTPStatusCode"] == 200, \
                "Failed to create launch-config {}".format(new_lc_name)
            logger.info("Created LaunchConfig for spot instances: %s",
                        new_lc_name)
        except ClientError as ce:
            if "AlreadyExists" in str(ce):
                logger.info("LaunchConfig %s already exists. Reusing it.",
                            new_lc_name)
                return
            raise ce

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def create_lc_on_demand(self, new_lc_name, launch_config):
        """ Creates a launch-config for using on-demand instances. """
        try:
            response = self._ac_client.create_launch_configuration(
                LaunchConfigurationName=new_lc_name,
                ImageId=launch_config.ImageId,
                KeyName=launch_config.KeyName,
                SecurityGroups=launch_config.SecurityGroups,
                ClassicLinkVPCSecurityGroups=launch_config.
                ClassicLinkVPCSecurityGroups,
                UserData=base64.b64decode(launch_config.UserData),
                InstanceType=launch_config.InstanceType,
                BlockDeviceMappings=launch_config.BlockDeviceMappings,
                InstanceMonitoring=launch_config.InstanceMonitoring,
                IamInstanceProfile=launch_config.IamInstanceProfile,
                EbsOptimized=launch_config.EbsOptimized,
                AssociatePublicIpAddress=launch_config.
                AssociatePublicIpAddress)
            assert response is not None, \
                "Failed to create launch-config {}".format(new_lc_name)
            assert response["HTTPStatusCode"] == 200, \
                "Failed to create launch-config {}".format(new_lc_name)
            logger.info("Created LaunchConfig for on-demand instances: %s",
                        new_lc_name)
        except ClientError as ce:
            if "AlreadyExists" in str(ce):
                logger.info("LaunchConfig %s already exists. Reusing it.",
                            new_lc_name)
                return
            raise ce

    def update_scaling_group(self, asg_meta, new_bid_info):
        """
        Updates the AWS AutoScalingGroup. Makes the next_bid_info as the new
        bid_info.
        """
        logger.info("Updating ASG: %s, Bid: %s", asg_meta.get_name(),
                    new_bid_info)
        launch_config = asg_meta.get_lc_info()

        orig_launch_config_name = launch_config.LaunchConfigurationName
        assert new_bid_info.get("type", None) is not None, \
            "Bid info has no bid type"
        if new_bid_info["type"] == "spot":
            spot_price = new_bid_info["price"]
        else:
            spot_price = None
        logger.info("ASG( %s ): New bid price %s", asg_meta.get_name(),
                    spot_price)

        if launch_config.LaunchConfigurationName[-2:] == "-0":
            new_lc_name = launch_config.LaunchConfigurationName[:-2]
        else:
            new_lc_name = launch_config.LaunchConfigurationName + "-0"
        logger.info("ASG( %s ): New launch-config name: %s",
                    asg_meta.get_name(), new_lc_name)

        if spot_price is None:
            self.create_lc_on_demand(new_lc_name, launch_config)
        else:
            self.create_lc_with_spot(new_lc_name, launch_config, spot_price)

        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
        def _update_asg_in_aws(asg_name, launch_config_name):
            self._ac_client.update_auto_scaling_group(
                AutoScalingGroupName=asg_name,
                LaunchConfigurationName=launch_config_name)
            logger.info("Updated ASG %s with new LaunchConfig: %s",
                        asg_name, launch_config_name)

        _update_asg_in_aws(asg_meta.get_name(), new_lc_name)

        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
        def _delete_launch_config(lc_name):
            self._ac_client.delete_launch_configuration(
                LaunchConfigurationName=lc_name)
            logger.info("Deleted launch-configuration %s", lc_name)

        _delete_launch_config(orig_launch_config_name)

        # Update asg_meta.
        launch_config.LaunchConfigurationName = new_lc_name
        if spot_price is None:
            launch_config.pop('SpotPrice', None)
        else:
            launch_config['SpotPrice'] = spot_price
        asg_meta.set_lc_info(launch_config)
        asg_meta.set_bid_info(new_bid_info)

        logger.info("Updated ASG %s, new launch-config %s, bid-info %s",
                    asg_meta.get_name(), launch_config.LaunchConfigurationName,
                    new_bid_info)
        return

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def run_or_die(self, instance_id, zone, instance_type, asg_meta):
        """
        Terminates the given "on-demand" instance if the current bid
        is "spot".
        """
        bid_info = self.bid_advisor.get_new_bid(zone, instance_type)
        try:
            if bid_info["type"] == "spot":
                self._ec2_client.terminate_instances(InstanceIds=[instance_id])
                logger.info("Terminated instance %s", instance_id)
                # Remove this instance from asg_meta.instance_info
                asg_meta.remove_instance(instance_id)
                logger.info("Removed terminated instance %s", instance_id)
            else:
                logger.info("Continuing to run %s", instance_id)
        finally:
            self.on_demand_kill_threads.pop(instance_id)

    def schedule_instance_termination(self, asg_meta):
        """
        Checks whether any of the given instances are "on-demand" and schedules
        their termination.
        """
        instances = asg_meta.get_instances()
        if len(instances) == 0:
            return

        for instance in instances:
            # On-demand instances don't have the InstanceLifecycle field in
            # their responses. Spot instances have InstanceLifecycle=spot.
            if 'InstanceLifecycle' not in instance:
                launch_time = instance.LaunchTime
                current_time = datetime.utcnow().replace(tzinfo=pytz.utc)
                elapsed_seconds = (current_time - launch_time). \
                    total_seconds()

                # If the instance is running for hours, only the seconds in
                # the current hour need to be used.
                elapsed_seconds_in_hour = elapsed_seconds % \
                    SECONDS_PER_HOUR
                # Start a thread that will check whether the instance
                # should continue running ~40 minutes later.
                seconds_before_check = abs((40.0 + randint(0, 19)) *
                                           SECONDS_PER_MINUTE -
                                           elapsed_seconds_in_hour)
                instance_id = instance.InstanceId
                if instance_id in self.on_demand_kill_threads.keys():
                    continue
                logger.info("Scheduling thread for %s after %s seconds",
                            instance_id, seconds_before_check)
                args = [instance_id, instance.Placement.AvailabilityZone,
                        instance.InstanceType, asg_meta]
                timed_thread = Timer(seconds_before_check, self.run_or_die,
                                     args=args)
                timed_thread.setDaemon(True)
                logger.info("Added instance %s to kill threads", instance_id)
                self.on_demand_kill_threads[instance_id] = timed_thread
                timed_thread.start()

        return

    def populate_instances(self, asg_meta):
        """ Populates info about all instances running in the given ASG. """
        asg_name = asg_meta.get_name()
        assert asg_name is not None, "No ASG name specified"

        response = AWSMinionManager.describe_asg_with_retries(
            self._ac_client, [asg_name])
        instance_ids = []
        asg = response.AutoScalingGroups[0]

        # if there are no instances running in the ASG, return
        if asg.DesiredCapacity == 0:
            logger.info("Desired capacity for %s is 0", asg_name)
            return

        for instance in asg.Instances:
            instance_ids.append(instance.InstanceId)

        # If the DesiredCapacity > 0, there should be instances running in the ASG. However,
        # in cases where the spot-instance price has spiked or just before some instances are
        # about to start, it may happen that there are no instances in the ASG.
        if len(instance_ids) == 0:
            logger.info("No instances found in %s", asg_name)
            return

        response = self.get_instances_with_retries(self._ec2_client,
                                                   instance_ids)
        for resv in response.Reservations:
            asg_meta.add_instances(resv.Instances)

    def minion_manager_work(self):
        """ The main work for dealing with spot-instances happens here. """
        while True:
            try:
                # Iterate over all asgs and update them if needed.
                with self._asg_lock:
                    asgs = copy.copy(self._asg_metas)

                for asg_meta in asgs:
                    logger.info("Processing ASG: %s", asg_meta.get_name())

                    # Populate info. about all instance in the ASG
                    self.populate_instances(asg_meta)

                    # Check if any of these are on-demand instances that can
                    # be terminated.
                    self.schedule_instance_termination(asg_meta)

                    if not self.update_needed(asg_meta):
                        continue

                    # Currently, the minion-manager only works for a single AZ.
                    new_bid_info = self.bid_advisor.get_new_bid(
                        zone=asg_meta.asg_info.AvailabilityZones[0],
                        instance_type=asg_meta.lc_info.InstanceType)

                    # Update ASGs iff new bid is different from current bid.
                    if self.are_bids_equal(asg_meta.bid_info, new_bid_info):
                        logger.info("No change in bid info for %s",
                                    asg_meta.get_name())
                        continue
                    logger.info("Got new bid info from BidAdvisor: %s",
                                new_bid_info)
                    self.update_scaling_group(asg_meta, new_bid_info)
            except Exception as ex:
                logger.exception("Failed while checking instances in ASG: " +
                                 str(ex))
            finally:
                # Cooling off period.
                time.sleep(10 * SECONDS_PER_MINUTE)

    def check_scaling_group_instances(self, scaling_group):
        """
        Checks whether desired number of instances are running in an ASG.
        Also, schedules termination of "on-demand" instances.
        """
        asg_meta = scaling_group
        attempts_to_converge = 3
        while attempts_to_converge > 0:
            asg_info = asg_meta.get_asg_info()
            response = AWSMinionManager.describe_asg_with_retries(
                self._ac_client, [asg_info.AutoScalingGroupName])
            asg = response.AutoScalingGroups[0]

            if asg.DesiredCapacity <= len(asg.Instances):
                # The DesiredCapacity can be <= actual number of instances.
                # This can happen during scale down. The autoscaler may have
                # reduced the DesiredCapacity. But it can take sometime before
                # the instances are actually terminated. If this check happens
                # during that time, the DesiredCapacity may be < actual number
                # of instances.
                logger.info("Desired number of minions running.")
                return True
            else:
                # It is possible that the autoscaler may have just increased
                # the DesiredCapacity but AWS is still in the process of
                # spinning up new instances. To given enough time to AWS to
                # spin up these new instances (i.e. for the desired state and
                # actual state to converge), sleep for 1 minute and try again.
                # If the state doesn't converge even after retries, return
                # False.
                logger.info("Desired number of instances not running." +
                            "Desired %d, actual %d", asg.DesiredCapacity,
                            len(asg.Instances))
                attempts_to_converge = attempts_to_converge - 1

                # Wait for sometime before checking again.
                time.sleep(60)
        return False

    def get_asg_metas(self):
        """ Return a copy of all asg_metas"""
        asgs = None
        with self._asg_lock:
            asgs = copy.deepcopy(self._asg_metas)
        return asgs

    def rest_api(self):
        """ Thread that responds to the Flask api endpoints. """
        k8s_client = KubernetesApiClient()
        app = Flask("MinionManagerRestAPI")

        def _update_config_map(enabled_str, asgs):
            cmap = self.k8s_client.api.read_namespaced_config_map(
                namespace=MM_CONFIG_MAP_NAMESPACE, name=MM_CONFIG_MAP_NAME)
            cmap.data["MM_SPOT_INSTANCE_ENABLED"] = enabled_str
            if asgs:
                cmap.data["MM_SCALING_GROUPS"] = asgs

            k8s_client.api.replace_namespaced_config_map(
                cmap, MM_CONFIG_MAP_NAMESPACE, MM_CONFIG_MAP_NAME)

        @app.route('/spot_instance_config', methods=['PUT'])
        def _update_spot_instances():
            """ Update whether spot instances config. """
            enabled_str = request.args.get('enabled').title()
            assert enabled_str.lower() in ("true", "false")

            # Update the config-map first
            asgs = request.args.get('asgs', None)
            _update_config_map(enabled_str, asgs)
            if asgs:
                os.environ["MM_SCALING_GROUPS"] = asgs
                logger.info("Set MM_SCALING_GROUPS to %s", asgs)
                with self._asg_lock:
                    del self._asg_metas[:]
                self._scaling_groups = asgs.split()
                self.start()

            os.environ["MM_SPOT_INSTANCE_ENABLED"] = enabled_str
            logger.info("Set MM_SPOT_INSTANCE_ENABLED to %s", enabled_str)
            return jsonify({"status": "ok"})

        @app.route('/spot_instance_config', methods=['GET'])
        def _get_spot_instances():
            """ Get spot-instances config. """
            cmap = k8s_client.api.read_namespaced_config_map(
                namespace=MM_CONFIG_MAP_NAMESPACE, name=MM_CONFIG_MAP_NAME)
            return jsonify({"status": cmap.data["MM_SPOT_INSTANCE_ENABLED"], "asgs": cmap.data["MM_SCALING_GROUPS"]})

        app.run(host='0.0.0.0', port=6000)

    def run(self):
        """Entrypoint for the AWS specific minion-manager."""
        logger.info("Running AWS Minion Manager")

        self.start()

        self.bid_advisor.run()

        self.price_reporter.run()

        if self.minion_monitor:
            self.minion_monitor.run()

        self.rest_thread.start()
        self.minion_manager_work()
        return
