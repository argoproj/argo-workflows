#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Get all AWS account info associated with AWS profile.
"""

import boto3


class AWSAccountInfo(object):
    def __init__(self, aws_profile=None):
        self._session = boto3.Session(profile_name=aws_profile)

    def get_account_id(self):
        """
        Get AWS account ID.
        Read from caller identify based on profile
        """
        sts = self._session.client("sts")
        return sts.get_caller_identity()["Account"]

    def get_account_id_from_iam(self, iam):
        return iam.split(":")[-2]
