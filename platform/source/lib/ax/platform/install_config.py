#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Config module used only for kubernetes cluster bring up.
No other code should access this at normal run time.

Install config is stored at portal account S3. Install script downloads config to local
environment and doesn't need portal data any more.

Portal code would generate these data and start cluster bootstrap. <cookie> can be any unique string,
likely UUID.
For internal use, <cookie> is cluster-name developer specifies.

S3 path:
/ax-install-config/<cookie>/name:
        Cluster name (likely without UUID)
/ax-install-config/<cookie>/version:
        Json config for cluster version {"namespace": "<namespace>", "version": "<version>"}
/ax-install-config/<cookie>/config:
        Json config for AX cluster config.
"""

import json
import os
import logging

import boto3

from future.utils import with_metaclass

from ax.util.singleton import Singleton
from ax.aws.meta_data import AWSMetaData
from ax.platform_client.env import AXEnv

logger = logging.getLogger(__name__)


class AXInstallConfig(with_metaclass(Singleton, object)):

    def __init__(self, aws_profile=None):
        self._aws_profile = aws_profile
        self._name = None
        self._namespace = None
        self._version = None
        self._config_file = None

        # This runs from creator only.
        if AXEnv().is_in_pod() or AXEnv().on_kube_host():
            return

        cookie = AWSMetaData().get_user_data(attr="ax-install-cookie")
        if cookie is None:
            # For testing only.
            cookie = os.getenv("AX_INSTALL_COOKIE", None)
            if cookie is None:
                return

        # Use different bucket for AX testing.
        if cookie.startswith("ax-internal-"):
            self._bucket = "ax-install-config-dev"
            cookie = cookie.replace("ax-internal-", "")
        else:
            self._bucket = "ax-install-config"
        self._path = cookie
        self._get_install_config()

    def get_name(self):
        return self._name

    def get_namespace(self):
        return self._namespace

    def get_version(self):
        return self._version

    def get_config_file_path(self):
        return self._config_file

    def _get_install_config(self):
        try:
            s3 = boto3.Session(profile_name=self._aws_profile).client("s3")
            data = s3.get_object(Bucket=self._bucket, Key="{}/version".format(self._path))["Body"].read()
            self._namespace = json.loads(data)["namespace"]
            self._version = json.loads(data)["version"]
            data = s3.get_object(Bucket=self._bucket, Key="{}/name".format(self._path))["Body"].read()
            self._name = str(data).strip()
            config_path = "/tmp/cluster_config.json"
            s3.download_file(Bucket=self._bucket, Key="{}/config".format(self._path), Filename=config_path)
            self._config_file = config_path
        except Exception:
            logger.exception("Failed to get install config")
