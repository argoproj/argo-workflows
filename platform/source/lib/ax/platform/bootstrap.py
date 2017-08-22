#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging
import time

from ax.cloud.aws import EC2, EC2IPPermission
from ax.platform.ax_asg import AXUserASGManager
from ax.platform.consts import COMMON_CLOUD_RESOURCE_TAG_KEY
from ax.platform.exceptions import AXPlatformException
from botocore.exceptions import ClientError


logger = logging.getLogger(__name__)


"""
Permissions
The worker running the cluster autoscaler will need access to certain resources and actions:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "autoscaling:DescribeAutoScalingGroups",
                "autoscaling:DescribeAutoScalingInstances",
                "autoscaling:SetDesiredCapacity",
                "autoscaling:TerminateInstanceInAutoScalingGroup"
            ],
            "Resource": "*"
        }
    ]
}

Unfortunately AWS does not support ARNs for autoscaling groups yet so you must use "*" as the resource.
More information [here](http://docs.aws.amazon.com/autoscaling/latest/userguide/IAM.html#UsingWithAutoScaling_Actions).
"""

# TODO (Harry): Make AXBootstrap a generic cloud management package


class AXBootstrap(object):
    def __init__(self, cluster_name_id, aws_profile=None, region=None):
        self._cluster_name_id = cluster_name_id
        self._aws_profile = aws_profile
        self._region = region

    def modify_asg(self, min, max):
        logger.info("Modifying autoscaling group ...")

        asg_manager = AXUserASGManager(self._cluster_name_id, self._region, self._aws_profile)

        asg = asg_manager.get_variable_asg()
        if not asg:
            raise AXPlatformException("Failed to get variable autoscaling group for cluster {}".format(self._cluster_name_id))
        asg_name = asg["AutoScalingGroupName"]
        try:
            asg_manager.set_asg_spec(name=asg_name, minsize=1, maxsize=max)
        except ClientError as ce:
            raise AXPlatformException("Failed to set cluster's variable autoscaling group min/max. Error: {}".format(ce))

        logger.info("Modifying cluster autoscaling group ... DONE")

    def modify_node_security_groups(self, old_cidr, new_cidr, action_name):
        """
        Modify kube-up default security groups:
        For master node:
            - tcp, 22, 22, <new_cidr>
            - tcp, 443, 443, <new_cidr>
        For minion nodes:
            - tcp, 22, 22, <new_cidr>
        :param cluster_name_id:
        :param old_cidr:
        :param new_cidr:
        :param action_name: for debugging purposes. e.g. "allow-creator"
        :return:
        """
        logger.info("Modifying security groups for \"%s\" ...", action_name)
        ec2 = EC2(profile=self._aws_profile, region=self._region)
        sgs = ec2.get_security_groups(
            tags={
                COMMON_CLOUD_RESOURCE_TAG_KEY: [self._cluster_name_id]
            }
        )

        if not sgs:
            raise AXPlatformException("Failed to find security groups of cluster %s", self._cluster_name_id)

        cidrs_to_delete = set(old_cidr) - set(new_cidr)
        cidrs_to_add = set(new_cidr) - set(old_cidr)
        logger.info("Revoking accesses from %s; authorizing accesses to %s", cidrs_to_delete, cidrs_to_add)

        for sg in sgs:
            if "k8s-elb" in sg["GroupName"]:
                continue
            sg_id = sg["GroupId"]

            for cidr in cidrs_to_delete:
                ec2.revoke_ingress(
                    security_group_id=sg_id,
                    rule=EC2IPPermission(EC2IPPermission.TCP, from_port=22, to_port=22, cidr=cidr)
                )
            for cidr in cidrs_to_add:
                ec2.authorize_ingress(
                    security_group_id=sg_id,
                    rule=EC2IPPermission(EC2IPPermission.TCP, from_port=22, to_port=22, cidr=cidr)
                )

            if "master" in sg["GroupName"]:
                for cidr in cidrs_to_delete:
                    ec2.revoke_ingress(
                        security_group_id=sg_id,
                        rule=EC2IPPermission(EC2IPPermission.TCP, from_port=443, to_port=443, cidr=cidr)
                    )
                for cidr in cidrs_to_add:
                    ec2.authorize_ingress(
                        security_group_id=sg_id,
                        rule=EC2IPPermission(EC2IPPermission.TCP, from_port=443, to_port=443, cidr=cidr)
                    )

        logger.info("Modifying security groups for \"%s\" ... DONE", action_name)

    def stop(self):
        pass
