#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Cloud environment

import os
import logging
import requests
from future.utils import with_metaclass

from ax.util.singleton import Singleton

logger = logging.getLogger(__name__)


class Cloud(with_metaclass(Singleton, object)):
    """
    There are two sets of cloud methods:
      - Target cloud (The cloud currently running software is operating on)
      - Own cloud (The cloud currently running software is running on)

    For example, when we want to install a GCP cluster from an AWS machine,
    Own cloud points to AWS and Target cloud points to GCP

    If code always runs inside a POD as part of cluster and you need to decide which cloud this is,
    use in_cloud_*() methods. These methods will always try to guess which cloud provider this code run in.

    If you need to operate a remote target cluster, you should use target_cloud_* methods.
    Any program requiring this logic should call set_target_cloud() at initialization time.
    Usually this is provided by command arguments.
    """
    AX_CLOUD_UNKNOWN = "AX_CLOUD_UNKNOWN"
    AX_CLOUD_AWS = "AX_CLOUD_AWS"
    AX_CLOUD_GCP = "AX_CLOUD_GCP"

    CLOUD_AWS = "aws"
    CLOUD_GCP = "gcp"
    CLOUD_UNKNOWN = "unknown"

    VALID_TARGET_CLOUD_INPUT = [CLOUD_AWS, CLOUD_GCP]

    # Need these translations as env/cli input literals, i.e.
    # "aws" or "gcp" can be convenient for end users, but such
    # key words can be reserved by cloud provider in some cases.
    # So we always want to use Argo specific key words internally
    CLOUD_INPUT_TO_INTERNAL = {
        CLOUD_AWS: AX_CLOUD_AWS,
        CLOUD_GCP: AX_CLOUD_GCP
    }

    CLOUD_INTERNAL_TO_INPUT = {
        AX_CLOUD_AWS: CLOUD_AWS,
        AX_CLOUD_GCP: CLOUD_GCP,
        AX_CLOUD_UNKNOWN: CLOUD_UNKNOWN
    }

    def __init__(self, target_cloud=None):
        self._own_cloud = None
        if target_cloud:
            self._target_cloud = self.CLOUD_INPUT_TO_INTERNAL[target_cloud]
        else:
            self._try_initialize_target_cloud()

    def _try_initialize_target_cloud(self):
        target_cloud = os.getenv("AX_TARGET_CLOUD", None)

        # TODO: might want to enforce env "AX_TARGET_CLOUD"
        if not target_cloud:
            logger.warning("Target cloud not explicitly set, trying to set it to own cloud type")
            try:
                self._target_cloud = self._get_own_cloud_type()
            except Exception as e:
                logger.warning("Cannot determine own cloud: %s, program might be running locally", e)
        else:
            self.set_target_cloud(target_cloud)

    def own_cloud(self):
        if not self._own_cloud:
            self._own_cloud = self._get_own_cloud_type()
        return self.CLOUD_INTERNAL_TO_INPUT.get(self._own_cloud, None)

    def target_cloud(self):
        return self.CLOUD_INTERNAL_TO_INPUT.get(self._target_cloud, None)

    def set_target_cloud(self, target_cloud):
        assert target_cloud in self.VALID_TARGET_CLOUD_INPUT, "Invalid target cloud {}. Please choose from {}".format(
            target_cloud, self.VALID_TARGET_CLOUD_INPUT)
        self._target_cloud = self.CLOUD_INPUT_TO_INTERNAL.get(target_cloud)

    def in_cloud_aws(self):
        if not self._own_cloud:
            self._own_cloud = self._get_own_cloud_type()
        return self._own_cloud == self.AX_CLOUD_AWS

    def in_cloud_gcp(self):
        if not self._own_cloud:
            self._own_cloud = self._get_own_cloud_type()
        return self._own_cloud == self.AX_CLOUD_GCP

    def target_cloud_aws(self):
        return self._target_cloud == self.AX_CLOUD_AWS

    def target_cloud_gcp(self):
        return self._target_cloud == self.AX_CLOUD_GCP

    def meta_data(self):
        """
        Return a meta data object that can be used to get instance metadata.
        As we can ONLY access metadata within the cloud, metadata is always
        determined based on own cloud type
        """
        if not self._own_cloud:
            self._own_cloud = self._get_own_cloud_type()
        if self.in_cloud_gcp():
            from .gke.meta_data import GCEMetaData
            return GCEMetaData()
        elif self.in_cloud_aws():
            from ax.aws.meta_data import AWSMetaData
            return AWSMetaData()
        else:
            assert 0, "Cloud {} not supported".format(self._own_cloud)

    def _get_own_cloud_type(self):
        # This is hack but works well.
        # Using one requests get call speeds up detection for both AWS and GCP.
        # We could also use other host signature but we need to support in pod call too.
        try:
            r = requests.get("http://169.254.169.254/", timeout=5)
            if r.status_code == requests.codes.ok:
                if "2012-01-12" in r.text and "2016-04-19" in r.text:
                    return self.AX_CLOUD_AWS
                elif "computeMetadata" in r.text:
                    return self.AX_CLOUD_GCP
        except:
            assert self._target_cloud is not None, "Need to set target cloud or detect own cloud"

        return self.AX_CLOUD_UNKNOWN

    def get_bucket(self, bucket_name, **kwargs):
        """
        Return a bucket object that contains methods to operate on a bucket
        of a cloud that current running code operates on (target cloud).

        :param bucket_name:
        :param kwargs:
        :return:
        """
        if self.target_cloud_gcp():
            from ax.cloud.gke.gcs import AXGCSBucket
            return AXGCSBucket(bucket_name)
        elif self.target_cloud_aws():
            from ax.cloud.aws import AXS3Bucket
            aws_profile = kwargs.get('aws_profile', None)
            region = kwargs.get("region", None)
            return AXS3Bucket(bucket_name, aws_profile, region=region)
        else:
            assert 0, "Unsupported cloud provider {}".format(self._target_cloud)
