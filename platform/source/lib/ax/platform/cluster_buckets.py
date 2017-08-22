#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import json
import logging

from ax.aws.profiles import AWSAccountInfo
from ax.cloud import Cloud
from ax.meta import AXClusterId, AXClusterConfigPath, AXClusterDataPath, AXSupportConfigPath, AXUpgradeConfigPath

from ax.platform.exceptions import AXPlatformException


logger = logging.getLogger(__name__)

DATA_CORS_CONFIG = {
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

upgrade_bucket_policy_template = """
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::{id}:root"
            },
            "Action": [
                "s3:ListBucket",
                "s3:GetBucketLocation",
                "s3:ListBucketMultipartUploads",
                "s3:ListBucketVersions"
            ],
            "Resource": "arn:aws:s3:::{s3}"
        },
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::{id}:root"
            },
            "Action": "s3:*",
            "Resource": "arn:aws:s3:::{s3}/*"
        }
    ]
}
"""

support_bucket_policy_template = """
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::{id}:root"
            },
            "Action": [
                "s3:ListBucket",
                "s3:GetBucketLocation",
                "s3:ListBucketMultipartUploads",
                "s3:ListBucketVersions",
                "s3:GetBucketAcl"
            ],
            "Resource": "arn:aws:s3:::{s3}"
        },
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::{id}:root"
            },
            "Action": [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:AbortMultipartUpload",
                "s3:ListMultipartUploadParts"
            ],
            "Resource": "arn:aws:s3:::{s3}/*"
        }
    ]
}
"""


class AXClusterBuckets(object):
    """
    Bucket created in target account, same as cluster account.
    """

    def __init__(self, name_id, aws_profile, aws_region):
        self._name_id = name_id
        self._aws_profile = aws_profile
        self._aws_region = aws_region

    def update(self):
        logger.info("Creating and updating all cluster buckets ...")
        self._update_cluster_bucket()
        self._update_data_bucket()
        logger.info("Creating and updating all cluster buckets ... DONE")

    def delete(self):
        logger.info("Deleting all cluster buckets ...")
        self._delete_cluster_bucket()
        self._delete_data_bucket()
        logger.info("Deleting all cluster buckets ... DONE")

    def _update_cluster_bucket(self):
        bucket_name = AXClusterConfigPath(name_id=self._name_id).bucket()
        cluster_bucket = Cloud().get_bucket(bucket_name, aws_profile=self._aws_profile, region=self._aws_region)

        if not cluster_bucket.create():
            raise AXPlatformException("Failed to create S3 bucket {}".format(cluster_bucket.get_bucket_name()))
        logger.info("Created %s bucket ... DONE", cluster_bucket.get_bucket_name())

    def _update_data_bucket(self):
        data_bucket = Cloud().get_bucket(AXClusterDataPath(name_id=self._name_id).bucket(),
                                         aws_profile=self._aws_profile, region=self._aws_region)

        if not data_bucket.create():
            raise AXPlatformException("Failed to create S3 bucket {}".format(data_bucket.get_bucket_name()))
        # Update CORS config for data bucket too.
        logger.info("Checking CORS config for %s.", data_bucket.get_bucket_name())
        data_bucket.put_cors(DATA_CORS_CONFIG)

        logger.info("Created %s bucket ... DONE", data_bucket.get_bucket_name())

    def _delete_cluster_bucket(self):
        logger.info("Deleting applatix-cluster bucket contents for cluster %s ...", self._name_id)
        cluster_bucket = Cloud().get_bucket(AXClusterConfigPath(name_id=self._name_id).bucket(),
                                            aws_profile=self._aws_profile, region=self._aws_region)

        idobj = AXClusterId(name=self._name_id)
        cluster_config_path = AXClusterConfigPath(name_id=self._name_id)
        cluster_name = idobj.get_cluster_name()
        prefix = cluster_name + "/"

        # TODO: Not idempotent here.
        # Consider the following case: if there is exception thrown when deleting S3 objects, install stage 1
        # information has already been deleted but not everything are successfully deleted, the next time user
        # executes "delete", this program will assume install stage 1 has been cleaned up.
        exempt = [idobj.get_cluster_id_s3_key(), cluster_config_path.cluster_install_stage0_key()]
        logger.info("Deleting objects for cluster %s from bucket %s. This may take some while.",
                    cluster_name,
                    cluster_bucket.get_bucket_name())
        cluster_bucket.delete_all(obj_prefix=prefix, exempt=exempt)
        logger.info("Deleting objects for cluster %s from bucket %s ... DONE",
                    cluster_name, cluster_bucket.get_bucket_name())
        logger.info("Deleting stage0 information ...")
        for item in exempt:
            cluster_bucket.delete_object(item)
        logger.info("Deleting stage0 information ... DONE")

    def _delete_data_bucket(self):
        logger.info("Deleting applatix-data bucket contents for cluster %s ...", self._name_id)
        data_bucket = Cloud().get_bucket(AXClusterDataPath(name_id=self._name_id).bucket(),
                                         aws_profile=self._aws_profile, region=self._aws_region)
        cluster_name = AXClusterId(name=self._name_id).get_cluster_name()
        prefix = cluster_name + "/"
        logger.info("Deleting objects for cluster %s from bucket %s. This may take some while.",
                    cluster_name,
                    data_bucket.get_bucket_name())
        data_bucket.delete_all(obj_prefix=prefix)
        logger.info("Deleting objects for cluster %s from bucket %s ... DONE",
                    cluster_name, data_bucket.get_bucket_name())


class AXPortalBuckets(object):
    """
    Bucket created in AX portal account, for support purpose.
    """
    def __init__(self, name_id, aws_profile, aws_region):
        self._name_id = name_id
        self._aws_profile = aws_profile
        self._aws_region = aws_region

    def update(self, iam):
        """
        Create all buckets in portal account.
        """
        logger.info("Creating applatix-support and applatix-upgrade buckets ...")
        support_bucket = Cloud().get_bucket(AXSupportConfigPath(name_id=self._name_id).bucket(),
                                            aws_profile=self._aws_profile, region=self._aws_region)
        upgrade_bucket = Cloud().get_bucket(AXUpgradeConfigPath(name_id=self._name_id).bucket(),
                                            aws_profile=self._aws_profile, region=self._aws_region)

        # Retry create while bucket is created is fine
        if not support_bucket.create():
            raise AXPlatformException("Failed to create S3 bucket {}".format(support_bucket.get_bucket_name()))

        # If policy is already there, we don't update
        if not support_bucket.get_policy():
            logger.info("Argo support bucket policy does not exist, creating new one...")
            if not support_bucket.put_policy(
                    policy=self._generate_bucket_policy_string(template=support_bucket_policy_template,
                                                               bucket_name=support_bucket.get_bucket_name(),
                                                               iam=iam)
            ):
                raise AXPlatformException(
                    "Failed to configure policy for S3 bucket {}".format(support_bucket.get_bucket_name()))

        if not upgrade_bucket.create():
            raise AXPlatformException("Failed to create S3 bucket {}".format(support_bucket.get_bucket_name()))

        if not upgrade_bucket.get_policy():
            logger.info("Argo upgrade bucket policy does not exist, creating new one...")
            if not upgrade_bucket.put_policy(
                    policy=self._generate_bucket_policy_string(template=support_bucket_policy_template,
                                                               bucket_name=upgrade_bucket.get_bucket_name(),
                                                               iam=iam)
            ):
                raise AXPlatformException(
                    "Failed to configure policy for S3 bucket {}".format(support_bucket.get_bucket_name()))

        # Tag them right away to avoid race deletion.
        upgrade_bucket.put_object(key=AXUpgradeConfigPath(name_id=self._name_id).tag(),
                                  data="tag",
                                  ACL="bucket-owner-full-control")
        support_bucket.put_object(key=AXSupportConfigPath(name_id=self._name_id).tag(),
                                  data="tag",
                                  ACL="bucket-owner-full-control")
        logger.info("Created %s and %s buckets ... DONE", support_bucket.get_bucket_name(),
                    upgrade_bucket.get_bucket_name())

    def _generate_bucket_policy_string(self, template, bucket_name, iam):
        """
        Create actual policy based on input from cluster info.
        """
        aws_cid = AWSAccountInfo(aws_profile=self._aws_profile).get_account_id_from_iam(iam)
        policy = json.loads(template)
        for s in policy["Statement"]:
            s["Principal"]["AWS"] = s["Principal"]["AWS"].format(id=aws_cid)
            s["Resource"] = s["Resource"].format(s3=bucket_name)
        return json.dumps(policy)
