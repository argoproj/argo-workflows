#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module for AWS network management
"""

import logging

import boto3
from retrying import retry

from ax.util.ax_network_address import find_fitting_subnet

logger = logging.getLogger(__name__)


class AWSNetwork(object):
    def __init__(self, vpc_id, aws_profile=None, region_name=None):
        if aws_profile is None:
            self._ec2_client = boto3.Session(region_name=region_name).client("ec2")
        else:
            self._ec2_client = boto3.Session(profile_name=aws_profile, region_name=region_name).client("ec2")
        self._vpc_id = vpc_id

    @retry(stop_max_attempt_number=3, wait_fixed=2000)
    def find_subnet(self, size):
        vpc_cidr = self._ec2_client.describe_vpcs(VpcIds=[self._vpc_id])["Vpcs"][0]["CidrBlock"]
        subnets = self._ec2_client.describe_subnets(Filters=[{"Name": "vpc-id", "Values": [self._vpc_id]}])["Subnets"]
        cidrs = [net["CidrBlock"] for net in subnets]
        return find_fitting_subnet(vpc_cidr, cidrs, size)
