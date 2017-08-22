#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Platform startup for GKE cluster
"""

import os
import time
import logging
import subprocess
import yaml

from ax.kubernetes.ax_kube_dict import AXNameSpaces

from .platform import AXPlatform
from .ax_platform_config import KUBERNETES_SET, CREATE_SET, START_SET, START_SET_KUBE_SYSTEM

logger = logging.getLogger(__name__)


class AXGKEPlatform(AXPlatform):
    def start(self):
        """
        Start platform running services, mostly deployments.
        """
        logger.info("Starting Argo services for GKE cluster %s", self._cluster_name_id)
        logger.info("\n%s\nImage Namespace: %s\nImage Version: %s\n%s\n",
                    "=" * 40, self._image_namespace, self._image_version, "=" * 40)

        self._set_ext_dns()
        self.create_objects(objects=START_SET_KUBE_SYSTEM, namespace=AXNameSpaces.AXSYS)
        self.create_objects(objects=["kafka-zk-svc"], namespace=AXNameSpaces.AXSYS)
        self._create_axdb_stateful_set()
        self.create_objects(objects=START_SET - frozenset(["kafka-zk-svc", "axdb-svc", "cron"]),
                            namespace=AXNameSpaces.AXSYS)
        self._update_version()
        logger.info("New cluster is %s", self._cluster_name_id)

    def stop(self):
        """
        Stop platform running services, mostly deployments.
        """
        logger.info("Stopping Argo services for GKE cluster %s", self._cluster_name_id)
        self.delete_objects(START_SET - frozenset(["axdb-svc"]))
        self._delete_axdb_stateful_set()
        self.delete_objects(START_SET_KUBE_SYSTEM)

    def create(self, volume_only=False):
        """
        Create platform persistent volumes. Also call start() by default.
        """
        # Same as AWS now. Calling parent.
        super(AXGKEPlatform, self).create(volume_only)

    def delete(self, volume_only=False):
        """
        Delete platform persistent volumes.
        """
        if not volume_only:
            self.stop()

        self.delete_objects(CREATE_SET)
        self._delete_by_namespaces_and_wait()

    def install(self):
        """
        Install platform of a GKE cluster. Mostly for axops load balancer.
        Also call create() by default.
        """
        logger.info("Installing Argo software for GKE cluster %s", self._cluster_name_id)
        self.create_namespaces()
        self.create_objects(objects=KUBERNETES_SET, namespace=AXNameSpaces.AXSYS)

        self.create()

    def cleanup(self):
        """
        Uninstall platform of a GKE cluster. Mostly for axops load balancer.
        """
        logger.info("Uninstalling Argo software for GKE cluster %s", self._cluster_name_id)
        self.delete()
        self.delete_objects(objects=KUBERNETES_SET, namespace=AXNameSpaces.AXSYS)

    def _delete_axdb_stateful_set(self):
        # TODO: Upgrade to official kubernetes client to fix this.
        # Use kubectl to delete axdb statefule set.
        kubectl_cmd = ["kubectl", "--kubeconfig", "/tmp/ax_kube/cluster_{}.conf".format(self._cluster_name_id)]
        subprocess.call(kubectl_cmd + ["delete", "statefulsets", "axdb", "--namespace", "axsys"])
        subprocess.call(kubectl_cmd + ["delete", "svc", "axdb", "--namespace", "axsys"])

    def _create_axdb_stateful_set(self):
        # TODO: Upgrade to official kubernetes client to fix this.
        # Hack to creat axdb stateful set.
        # Current swagger client is too old to handle stateful sets.
        # Use kubectl to create and wait for it.
        # Keep this hack to be self contained and easy to change.
        import tempfile
        from ax.util.macro import macro_replace
        kubectl_cmd = ["kubectl", "--kubeconfig", "/tmp/ax_kube/cluster_{}.conf".format(self._cluster_name_id)]

        with open("/ax/config/service/axdb-svc-stateful.yml.in", "r") as f:
            data = f.read()
        data = macro_replace(data, self._replacing)
        config = yaml.load_all(data)
        for conf in config:
            try:
                logger.info("Creating %s %s ...", conf["kind"], conf["metadata"]["name"])
                if conf["kind"] == "StatefulSet":
                    # TODO: Fix multiplier.
                    for c in conf["spec"]["template"]["spec"]["containers"]:
                        c["env"] += [
                            {"name": "CPU_MULT", "value": "1"},
                            {"name": "MEM_MULT", "value": "1"},
                            {"name": "DISK_MULT", "value": "1"},
                        ]
                tmp = tempfile.NamedTemporaryFile(delete=False)
                tmp.write(yaml.dump(data=conf, default_flow_style=False))
                tmp.close()
                subprocess.check_call(kubectl_cmd + [
                    "create",
                    "--namespace", "axsys",
                    "-f", tmp.name])
            except:
                logger.exception("Failed to create %s %s.", conf["kind"], conf["metadata"]["name"])
            finally:
                try:
                    os.unlink(tmp.name)
                except:
                    pass
        for po in ["axdb-0", "axdb-1", "axdb-2"]:
            for _ in range(30):
                try:
                    out = subprocess.check_output(kubectl_cmd + ["get", "pods", po, "--namespace", "axsys"])
                    if "Running" in out:
                        break
                except subprocess.CalledProcessError as e:
                    logger.info("Waiting for pod %s %s ...", po, e.output)
                time.sleep(10)
            else:
                assert 0, "Pod {} not started.".format(po)
