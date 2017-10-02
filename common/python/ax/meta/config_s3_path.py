#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import uuid
import logging
import time
import os

from future.utils import with_metaclass

from ax.util.singleton import Singleton
from ax.cloud import Cloud
from ax.meta import AXCustomerId, AXClusterNameIdParser

logger = logging.getLogger(__name__)


class AXConfigBase(with_metaclass(Singleton, object)):

    CLUSTER_S3_ARTIFACT_EXTERNAL_PREFIX = "{name}/{id}/artifacts"
    CLUSTER_S3_ARTIFACT_INTERNAL_PREFIX = "{name}/{id}/artifacts"

    def __init__(self, name_id):
        # Account is the source of truth of all S3 config related operations as we
        # use one bucket per customer account and customer id is the unique identifier
        # of all buckets related to that customer
        self._account = AXCustomerId().get_customer_id()

        # Different configs are stored in different bucket, so set it to None
        # in base class
        self._bucket_name = None

        # We use cluster_name/id/ to separate different directories for different
        # cluster under same account
        self._cluster_name_id = name_id

        # Set a default value but will be overridden by classes inheriting it. External
        # means these configs are stored in the account that is different than the one
        # cluster is running in, for example, we upload support logs to a support AWS
        # account, but upload artifacts to customer account.
        self._external = False

        # Bucket exists is a sign to see if the corresponding S3 path are valid (i.e. this
        # class can be used. For Support, this would point to the support bucket, for cluster
        # artifacts, this will point to data bucket, etc.)
        self._bucket_exists = None

        # We have different naming scheme for GCP and AWS so we parse cluster
        # name_id in a different way. This series of classes enforces a name_id
        # be passes as "<cluster_name>-<cluster_id>"
        if Cloud().target_cloud_gcp():
            self._cluster_name, self._cluster_id = AXClusterNameIdParser.parse_cluster_name_id_gcp(name_id)
        elif Cloud().target_cloud_aws():
            self._cluster_name, self._cluster_id = AXClusterNameIdParser.parse_cluster_name_id_aws(name_id)
        else:
            assert False, "Invalid cloud provider: {}. Only aws and gcp are supported".format(Cloud().target_cloud())

        assert self._cluster_name and self._cluster_id, "Failed to extract cluster name and id from [{}]".format(name_id)

    def bucket(self):
        return self._bucket_name

    def bucket_exists(self):
        if self._bucket_exists is None:
            self._bucket_exists = Cloud().get_bucket(self._bucket_name).exists()
        return self._bucket_exists

    def artifact(self):
        # There is no good reason for this. It's just names are already created this way.
        if self._external:
            template = self.CLUSTER_S3_ARTIFACT_EXTERNAL_PREFIX
        else:
            template = self.CLUSTER_S3_ARTIFACT_INTERNAL_PREFIX
        return template.format(name=self._cluster_name, id=self._cluster_id)

    def is_external(self):
        return self._external


class AXUpgradeConfigPath(with_metaclass(Singleton, AXConfigBase)):

    UPGRADE_S3_CURRENT_VERSIONS_KEY = "{name_id}/current_versions"
    UPGRADE_S3_TARGET_VERSIONS_KEY = "{name_id}/target_versions"
    UPGRADE_S3_UPGRADE_WINDOW_KEY = "{name_id}/upgrade_window"
    UPGRADE_S3_TAG_KEY = "{name_id}/tag"

    def __init__(self, name_id):
        super(AXUpgradeConfigPath, self).__init__(name_id)
        self._bucket_template = "applatix-upgrade-{account}-{seq}"
        self._bucket_name = os.getenv("ARGO_DATA_BUCKET_NAME") or self._bucket_template.format(account=self._account, seq=0)
        self._external = True
        logger.info("Using AX upgrade config path %s", self._bucket_name)

    def tag(self):
        return self.UPGRADE_S3_TAG_KEY.format(name_id=self._cluster_name_id)

    def target_versions(self):
        return self.UPGRADE_S3_TARGET_VERSIONS_KEY.format(name_id=self._cluster_name_id)

    def upgrade_window(self):
        return self.UPGRADE_S3_UPGRADE_WINDOW_KEY.format(name_id=self._cluster_name_id)


