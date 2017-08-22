#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import boto3
import logging

from botocore.exceptions import ClientError
from retrying import retry

from .util import default_aws_retry, tag_dict_to_aws_filter

logger = logging.getLogger(__name__)


class EC2InstanceState(object):
    Pending = "pending"
    Running = "running"
    ShuttingDown = "shutting-down"
    Terminated = "terminated"
    Stopping = "stopping"
    Stopped = "stopped"


class EC2IPPermission(object):
    AllProtocols = "-1"
    AllIP = "0.0.0.0/0"
    TCP = "tcp"
    UDP = "udp"
    ICMP = "icmp"

    def __init__(self, protocol=None, from_port=None, to_port=None, cidr=None):
        """
        :param protocol: protocol string, e.g. "tcp", "udp", "icmp", etc. or "all"
        :param from_port: integer of port number
        :param to_port: integer of port number
        :param cidr: string of IP CIDR
        """
        self.protocol = protocol
        self.from_port = from_port
        self.to_port = to_port
        self.cidr = cidr

    def __repr__(self):
        return '{} protocol: "{}"; from_port: {}; to_port: {}; cidr: "{}"'.format(self.__class__, self.protocol,
                                                                                  self.from_port, self.to_port,
                                                                                  self.cidr)


class EC2(object):
    def __init__(self, profile=None, region=None):
        self._profile = profile
        self._region = region
        self._client = boto3.Session(profile_name=profile).client("ec2", region_name=region)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_availability_zones(self):
        return [z["ZoneName"] for z in self._client.describe_availability_zones()["AvailabilityZones"] if z["State"] == "available"]

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_routetables(self, tags=None):
        """
        :param tags:
        :return: list of route table information
        """
        filters = tag_dict_to_aws_filter(tags)
        rtb = self._client.describe_route_tables(
            Filters=filters
        )
        return rtb["RouteTables"]

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_instances(self, name=None, states=None, tags=None):
        """
        Get instances with assorted parameters
        :param name: Instance name
        :param states: list of instance states
        :param tags: dictionary of tags
        :return:
        """
        filters = []
        if name:
            filters = [
                {
                    "Name": "tag-key",
                    "Values": ["Name"]
                },
                {
                    "Name": "tag-value",
                    "Values": [name]
                }
            ]
        if states:
            assert isinstance(states, list), "Instance state should be a list"
            filters.append(
                {
                    "Name": "instance-state-name",
                    "Values": states
                }
            )
        filters += tag_dict_to_aws_filter(tags)

        reservations = self._client.describe_instances(
            Filters=filters
        )["Reservations"]

        minions = []
        for r in reservations:
            instances = r["Instances"]
            for i in instances:
                minions.append(i)
        return minions

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_subnets(self, zones=None, tags=None):
        """
        :param zones: list of availability zones
        :param tags: dictionary of cluster tags
        :return: a list of subnets
        """
        filters = tag_dict_to_aws_filter(tags)
        if zones:
            assert isinstance(zones, list), "Subnet zones should be a list"
            filters.append(
                {
                    "Name": "availabilityZone",
                    "Values": zones
                }
            )

        filters += tag_dict_to_aws_filter(tags)
        subnets = self._client.describe_subnets(
            Filters=filters
        )
        return subnets["Subnets"]

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_vpc_igws(self, vpc_id):
        """
        :param cluster_name_id:
        :param vpc_id:
        :return: cluster VPC's internet gateway
        """
        igw = self._client.describe_internet_gateways(
            Filters=[
                {
                    "Name": "attachment.vpc-id",
                    "Values": [vpc_id]
                }
            ]
        )
        return igw["InternetGateways"]

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_security_groups(self, tags=None):
        """
        :param tags: dictionary of key-value pair of security group tags
        :return:
        """
        filters = tag_dict_to_aws_filter(tags)
        sgs = self._client.describe_security_groups(
            Filters=filters
        )
        return sgs["SecurityGroups"]

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def authorize_ingress(self, security_group_id, rule):
        """
        :param security_group_id:
        :param rule: EC2IPPermission objects describing IP permissions
        :return:
        """
        assert security_group_id, "Must provide security group ID"
        assert isinstance(rule, EC2IPPermission), "Rules must be a list of EC2IPPermission describing IP permissions"
        logger.info("Authorizing ingress of security group %s. Rule: %s", security_group_id, rule)
        try:
            self._client.authorize_security_group_ingress(
                GroupId=security_group_id,
                IpProtocol=rule.protocol,
                FromPort=rule.from_port,
                ToPort=rule.to_port,
                CidrIp=rule.cidr
            )
        except ClientError as ce:
            if "InvalidPermission.Duplicate" in str(ce):
                pass
            else:
                raise ce

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def revoke_ingress(self, security_group_id, rule):
        """
        :param security_group_id:
        :param rule: EC2IPPermission objects describing IP permissions
        :return:
        """
        assert security_group_id, "Must provide security group ID"
        assert isinstance(rule, EC2IPPermission), "Rules must be a list of EC2IPPermission describing IP permissions"
        logger.info("Revoking ingress of security group %s. Rule: %s", security_group_id, rule)
        try:
            self._client.revoke_security_group_ingress(
                GroupId=security_group_id,
                IpProtocol=rule.protocol,
                FromPort=rule.from_port,
                ToPort=rule.to_port,
                CidrIp=rule.cidr
            )
        except ClientError as ce:
            if "InvalidPermission.NotFound" in str(ce):
                pass
            else:
                raise ce
