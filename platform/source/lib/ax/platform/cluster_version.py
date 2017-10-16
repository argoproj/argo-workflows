#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import yaml
import logging
import os

from ax.cloud import Cloud
from ax.meta import AXClusterId, AXClusterConfigPath, AXSupportConfigPath
import requests
from retrying import retry


logger = logging.getLogger(__name__)


class AXVersion(object):
    def __init__(self, customer_id, cluster_name_id, aws_profile):
        self._customer_id = customer_id
        self._cluster_name_id = cluster_name_id
        self._cluster_name = AXClusterId(cluster_name_id).get_cluster_name()
        self._aws_profile = aws_profile

        cluster_bucket_name = AXClusterConfigPath(cluster_name_id).bucket()
        self._cluster_bucket = Cloud().get_bucket(cluster_bucket_name, aws_profile=self._aws_profile)

        support_bucket_name = AXSupportConfigPath(cluster_name_id).bucket()
        self._support_bucket = Cloud().get_bucket(support_bucket_name, aws_profile=self._aws_profile)

    def update(self, new):
        self._report_version_to_s3(new)

    def _get_current_version(self):
        # TODO: combine cluster bucket operations to AXClusterInfo object
        data = self._cluster_bucket.get_object(key=AXClusterConfigPath(self._cluster_name_id).versions())
        return yaml.load(data) if data else {}

    def _report_version_to_s3(self, new):
        old = self._get_current_version()
        history = {"from": old, "to": new}
        # Update current version in cluster bucket.
        cluster_version_key = AXClusterConfigPath(self._cluster_name_id).versions()
        self._cluster_bucket.put_object(cluster_version_key,
                                        yaml.dump(new))

        # Update current version in support bucket.
        support_version_key = AXSupportConfigPath(self._cluster_name_id).current_versions()
        self._support_bucket.put_object(support_version_key,
                                        yaml.dump(new))

        # Update version history in support bucket.
        support_version_history_key = AXSupportConfigPath(self._cluster_name_id).version_history()
        self._support_bucket.put_object(support_version_history_key,
                                        yaml.dump(history))
