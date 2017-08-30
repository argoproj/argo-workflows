#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os
import random

os.environ["AX_TARGET_CLOUD"] = "aws"

AWS_REGIONS = [
    "us-east-1", "us-east-2", "us-west-1", "us-west-2", "ca-central-1", "eu-west-1", "eu-west-2", "eu-central-1",
    "ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2", "ap-south-1", "sa-east-1"
]

TEST_AWS_PROFILE = None
TEST_BUCKET_NAME_TEMPLATE = "axawss3-test-{region}"
TEST_AWS_REGION = random.choice(AWS_REGIONS)
TEST_BUCKET_NAME = "axawss3-test-{}".format(TEST_AWS_REGION)

TEST_INVALID_POLICY_FORMAT = """
{
xxx
}
"""


TEST_INVALID_POLICY_CONTENT = """
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::123456789012:root"
            },
            "Action": [
                "s3:ListBucket",
                "s3:GetBucketLocation",
                "s3:ListBucketMultipartUploads",
                "s3:ListBucketVersions",
                "s3:GetBucketAcl"
                ],
            "Resource": "arn:aws:s3:::{s3}"
        }
    ]
}
"""

TEST_CORS_CONFIG = {
    "version": 1,
    "config": {
        "CORSRules": [
            {
                "AllowedMethods": ["GET"],
                "AllowedOrigins": ["*"],
                "AllowedHeaders": ["*"]
            }
        ]
    }
}

