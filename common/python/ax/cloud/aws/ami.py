#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Wrapper for AWS AMI


import boto3
import logging
from retrying import retry

from .util import default_aws_retry

logger = logging.getLogger(__name__)


class AMI(object):
    def __init__(self, aws_region, aws_profile=None):
        self._aws_profile = aws_profile
        self._region = aws_region
        self._amicli = boto3.Session(region_name=aws_region, profile_name=aws_profile).client("ec2")

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_ami_id_from_name(self, ami_name):
        ret = self._amicli.describe_images(
            Filters=[
                {
                    "Name": "name",
                    "Values": [
                        ami_name
                    ]
                }
            ]
        )

        ami_id = ret["Images"][0]["ImageId"]
        logger.info("AMI id for %s in region %s is %s", ami_name, self._region, ami_id)
        return ami_id
