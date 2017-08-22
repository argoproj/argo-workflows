# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os
import subprocess
import logging

logger = logging.getLogger(__name__)


class Cluster(object):
    def __init__(self, name, config):
        self._name = name
        self._config = config
        self._cluster_cmd = ["gcloud", "-q", "container", "clusters"]
        self._nodepool_cmd = ["gcloud", "-q", "alpha", "container", "node-pools"]
        self._common_flags = ["--zone", self._config["zone"]]
        self._hack_ax_credentials()

    def create(self):
        if self._exists():
            logger.info("Cluster %s exists already.", self._name)
            return
        create_cmd = []
        create_cmd += self._cluster_cmd
        create_cmd += ["create"]
        create_cmd += self._common_flags
        if self._config["enable-cloud-monitoring"]:
            create_cmd += ["--no-enable-cloud-monitoring"]
        if self._config["enable-cloud-logging"]:
            create_cmd += ["--no-enable-cloud-logging"]
        if self._config["preemptible"]:
            create_cmd += ["--preemptible"]
        create_cmd += ["--machine-type", self._config["machine-type"]]
        create_cmd += ["--image-type", "COS"]
        create_cmd += ["--cluster-version", "1.5.7"]
        create_cmd += ["--num-nodes", str(self._config["num-axsys-nodes"])]
        create_cmd += ["--disk-size", str(self._config["disk-size"])]
        create_cmd += ["--node-labels", "ax.tier=applatix"]
        scopes = [
            "https://www.googleapis.com/auth/devstorage.read_write",
            "https://www.googleapis.com/auth/iam",
            "https://www.googleapis.com/auth/cloud-platform",
        ]
        create_cmd += ["--scopes", ",".join(scopes)]
        create_cmd += [self._name]
        logger.info("Creating GKE cluster %s [%s] ...", self._name, create_cmd)
        subprocess.check_call(create_cmd)

        num_axuser_nodes = self._config["num-nodes"] - self._config["num-axsys-nodes"]
        max_axuser_nodes = self._config["max-nodes"] - self._config["num-axsys-nodes"]
        update_cmd = []
        update_cmd += self._nodepool_cmd
        update_cmd += ["create"]
        update_cmd += self._common_flags
        update_cmd += ["--cluster", self._name]
        if self._config["preemptible"]:
            update_cmd += ["--preemptible"]
        update_cmd += ["--enable-autoscaling"]
        update_cmd += ["--machine-type", self._config["machine-type"]]
        update_cmd += ["--image-type", "COS"]
        update_cmd += ["--num-nodes", str(num_axuser_nodes)]
        update_cmd += ["--min-nodes", str(num_axuser_nodes)]
        update_cmd += ["--max-nodes", str(max_axuser_nodes)]
        update_cmd += ["--disk-size", str(self._config["disk-size"])]
        update_cmd += ["--node-labels", "ax.tier=user"]
        update_cmd += ["--scopes", "https://www.googleapis.com/auth/devstorage.read_write"]
        update_cmd += ["axuser-pool"]
        # TODO: Run this in background to speed up install.
        logger.info("Adding user nodes to GKE cluster %s [%s] ...", self._name, update_cmd)
        subprocess.check_call(update_cmd)
        logger.info("Creating GKE cluster %s ... DONE.", self._name)

    def delete(self):
        delete_cmd = []
        delete_cmd += self._cluster_cmd
        delete_cmd += ["delete"]
        delete_cmd += self._common_flags
        delete_cmd += [self._name]
        logger.info("Deleting GKE cluster %s [%s] ...", self._name, delete_cmd)
        subprocess.check_call(delete_cmd)
        logger.info("Deleting GKE cluster %s ... DONE.", self._name)

    def download_config(self, pathname):
        download_cmd = []
        download_cmd += self._cluster_cmd
        download_cmd += ["get-credentials"]
        download_cmd += self._common_flags
        download_cmd += [self._name]
        env = {"KUBECONFIG": pathname}
        env.update(os.environ)
        logger.info("Downloading kueconfig for GKE cluster %s to %s ...", self._name, pathname)
        subprocess.check_call(download_cmd, env=env)
        logger.info("Downloading kueconfig for GKE cluster %s to %s ... DONE.", self._name, pathname)

    def _exists(self):
        list_cmd = []
        list_cmd += self._cluster_cmd
        list_cmd += ["list"]
        list_cmd += self._common_flags
        list_cmd += ["--filter", self._name]
        for l in subprocess.check_output(list_cmd).splitlines():
            if l.startswith(self._name):
                return True
        else:
            return False

    def _hack_ax_credentials(self):
        """
        Hack to set up gcloud credentials for AX account only.
        """
        # TODO: GCP_HACK Convert this module to use Google API library with ADC.
        # This cluster is called from two paths, developer kcluster and portal install.
        # Kcluster would set up developer's gcloud config inside container.
        # For portal install, we don't have a developer credential.
        # Correct way is to use API library that honor GOOGLE_APPLICATION_CREDENTIALS.
        # For now, use hacked service account for gcloud cluster operations.
        if not os.path.isfile("/root/.config/gcloud/credentials"):
            # No default credential. Likely running from portal.
            logger.info("Set up hacked AX service account credentials.")
            cmd = ["gcloud", "-q", "auth", "activate-service-account"]
            cmd += ["--key-file", "/ax/config/ax-gcloud-editor.json"]
            subprocess.check_call(cmd)
            account_id = os.getenv("GOOGLE_CLOUD_PROJECT")
            assert account_id is not None
            cmd = ["gcloud", "-q", "config", "set", "project", account_id]
            subprocess.check_call(cmd)
