#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Module to manage instance profiles for AX cluster.

import copy
import logging

from ax.aws.profiles import AWSAccountInfo
from ax.cloud.aws import get_aws_partition_from_region, AWS_ALL_RESOURCES
from ax.cloud.aws.instance_profile import InstanceProfile

logger = logging.getLogger(__name__)


# Default instance profile statement for master.
MASTER_POLICY_TEMPLATE = {
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "ec2:*",
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "elasticloadbalancing:*",
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "route53:*",
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": "arn:{partition}:s3:::applatix-*"
        },
        {
            "Action": [
                "autoscaling:DescribeAutoScalingGroups",
                "autoscaling:DescribeAutoScalingInstances",
                "autoscaling:SetDesiredCapacity",
                "autoscaling:TerminateInstanceInAutoScalingGroup",
                "autoscaling:UpdateAutoScalingGroup",
                "autoscaling:DescribeLaunchConfigurations",
                "autoscaling:CreateLaunchConfiguration",
                "autoscaling:DeleteLaunchConfiguration"
            ],
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": "iam:PassRole",
            "Resource": [
                "arn:{partition}:iam::{account}:role/{master_name}",
                "arn:{partition}:iam::{account}:role/{minion_name}"
            ],
            "Effect": "Allow"
        }
    ]
}

# Default instance profile statement for minions.
MINION_POLICY_TEMPLATE = {
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "s3:*",
            "Resource": [
                "arn:{partition}:s3:::applatix-*",
                "arn:{partition}:s3:::*axawss3-test*",
                "arn:{partition}:s3:::ax-public",
                "arn:{partition}:s3:::ax-public/*"
            ],
            "Effect": "Allow"
        },
        {
            "Action": [
                "ec2:Describe*",
                "ec2:CreateVolume",
                "ec2:DeleteVolume",
                "ec2:AttachVolume",
                "ec2:DetachVolume",
                "ec2:ReplaceRoute",
                "ec2:CreateSnapshot",
                "ec2:DeleteSnapshot",
                "ec2:AuthorizeSecurityGroupIngress",
                "ec2:AuthorizeSecurityGroupEgress",
                "ec2:RevokeSecurityGroupIngress",
                "ec2:RevokeSecurityGroupEgress",
                "ec2:RunInstances",
                "ec2:TerminateInstances",
                "ec2:AssociateAddress",
                "ec2:CreateTags",
                "ec2:CreateSecurityGroup",
                "ec2:DeleteSecurityGroup",
                "ec2:DescribeSecurityGroups"
            ],
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": "route53:*",
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": [
                "ecr:GetAuthorizationToken",
                "ecr:BatchCheckLayerAvailability",
                "ecr:GetDownloadUrlForLayer",
                "ecr:GetRepositoryPolicy",
                "ecr:DescribeRepositories",
                "ecr:ListImages",
                "ecr:BatchGetImage"
            ],
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": [
                "autoscaling:DescribeAutoScalingGroups",
                "autoscaling:DescribeAutoScalingInstances",
                "autoscaling:SetDesiredCapacity",
                "autoscaling:TerminateInstanceInAutoScalingGroup",
                "autoscaling:UpdateAutoScalingGroup",
                "autoscaling:DescribeLaunchConfigurations",
                "autoscaling:CreateLaunchConfiguration",
                "autoscaling:DeleteLaunchConfiguration",
                "autoscaling:AttachLoadBalancers",
                "autoscaling:DetachLoadBalancers"
            ],
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": "elasticloadbalancing:*",
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": "sts:AssumeRole",
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": [
                "iam:GetServerCertificate",
                "iam:DeleteServerCertificate",
                "iam:UploadServerCertificate"
            ],
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": "iam:PassRole",
            "Resource": [
                "arn:{partition}:iam::{account}:role/{master_name}",
                "arn:{partition}:iam::{account}:role/{minion_name}"
            ],
            "Effect": "Allow"
        }
    ]
}


class AXClusterInstanceProfile(object):
    def __init__(self, name_id, region_name, aws_profile=None):
        self._name_id = name_id
        self._master_name = name_id + "-master"
        self._minion_name = name_id + "-minion"
        self._aws_partition = get_aws_partition_from_region(region_name)
        self._master_profile = InstanceProfile(self._master_name, aws_profile=aws_profile)
        self._minion_profile = InstanceProfile(self._minion_name, aws_profile=aws_profile)

        # Create pass_role statement specific to this cluster.
        self._account = AWSAccountInfo(aws_profile=aws_profile).get_account_id()

        self._master_policy = self._format_policy_resources(MASTER_POLICY_TEMPLATE)
        self._minion_policy = self._format_policy_resources(MINION_POLICY_TEMPLATE)

    def update(self):
        """
        Update or create all instance profiles for a cluster.
        """
        self._master_profile.update(self._master_policy)
        self._minion_profile.update(self._minion_policy)

    def delete(self):
        """
        Delete all instance profiles for a cluster.
        """
        self._master_profile.delete()
        self._minion_profile.delete()

    def get_master_arn(self):
        """
        Get ARN for master instance profile.
        Currently needed since our upgrade code need ARN rather than name.
        """
        profile = self._master_profile.get_instance_profile(self._master_name)
        if profile:
            return profile["InstanceProfile"]["Arn"]

    def get_minion_instance_profile_name(self):
        """
        Get name for minion instance profile.
        No need to lookup from AWS as name is sufficient.
        """
        return self._minion_name

    def _format_policy_resources(self, policy):
        """
        Dynamically insert critical information into policy: aws partition, aws account id,
        master name, minion name
        :param policy:
        :return:
        """
        p = copy.deepcopy(policy)
        for statement in p.get("Statement", []):
            resource = statement.get("Resource", AWS_ALL_RESOURCES)
            if resource != AWS_ALL_RESOURCES:
                new_resource = []
                if isinstance(resource, list):
                    for r in resource:
                        new_resource.append(r.format(partition=self._aws_partition, account=self._account,
                                                     master_name=self._master_name, minion_name=self._minion_name))
                else:
                    new_resource.append(resource.format(partition=self._aws_partition, account=self._account,
                                                        master_name=self._master_name, minion_name=self._minion_name))
                statement["Resource"] = new_resource
        return p
