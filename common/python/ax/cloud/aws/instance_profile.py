#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Wrapper for AWS IAM

import json
import logging

import boto3
import botocore
from retrying import retry

from .util import default_aws_retry

logger = logging.getLogger(__name__)

# All instance profiles require this permission in assume_role_statement.
assume_role_statement = {
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {"Service": "ec2.amazonaws.com"},
            "Action": "sts:AssumeRole"
        }
    ]
}


class InstanceProfile(object):
    def __init__(self, name, aws_profile=None):
        self._name = name
        self._iam = boto3.Session(profile_name=aws_profile).client("iam")

    def update(self, policy):
        self.create_role(self._name, assume_role_statement)
        self.put_policy(self._name, self._name, policy)
        self.create_instance_profile(self._name)
        self.remove_role_from_instance_profile(self._name)
        self.add_role_to_instance_profile(self._name)

    def delete(self):
        self.remove_role_from_instance_profile(self._name)
        self.delete_instance_profile(self._name)
        self.delete_policy(self._name, self._name)
        self.delete_role(self._name)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def create_role(self, name, assume_role):
        try:
            self._iam.create_role(RoleName=name, AssumeRolePolicyDocument=json.dumps(assume_role))
            logger.info("Created role %s.", name)
        except Exception as e:
            # TODO: how to handle factory exceptions?
            # Exception is botocore.errorfactory.EntityAlreadyExistsException
            if "EntityAlreadyExists" in str(e):
                logger.info("Role %s already exists.", name)
            else:
                raise

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def delete_role(self, name):
        try:
            self._iam.delete_role(RoleName=name)
            logger.info("Deleted role %s", name)
        except Exception as e:
            if "NoSuchEntity" in str(e):
                logger.info("Role %s already deleted.", name)
            else:
                raise

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_instance_profile(self, name):
        try:
            profile = self._iam.get_instance_profile(InstanceProfileName=name)
            logger.info("Instance profile %s is %s", name, profile)
            return profile
        except Exception:
            logger.info("Instance profile %s not found.", name)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def put_policy(self, role_name, policy_name, policy):
        # Put always overwrite, which is what we need.
        self._iam.put_role_policy(RoleName=role_name, PolicyName=policy_name, PolicyDocument=json.dumps(policy))
        logger.info("Added policy %s to role %s", policy_name, role_name)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def delete_policy(self, role_name, policy_name):
        try:
            self._iam.delete_role_policy(RoleName=role_name, PolicyName=policy_name)
            logger.info("Deleted policy %s from role %s", policy_name, role_name)
        except Exception as e:
            if "NoSuchEntity" in str(e):
                logger.info("Policy %s already deleted from role %s.", policy_name, role_name)
            else:
                raise

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def create_instance_profile(self, name):
        try:
            self._iam.create_instance_profile(InstanceProfileName=name)
            logger.info("Created instance profile %s", name)
        except Exception as e:
            if "EntityAlreadyExists" in str(e):
                logger.info("Instance profile %s already exists.", name)
            else:
                raise

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def delete_instance_profile(self, name):
        try:
            self._iam.delete_instance_profile(InstanceProfileName=name)
            logger.info("Deleted instance profile %s", name)
        except Exception as e:
            if "NoSuchEntity" in str(e):
                logger.info("Instance profile %s already deleted.", name)
            else:
                raise

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def add_role_to_instance_profile(self, name):
        self._iam.add_role_to_instance_profile(RoleName=name, InstanceProfileName=name)
        logger.info("Added role %s to instance profile %s", name, name)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def remove_role_from_instance_profile(self, name):
        try:
            self._iam.remove_role_from_instance_profile(RoleName=name, InstanceProfileName=name)
            logger.info("Removed role %s from instance profile %s", name, name)
        except Exception as e:
            if "NoSuchEntity" in str(e):
                logger.info("Instance profile %s doesn't have a role", name)
            else:
                raise
