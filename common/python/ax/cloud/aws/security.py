#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
AWS sts related services
"""

import boto3
import logging
import os
import time

from botocore.exceptions import ProfileNotFound
from retrying import retry
from .util import default_aws_retry

logger = logging.getLogger(__name__)

DEFAULT_SESSION_TEMPLATE = "default-aws-session-{ts}"
DEFAULT_SESSION_DURATION = 3600

AWS_PROFILE_LOCATION = os.path.join(os.getenv("HOME", "/root"), ".aws")
AWS_PROFILE_PATH = os.path.join(AWS_PROFILE_LOCATION, "credentials")
AWS_PROFILE_CREDENTIAL_CONTENT = """
[{name}]
aws_access_key_id = {ak}
aws_secret_access_key = {sak}
"""


class SecurityToken(object):
    def __init__(self, aws_profile=None):
        self.sts = boto3.Session(profile_name=aws_profile).client("sts")

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_caller_identity(self):
        return self.sts.get_caller_identity()

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_credential(self, role_arn, external_id=None, session_name=None, duration=DEFAULT_SESSION_DURATION):
        assert role_arn, "Must specify a role arn to do assume_role"
        if not session_name:
            session_name = DEFAULT_SESSION_TEMPLATE.format(ts=int(time.time()*1000))
        ret = self.sts.assume_role(RoleArn=role_arn,
                                   ExternalId=external_id,
                                   RoleSessionName=session_name,
                                   DurationSeconds=duration)

        access_key_id = ret["Credentials"]["AccessKeyId"]
        secret_access_key = ret["Credentials"]["SecretAccessKey"]
        session_token = ret["Credentials"]["SessionToken"]

        logger.info("Successfully assumed role from aws")
        return access_key_id, secret_access_key, session_token

    @staticmethod
    def generate_profile_from_credentials(access_key, secret_access_key, token=None, profile_name="default"):
        """
        Because aws's profile must be written to a file in order to use it in boto3.Session(),
        we dump credentials into file
        """

        try:
            boto3.Session(profile_name=profile_name)
            raise Exception("Invalid Profile: profile \"{}\" already exists".format(profile_name))
        except ProfileNotFound:
            pass

        content = AWS_PROFILE_CREDENTIAL_CONTENT.format(name=profile_name,
                                                        ak=access_key,
                                                        sak=secret_access_key)
        if token:
            content += "\naws_session_token = {st}\n".format(st=token)

        if not os.path.isdir(AWS_PROFILE_LOCATION):
            os.makedirs(AWS_PROFILE_LOCATION)

        with open(AWS_PROFILE_PATH, "a") as f:
            f.write(content)