class AXClusterConfigPath(with_metaclass(Singleton, AXConfigBase)):

    CLUSTER_S3_OBJECT_COMMON_PREFIX = "{name}/{id}"
    CLUSTER_S3_KUBE_CONFIG_KEY = "{}/kube_config".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_KUBE_SSH_KEY = "{}/kube_ssh".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_CLUSTER_CONFIG_KEY = "{}/cluster_config".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_INSTALL_STAGE_0_KEY = CLUSTER_S3_CLUSTER_CONFIG_KEY
    CLUSTER_S3_INSTALL_STAGE_1_KEY = "{}/stage1".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_INSTALL_STAGE_2_KEY = "{}/stage2".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_VERSIONS_KEY = "{}/current_versions".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_MASTER_CONFIG_DIR = "{}/master_config/".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)     # The trailing / is required.
    CLUSTER_S3_MASTER_ATTRIBUTES = CLUSTER_S3_MASTER_CONFIG_DIR + "attributes"
    CLUSTER_S3_MASTER_USER_DATA = CLUSTER_S3_MASTER_CONFIG_DIR + "user-data"
    CLUSTER_S3_STATE_BEFORE_PAUSE = "{}/state_before_pause".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_METADATA = "{}/metadata/v1/metadata.yaml".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_TERRAFORM_DIR = "{}/terraform/".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_PLATFORM_MANIFEST_DIR = "{}/platform/manifests/".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_PLATFORM_CONFIG = "{}/platform/config".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_CURRENT_STATE = "{}/current_state".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)
    CLUSTER_S3_PORTAL_SUPPORT = "{}/portal_support".format(CLUSTER_S3_OBJECT_COMMON_PREFIX)

    def __init__(self, name_id):
        super(AXClusterConfigPath, self).__init__(name_id)
        self._bucket_template = "applatix-cluster-{account}-{seq}"
        self._bucket_name = os.getenv("ARGO_DATA_BUCKET_NAME") or self._bucket_template.format(account=self._account, seq=0)
        self._external = False
        logger.info("Using AX cluster config path %s", self._bucket_name)

    def cluster_install_stage0_key(self):
        return self.CLUSTER_S3_INSTALL_STAGE_0_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def cluster_install_stage1_key(self):
        return self.CLUSTER_S3_INSTALL_STAGE_1_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def cluster_install_stage2_key(self):
        return self.CLUSTER_S3_INSTALL_STAGE_2_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def kube_config(self):
        return self.CLUSTER_S3_KUBE_CONFIG_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def kube_ssh(self):
        return self.CLUSTER_S3_KUBE_SSH_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def cluster_config(self):
        return self.CLUSTER_S3_CLUSTER_CONFIG_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def master_config_dir(self):
        return self.CLUSTER_S3_MASTER_CONFIG_DIR.format(name=self._cluster_name, id=self._cluster_id)

    def master_attributes_path(self):
        return self.CLUSTER_S3_MASTER_ATTRIBUTES.format(name=self._cluster_name, id=self._cluster_id)

    def master_user_data_path(self):
        return self.CLUSTER_S3_MASTER_USER_DATA.format(name=self._cluster_name, id=self._cluster_id)

    def state_before_pause(self):
        return self.CLUSTER_S3_STATE_BEFORE_PAUSE.format(name=self._cluster_name, id=self._cluster_id)

    def cluster_metadata(self):
        return self.CLUSTER_S3_METADATA.format(name=self._cluster_name, id=self._cluster_id)

    def versions(self):
        return self.CLUSTER_S3_VERSIONS_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def terraform_dir(self):
        return self.CLUSTER_TERRAFORM_DIR.format(name=self._cluster_name, id=self._cluster_id)

    def platform_manifest_dir(self):
        return self.CLUSTER_S3_PLATFORM_MANIFEST_DIR.format(name=self._cluster_name, id=self._cluster_id)

    def platform_config(self):
        return self.CLUSTER_S3_PLATFORM_CONFIG.format(name=self._cluster_name, id=self._cluster_id)

    def current_state(self):
        return self.CLUSTER_S3_CURRENT_STATE.format(name=self._cluster_name, id=self._cluster_id)

    def portal_support(self):
        return self.CLUSTER_S3_PORTAL_SUPPORT.format(name=self._cluster_name, id=self._cluster_id)


class AXClusterDataPath(with_metaclass(Singleton, AXConfigBase)):
    def __init__(self, name_id):
        super(AXClusterDataPath, self).__init__(name_id)
        self._bucket_template = "applatix-data-{account}-{seq}"
        self._bucket_name = os.getenv("ARGO_DATA_BUCKET_NAME") or self._bucket_template.format(account=self._account, seq=0)
        self._external = False
        logger.info("Using AX cluster data path %s", self._bucket_name)


class AXSupportConfigPath(with_metaclass(Singleton, AXConfigBase)):

    SUPPORT_S3_TAG_KEY = "{name}/{id}/tag"
    SUPPORT_S3_CURRENT_VERSION_KEY = "{name}/{id}/current_versions"
    SUPPORT_S3_VERSION_HISTORY_KEY = "{name}/{id}/version_history/{time}"
    SUPPORT_S3_UPLOAD_KEY = "{name}/{id}/support"

    def __init__(self, name_id):
        super(AXSupportConfigPath, self).__init__(name_id)
        self._bucket_template = "applatix-cluster-{account}-{seq}"
        self._bucket_name = os.getenv("ARGO_LOG_BUCKET_NAME") or self._bucket_template.format(account=self._account, seq=0)
        self._external = True
        logger.info("Using AX support config path %s", self._bucket_name)

    def tag(self):
        return self.SUPPORT_S3_TAG_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def current_versions(self):
        return self.SUPPORT_S3_CURRENT_VERSION_KEY.format(name=self._cluster_name, id=self._cluster_id)

    def version_history(self):
        return self.SUPPORT_S3_VERSION_HISTORY_KEY.format(name=self._cluster_name, id=self._cluster_id,
                                                          time=time.strftime("%Y-%m-%d-%H-%M-%S"))

    def support(self):
        return self.SUPPORT_S3_UPLOAD_KEY.format(name=self._cluster_name, id=self._cluster_id)


class AXLogPath(object):
    """
    Logical bucket pathnames.
    Intended for workflow artifact log use only.
    """
    def __new__(cls, name_id):
        if AXSupportConfigPath(name_id).bucket_exists():
            return AXSupportConfigPath(name_id)
        else:
            return AXClusterConfigPath(name_id)

