#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module to handle AWS launch configurations.
"""

import logging
import base64

import botocore
import boto3
from retrying import retry

from .util import default_aws_retry

logger = logging.getLogger(__name__)


class LaunchConfig(object):
    def __init__(self, name, aws_profile=None, aws_region=None):
        """
        Init method. Will create AWS clients.
        """
        self._name = name
        self._aws_profile = aws_profile
        self._client = boto3.Session(profile_name=aws_profile, region_name=aws_region).client("autoscaling")

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def create(self, config):
        """
        Create a new launch config using config dict passed in.
        Config name in __init__ and create must match.
        """
        config_name = config.get("LaunchConfigurationName", self._name)
        assert config_name == self._name, "Config name mismatch {} {}".format(config_name, self._name)
        config["LaunchConfigurationName"] = self._name
        self._client.create_launch_configuration(**config)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def delete(self):
        """
        Delete current config.
        Don't throw exception if launch config doesn't exist.
        """
        try:
            self._client.delete_launch_configuration(LaunchConfigurationName=self._name)
        except botocore.exceptions.ClientError as e:
            if "not found" in e.response["Error"]["Message"]:
                logger.warn("Launch configuration %s not found", self._name)
            else:
                raise

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def get(self):
        """
        Get launch config data based on current name.
        Handle base64 decode automatically. Returned user data is always decoded.
        Return None if config doesn't exist.
        """
        lc = self._client.describe_launch_configurations(LaunchConfigurationNames=[self._name])
        if len(lc["LaunchConfigurations"]) == 0:
            return None
        else:
            config = lc["LaunchConfigurations"][0]
            config["UserData"] = base64.b64decode(config["UserData"])
            return config

    def copy(self, new_name, new_config, retain_spot_price=False, delete_old=False):
        """
        Make a copy of current launch config and return new one.
        Optionally delete old one.
        This is composite API that is built with above basic APIs.
        No need to retry as all underlying APIs already have retry.
        """
        config = self.get()
        config.update(new_config)
        config["LaunchConfigurationName"] = new_name

        # The following fields are not allowed as launch config input.
        config.pop("LaunchConfigurationARN")
        config.pop("CreatedTime")
        config.pop("KernelId")
        config.pop("RamdiskId")
        if not retain_spot_price and config.get("SpotPrice", None) != None:
            logger.info("Not retaining spot price!")
            config.pop("SpotPrice")

        new_lc = LaunchConfig(new_name, aws_profile=self._aws_profile)
        new_lc.create(config)
        if delete_old:
            self.delete()
        return new_lc
