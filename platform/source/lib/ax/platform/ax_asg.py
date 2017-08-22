#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging
import sys
import time
from ax.util.singleton import Singleton
import boto3
from future.utils import with_metaclass
from retrying import retry

from ax.cloud.aws import default_aws_retry, ASGInstanceLifeCycle

logger = logging.getLogger(__name__)
logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S", stream=sys.stdout, level=logging.INFO)
logging.getLogger('boto3').setLevel(logging.WARNING)
logging.getLogger('botocore').setLevel(logging.WARNING)

ASG_STATE_POLL_INTERVAL = 10


class AXUserASGManager(with_metaclass(Singleton, object)):
    def __init__(self, cluster_name_id, region, aws_profile=None):
        self.cluster_name_id = cluster_name_id
        self.client = boto3.Session(profile_name=aws_profile, region_name=region).client("autoscaling")
        self.asgs = {}
        self.asg_name_to_tags = {}
        self.discover_asgs()

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def discover_asgs(self):
        parsed_all_asgs = False
        response = self.client.describe_auto_scaling_groups()
        while parsed_all_asgs is False:
            assert response is not None, "Not response returned for auto-scaling-groups"
            groups = response["AutoScalingGroups"]

            for asg in groups:
                tag_dict = {}
                try:
                    tags = asg["Tags"]
                    for tag in tags:
                        tag_dict[tag["Key"]] = tag["Value"]
                    self.asg_name_to_tags[asg["AutoScalingGroupName"]] = tag_dict
                    if tag_dict["KubernetesCluster"] == self.cluster_name_id:
                        if "AXTier" in tag_dict and tag_dict["AXTier"] == "user":
                            # AXTier flag will be present in new clusters. If so, identify the ASG with spot instances and on-demand instances.
                            if asg["AutoScalingGroupName"].endswith("spot"):
                                self.asgs["axuser_spot_asg"] = asg
                            elif asg["AutoScalingGroupName"].endswith("on-demand"):
                                self.asgs["axuser_on_demand_asg"] = asg
                            elif asg["AutoScalingGroupName"].endswith("variable"):
                                self.asgs["axuser_variable"] = asg
                            else:
                                logger.error("Discovered unknown ASG: %s", asg["AutoScalingGroupName"])
                        elif "AXTier" in tag_dict and tag_dict["AXTier"] == "applatix":
                            self.asgs["axsys"] = asg
                except KeyError:
                    pass

            next_token = response["NextToken"] if "NextToken" in response else None
            if next_token is None:
                parsed_all_asgs = True
            else:
                response = self.client.describe_auto_scaling_groups(NextToken=next_token)

        # There should be at least 2 ASGs.
        assert len(self.asgs) >= 2, "Minimum two ASGs should exist: " + str(self.asgs)
        assert "axuser_on_demand_asg" in self.asgs.keys(), "Ax-user-on-demand ASG not found!"
        logger.debug("Found ASGS {}".format([x["AutoScalingGroupName"] for x in self.asgs.values()]))

    def get_spot_asg(self):
        return self.asgs.get("axuser_spot_asg", None)

    def get_on_demand_asg(self):
        return self.asgs.get("axuser_on_demand_asg", None)

    def get_variable_asg(self):
        return self.asgs.get("axuser_variable", None)

    def get_all_asgs(self):
        return self.asgs.values()

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def set_asg_spec(self, name, minsize, maxsize, desired=-1):
        if desired < 0:
            self.client.update_auto_scaling_group(AutoScalingGroupName=name, MinSize=minsize, MaxSize=maxsize)
        else:
            self.client.update_auto_scaling_group(AutoScalingGroupName=name, MinSize=minsize, MaxSize=maxsize,
                                                  DesiredCapacity=desired)

    def wait_for_desired_asg_state(self):
        """
        This function waits for all ASGs to reach their desired states:
            - If DesiredCapacity is 0, it waits for all ASGs to be in "terminated" state
            - If DesiredCapacity is not 0, it waits for DesiredCapacity number of instances to reach "in-service" state
        :return:
        """
        while True:
            self.discover_asgs()
            all_asgs = self.get_all_asgs()
            all_asg_ready = True
            for asg in all_asgs:
                asg_state = "\nASG {}; Desired Capacity {};".format(asg["AutoScalingGroupName"], asg["DesiredCapacity"])
                asg_state += "\nInstances:"
                if asg["DesiredCapacity"] == 0:
                    # Mark all_asg_ready to False when ANY of its instances is NOT in "terminated" state
                    for i in asg["Instances"]:
                        asg_state += "\n{}: {}".format(i["InstanceId"], i["LifecycleState"])
                        if i["LifecycleState"] != ASGInstanceLifeCycle.Terminated:
                            all_asg_ready = False
                else:
                    # Mark all_asg_ready to False when NOT desired number of instances are in "in-service" state
                    num_in_service = 0
                    for i in asg["Instances"]:
                        asg_state += "\n{}: {}".format(i["InstanceId"], i["LifecycleState"])
                        num_in_service = num_in_service + 1 if i["LifecycleState"] == ASGInstanceLifeCycle.InService else num_in_service

                    if num_in_service != asg["DesiredCapacity"]:
                        all_asg_ready = False
                logger.info(asg_state)
            if not all_asg_ready:
                logger.info("Not all auto scaling groups are in desired states")
                time.sleep(ASG_STATE_POLL_INTERVAL)
            else:
                logger.info("All auto scaling groups are in desired states")
                return

    def pause_asg(self, name):
        self.set_asg_spec(name, 0, 0, 0)
