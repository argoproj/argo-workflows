# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import abc
import logging
import os
from future.utils import with_metaclass

from ax.meta import AXClusterId
from ax.platform.cluster_config import AXClusterConfig
from ax.platform.ax_cluster_info import AXClusterInfo


logger = logging.getLogger(__name__)


# Path used for downloading cluster's existing manifest and platform config
# from S3 during pause / restart / upgrade
TEMP_PLATFORM_MANIFEST_ROOT = "/tmp/platform/manifests/"
TEMP_PLATFORM_CONFIG_PATH = "/tmp/platform/platform-bootstrap.cfg"


def ensure_manifest_temp_dir():
    if not os.path.exists(TEMP_PLATFORM_MANIFEST_ROOT):
        os.makedirs(TEMP_PLATFORM_MANIFEST_ROOT)
    else:
        assert os.path.isdir(TEMP_PLATFORM_MANIFEST_ROOT), "Please make sure {} is a directory for downloading cluster platform manifests".format(TEMP_PLATFORM_MANIFEST_ROOT)


def check_cluster_staging(cluster_info_obj, stage):
        """
        stage0: pre-install
        stage1: bring up Kubernetes cluster and
        stage2: right before ax installation finishes

        - If we see stage0 information, we can assume there are already part of the cluster created
        - If we see stage1 information, we can assume Kubernetes cluster has been created successfully,
            kube_config and kube_ssh have been uploaded to s3
        - If we see stage2 information, we can assume there is a running Argo cluster

        :param cluster_info_obj: AXClusterInfo object
        :param stage: "stage0" or "stage1", or "stage2"
        :return: True if stage is already there, False otherwise
        """
        assert stage in ["stage0", "stage1", "stage2"], "Only stage0, stage1, and stage2 information is available"
        logger.info("Checking Argo install %s ...", stage)
        stage_exist_description = {
            "stage0": "Argo cluster installation has started",
            "stage1": "Argo cluster infrastructure exists",
            "stage2": "Argo software is running"
        }
        stage_not_exist_description = {
            "stage0": "Argo cluster installation has not started",
            "stage1": "Argo cluster infrastructure does not exist",
            "stage2": "Argo software is not running"
        }
        try:
            cluster_info_obj.download_staging_info(stage)
            logger.info("Checking Argo install %s ... DONE. %s", stage, stage_exist_description[stage])
            return True
        except Exception:
            logger.info("Checking Argo install %s ... DONE. %s", stage, stage_not_exist_description[stage])
            return False


class ClusterOperationBase(with_metaclass(abc.ABCMeta, object)):
    def __init__(self, cluster_name, cluster_id=None, cloud_profile=None):
        if cluster_id:
            input_name = "{}-{}".format(cluster_name, cluster_id)
        else:
            input_name = cluster_name

        self._idobj = AXClusterId(name=input_name, aws_profile=cloud_profile)

    @abc.abstractmethod
    def run(self):
        pass


class CommonClusterOperations(object):
    def __init__(self, input_name, cloud_profile):
        """
        :param input_name: cluster name or <cluster_name>-<cluster_id> format
        :param cloud_profile:
        """
        name_id = AXClusterId(name=input_name, aws_profile=cloud_profile).get_cluster_name_id()
        self.cluster_config = AXClusterConfig(
            cluster_name_id=name_id,
            aws_profile=cloud_profile
        )
        self.cluster_info = AXClusterInfo(
            cluster_name_id=name_id,
            aws_profile=cloud_profile
        )

