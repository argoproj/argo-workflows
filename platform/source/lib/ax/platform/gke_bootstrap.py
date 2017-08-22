#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os
import tempfile
import yaml
import logging

from kubernetes import config as kube_config
from kubernetes import client as kube_client
from kubernetes.client.rest import ApiException

from ax.cloud.gke.cluster import Cluster
from ax.kubernetes.namespace import Namespace

logger = logging.getLogger(__name__)

# TODO: Connect this with cluster_config.py
test_gke_config = {
    "zone": "us-west1-a",
    "num-nodes": 3,
    "num-axsys-nodes": 2,
    "max-nodes": 5,
    "machine-type": "n1-standard-2",
    "disk-size": 100,
    "enable-cloud-monitoring": False,
    "enable-cloud-logging": False,
    "preemptible": True,
}

DEFAULT_KUBE_CONFIG_PATH = "/tmp/ax_kube/cluster_{name_id}.conf"
DEFAULT_SERVICE_CONFIG_PATH = "/ax/config/service"


class GKE(object):
    def __init__(self, name_id):
        self._name_id = name_id
        self._gke_cluster = Cluster(self._name_id, test_gke_config)

    def up(self):
        self._gke_cluster.create()
        self._kube_config_setup()
        #self._cluster_init()

    def down(self):
        try:
            self._kube_config_setup()
            # TODO: Need to delete axops service and load balancer here.
            self._gke_cluster.delete()
        except:
            pass

    def _cluster_init(self):
        v1 = kube_client.CoreV1Api()
        api_client = kube_client.ApiClient()
        count = 0
        for i in v1.list_node().items:
            if count >= test_gke_config["num-axsys-nodes"]:
                v1.patch_node(i.metadata.name, {"metadata": {"labels": {"ax.tier": "user"}}})
            else:
                v1.patch_node(i.metadata.name, {"metadata":{"labels":{"ax.tier":"applatix"}}})
            count += 1
        for ns in ["axsys", "axuser"]:
            Namespace(ns).create()
            Namespace(ns).create()
            with open(os.path.join(DEFAULT_SERVICE_CONFIG_PATH, "registry-secrets.yml.in")) as f:
                spec = yaml.load(f)
            secret_obj = api_client._ApiClient__deserialize(spec, kube_client.V1Secret)
            try:
                v1.create_namespaced_secret(ns, secret_obj)
            except ApiException as e:
                if e.status != 409 or "AlreadyExists" not in e.body:
                    raise

    def _kube_config_setup(self):
        config_file = DEFAULT_KUBE_CONFIG_PATH.format(name_id=self._name_id)
        logger.info("Setting up kubeconfig environment at %s ...", config_file)
        self._gke_cluster.download_config(config_file)
        self._hack_config_file(config_file)
        logger.info("Setting up kubeconfig environment at %s ... DONE.", config_file)
        kube_config.load_kube_config(config_file)

    def _hack_config_file(self, path):
        # Different kubectl uses different config auth format.
        # Hack it to use format recognized by kubectl 1.5.7.
        # Remove absolute pathname as axclustermanager, laptop and dev machines have different paths.
        # Rely on PATH variable being set correctly to work.
        # See https://fossies.org/diffs/kubernetes/1.5.5_vs_1.6.0/staging/src/k8s.io/client-go/plugin/pkg/client/auth/gcp/gcp.go-diff.html
        tmp = tempfile.NamedTemporaryFile(delete=False, dir=os.path.dirname(path))
        with open(path, "r") as f:
            config = yaml.load(f)
        logger.debug("hack path before %s", config["users"][0]["user"]["auth-provider"]["config"]["cmd-path"])
        if "config-helper" not in config["users"][0]["user"]["auth-provider"]["config"]["cmd-path"]:
            config["users"][0]["user"]["auth-provider"]["config"]["cmd-path"] = "gcloud config config-helper --format=json"
        logger.debug("hack path after %s", config["users"][0]["user"]["auth-provider"]["config"]["cmd-path"])
        yaml.dump(config, tmp, default_flow_style=False)
        tmp.close()
        os.rename(tmp.name, path)
