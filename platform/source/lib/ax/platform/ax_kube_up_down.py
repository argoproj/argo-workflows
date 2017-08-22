#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import os
import logging

from ax.kubernetes.kube_up_down import KubeUpDown
from ax.platform.ax_cluster_info import AXClusterInfo

logger = logging.getLogger(__name__)


class AXKubeUpDown(object):
    """
    AX cluster bootstrap class.
    """

    def __init__(self, cluster_name_id, env=None, aws_profile=None):
        """
        :param cluster_name_id: String for cluster name_id, e.g. lcj-cluster-515d9828-7515-11e6-9b3e-a0999b1b4e15
        :param env: all environment variables for kube-up and kube-down.
        :param aws_profile: AWS profile used to access AWS account.
        """
        self._name_id = cluster_name_id
        self._aws_profile = aws_profile
        self._cluster_info = AXClusterInfo(cluster_name_id=cluster_name_id, aws_profile=aws_profile)
        self._kube_conf = self._cluster_info.get_kube_config_file_path()

        root = os.getenv("AX_KUBERNETES_ROOT")
        assert root, "Must set AX_KUBERNETES_ROOT to kubernetes directory"
        assert os.path.isdir(root), "AX_KUBERNETES_ROOT must be directory"

        self._kube = KubeUpDown(root, env)

    def up(self):
        """
        Bring up cluster and save kube_config in portal.
        """
        try:
            self._kube.up()
        finally:
            # Kube-up creates ssh key first. Try to save ssh key first.
            # We try to save keys/config (if generated) even if kube_up fails
            self._cluster_info.upload_kube_key()
            self._cluster_info.upload_kube_config()
        logger.info("New cluster id is %s", self._name_id)

    def down(self):
        """
        Get kube config from portal and shutdown cluster based on this config.
        """
        self._kube.down()
