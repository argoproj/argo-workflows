#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module to get cluster ID.
Care must be taken to run this from both inside and outside cluster.
Expect many other modules to import this. There should be minimum import to avoid circular dependency.

This class can deal with the following use cases:
1. caller pass in cluster name or cluster_name_id to create cluster name/id record in s3
2. caller wants to get cluster name/id information
    - caller input nothing and we look for AX_CLUSTER_NAME_ID env (which should happen in most our micro-services)
    - caller input name and we look up id from s3 (we should gradually get away from this method if we are
        able to provide all "who am i" information to the container)
This class assumes s3 bucket is created
"""

import logging
import os
import uuid

from ax.cloud import Cloud
from ax.util.singleton import Singleton
from future.utils import with_metaclass

from . import AXCustomerId, AXClusterNameIdParser

logger = logging.getLogger(__name__)
CLUSTER_NAME_ID_ENV_NAME = "AX_CLUSTER_NAME_ID"


class AXClusterId(with_metaclass(Singleton, object)):

    def __init__(self, name=None, aws_profile=None):
        self._input_name = name
        self._aws_profile = aws_profile

        # Cluster id related bucket and path info should be self-contained rather than
        # using config_s3_path object. Because config_s3_path needs both cluster name
        # and id to initialize. In case we haven't get cluster id yet, singletons in
        # config_s3_path cannot be properly initialized.
        self._bucket_template = "applatix-cluster-{account}-{seq}"
        self._cluster_id_bucket_path_template = "{name}/id"

        # Set bucket
        self._customer_id = AXCustomerId().get_customer_id()
        self._bucket_name = self._bucket_template.format(account=self._customer_id, seq=0)
        self._bucket = None

        # These values will be set when user calls get/create cluster name id
        self._cluster_name = None
        self._cluster_id = None
        self._cluster_name_id = None

    def create_cluster_name_id(self):
        """
        User input cluster name in format of "<name>" or "<name>-<id>", and this function creates
        a record in S3. If he name caller passed in does not include an ID, we generate one.

        If we already have a cluster name/id record in s3, this function should not be called to avoid
        existing clusters's records to get overridden
        :return: <cluster-name>-<cluster-id>
        """
        assert not self._cluster_name_id, "Cluster {} has it's name id already created".format(self._cluster_name_id)
        assert self._input_name, "Must provide input name to create cluster name id"
        name, cid = self._format_name_id(self._input_name)
        if cid is None:
            logger.info("Cluster id not provided, generate one.")
            if Cloud().target_cloud_gcp():
                cid = str(uuid.uuid4())[:8]
            elif Cloud().target_cloud_aws():
                cid = str(uuid.uuid1())
            else:
                assert False, "Must provide valid target cloud to create cluster name id. Currently target cloud is set to {}".format(Cloud().target_cloud())
        logger.info("Created new name-id %s", name + "-" + cid)

        # fill in cluster name id info
        self._cluster_name = name
        self._cluster_id = cid
        self._cluster_name_id = self._cluster_name + "-" + self._cluster_id
        return self._cluster_name_id

    def upload_cluster_name_id(self):
        """
        This function assumes cluster_name_id has been created already
        """
        logger.info("Uploading cluster name-id record to S3 ...")
        self._load_cluster_name_id_if_needed()
        self._instantiate_bucket_if_needed()
        id_key = self._cluster_id_bucket_path_template.format(name=self._cluster_name)
        self._bucket.put_object(id_key, self._cluster_id)
        logger.info("Uploaded cluster name (%s) and cluster id (%s) to S3", self._cluster_name, self._cluster_id)

    def get_cluster_name_id(self):
        """
        This function assumes cluster name/id record is created. It first looks for
        AX_CLUSTER_NAME_ID env, if not set, it looks up cluster id from s3.
        :return" cluster_name_id
        """
        self._load_cluster_name_id_if_needed()
        return self._cluster_name_id

    def get_cluster_name(self):
        self._load_cluster_name_id_if_needed()
        return self._cluster_name

    def get_cluster_id(self):
        self._load_cluster_name_id_if_needed()
        return self._cluster_id

    def get_cluster_id_s3_key(self):
        self._load_cluster_name_id_if_needed()
        return self._cluster_id_bucket_path_template.format(name=self._cluster_name)

    def _load_cluster_name_id_if_needed(self):
        if not self._cluster_name_id:
            self._load_cluster_name_id()

    def _instantiate_bucket_if_needed(self):
        if not self._bucket:
            logger.info("Instantiating cluster bucket ...")
            self._bucket = Cloud().get_bucket(self._bucket_name, aws_profile=self._aws_profile)
            assert self._bucket.exists(), "Bucket {} not created yet".format(self._bucket.get_bucket_name())

    def _load_cluster_name_id(self):
        """
        This function assumes cluster name/id record is created. It first looks for
        AX_CLUSTER_NAME_ID env, if not set, it looks up cluster id from s3.

        This function sets cluster_name_id, cluster_name, and cluster_id
        """
        # Try to get from env first
        name_id = os.getenv(CLUSTER_NAME_ID_ENV_NAME, None)
        if name_id:
            logger.info("Found cluster name id in env: %s", name_id)
            self._cluster_name_id = name_id
            self._cluster_name, self._cluster_id = self._format_name_id(self._cluster_name_id)

            # NOTE: if we find some cluster name id we cannot even parse from env, we still fail
            # directly even though it is possible that we might find something valid from s3 bucket,
            # as the program that brings up program (i.e. axinstaller) is already having trouble in
            # such case, which is already alerting
            assert self._cluster_name and self._cluster_id, "Failed to load cluster name and cluster id from env"
        else:
            self._lookup_id_from_bucket()
            assert self._cluster_name and self._cluster_id, "Failed to load cluster name and cluster id from bucket"
            self._cluster_name_id = "{}-{}".format(self._cluster_name, self._cluster_id)

    def _lookup_id_from_bucket(self):
        name, requested_cid = self._format_name_id(self._input_name)

        # Look up assumes bucket already exists, so there is no need to pass region
        # If bucket does not exist, AXS3Bucket will throw exception
        self._instantiate_bucket_if_needed()
        id_s3_key = self._cluster_id_bucket_path_template.format(name=name)
        cid = str(self._bucket.get_object(id_s3_key)).strip()
        if cid != "None":
            logger.info("Found existing cluster name %s-%s", name, cid)
            if cid != requested_cid:
                logger.info("Ignore requested cluster ID (%s). Real cluster id: %s", requested_cid, cid)
            self._cluster_name = name
            self._cluster_id = cid
        else:
            logger.info("Cannot find cluster name/id mapping from bucket")
            if requested_cid:
                logger.info("Using user defined cluster name: %s, cluster id: %s", name, requested_cid)
                self._cluster_name = name
                self._cluster_id = requested_cid

    @staticmethod
    def _format_name_id(input_name):
        if Cloud().target_cloud_aws():
            return AXClusterNameIdParser.parse_cluster_name_id_aws(input_name)
        elif Cloud().target_cloud_gcp():
            return AXClusterNameIdParser.parse_cluster_name_id_gcp(input_name)
        else:
            assert False, "Invalid cloud provider: {}. Only aws and gcp are supported".format(Cloud().target_cloud())
