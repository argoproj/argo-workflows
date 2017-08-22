#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import boto3
from retrying import retry

from .util import default_aws_retry


class ASGInstanceLifeCycle(object):
    Pending = "Pending"
    PendingWait = "Pending:Wait"
    PendingProceed = "Pending:Proceed"
    Quarantined = "Quarantined"
    InService = "InService"
    Terminating = "Terminating"
    TerminatingWait = "Terminating:Wait"
    TerminatingProceed = "Terminating:Proceed"
    Terminated = "Terminated"
    Detaching = "Detaching"
    Detached = "Detached"
    EnteringStandby = "EnteringStandby"
    StandBy = "Standby"


class ASG(object):
    """
    Module for ASG management.
    """
    def __init__(self, name, profile=None, region=None):
        self._name = name
        self._profile = profile
        self._region = region
        self._asg = boto3.Session(profile_name=profile, region_name=region).client("autoscaling")

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def get_asg_with_tags(self, tag_key, tag_values):
        """
        Return list of all ASGs with specified tag key and values.
        """
        assert isinstance(tag_values, list)
        filters = [{"Name": "value", "Values": tag_values}]
        tags = self._asg.describe_tags(Filters=filters)["Tags"]
        ret = []
        for t in tags:
            if t["Key"] == tag_key:
                ret += [t["ResourceId"]]
        return ret

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def get_launch_config(self):
        """
        Return my own launch config name.
        """
        asg = self._asg.describe_auto_scaling_groups(AutoScalingGroupNames=[self._name])
        return asg["AutoScalingGroups"][0]["LaunchConfigurationName"]

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def set_launch_config(self, lc_name):
        """
        Set my own launch config.
        """
        self._asg.update_auto_scaling_group(AutoScalingGroupName=self._name, LaunchConfigurationName=lc_name)
