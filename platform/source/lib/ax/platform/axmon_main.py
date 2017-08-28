#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module for AXmon service
"""
import logging
import os
import threading
import time
import json

try:
    from urlparse import urlparse
except:
    from urllib.parse import urlparse

from future.utils import with_metaclass

from ax.exceptions import AXNotFoundException
from ax.cloud import Cloud
from ax.platform.component_config import SoftwareInfo
from ax.platform.secrets import SecretsManager
from ax.version import __version__
from ax.kubernetes.client import KubernetesApiClient
from ax.util.singleton import Singleton
from ax.util.callbacks import ContainerEventCallbacks
from ax.util.docker_client import AXDockerClient
from .task import Task
from .volumes import VolumeManager

from ax.platform.ax_monitor_helper import KubeObjStatusCode
from ax.platform.ax_monitor import AXKubeMonitor

logger = logging.getLogger('ax.platform.axmon')

AXMON_DEFAULT_PORT = 8901
TASK_WAIT_TIMEOUT = 5 * 60
AX_CLUSTER_NAME_ID = os.getenv("AX_CLUSTER_NAME_ID")


class AXMon(with_metaclass(Singleton, object)):

    def __init__(self):
        super(AXMon, self).__init__()
        self.version = __version__

        self._cluster_cond = threading.Condition()
        self._shutdown = False

        self._kubectl = KubernetesApiClient(use_proxy=True)

        # Initialize SoftwareInfo singleton
        self._software_info = SoftwareInfo()

        if Cloud().target_cloud_aws():
            # init the volume manager singleton
            VolumeManager()

    def task_create(self, data):
        """
        Create a task
        """
        logger.debug("Task create call for {}".format(json.dumps(data)))
        conf = Task.insert_defaults(data)
        dry_run = data.get("dry_run", False)
        if dry_run:
            name = Task.generate_name(conf)
            logger.debug("task_create: dry_run name is {}".format(name))
            return {"{}".format(name): KubeObjStatusCode.OK}

        # create the workflow
        task_id = Task.generate_name(conf)
        task = Task(task_id)
        job_obj = task.create(conf)
        task.start(job_obj)
        return {"{}".format(task.name): KubeObjStatusCode.OK}

    def task_show(self, task_id):
        if task_id is None:
            raise ValueError("task id must be specified")
        task = Task(task_id)
        return task.status()

    def task_delete(self, task_id, force):
        logger.debug("Deleting task %s force=%s", task_id, force)
        if task_id is None:
            raise ValueError("task id must be specified")
        task = Task(task_id)
        return task.delete(force=force)

    def task_stop_running_pod(self, task_id):
        logger.debug("Stopping task %s", task_id)
        if task_id is None:
            raise ValueError("task id must be specified")
        task = Task(task_id)
        return task.stop()

    def task_log(self, task_id):
        logger.debug("Getting log endpoint for %s", task_id)
        if task_id is None:
            raise ValueError("task id must be specified")
        task = Task(task_id)
        return task.get_log_endpoint()

    def add_registry(self, server, username, password, save=True):
        client = AXDockerClient()
        indexserver = server
        # if server is docker hub then fix the registry server
        if server == "docker.io":
            indexserver = "https://index.docker.io/v1/"
        token = client.login(indexserver, username, password)
        if not save:
            return
        full_token = AXDockerClient.generate_kubernetes_image_secret(indexserver, token)
        dns_name = urlparse("https://{}".format(server)).netloc
        s = SecretsManager()
        # for now all client pods are started in axuser namespace
        s.insert_imgpull(dns_name, "axuser", full_token)

    def delete_registry(self, server):
        s = SecretsManager()
        dns_name = urlparse("https://{}".format(server)).netloc
        if s.get_imgpull(dns_name, "axuser"):
            s.delete_imgpull(dns_name, "axuser")
        else:
            raise AXNotFoundException("Registry server {} not registered with platform".format(dns_name))

    def _do_set_dnsname(self, dnsname):
        import subprocess
        logger.debug("Setting cluster dnsname to %s", dnsname)
        cmd = ["/ax/bin/restart_pod_for_eip.sh", dnsname]
        subprocess.check_call(cmd)
        logger.info("Set cluster name %s successfully", dnsname)

    def set_dnsname(self, dnsname):
        t = threading.Thread(name="set_dnsname", target=self._do_set_dnsname, args=(dnsname,))
        t.start()
        time.sleep(2)
        logger.debug("Setting dnsname returning OK")

    def shutdown(self):
        logger.info("AXMon exiting, send notification")
        self._shutdown = True
        with self._cluster_cond:
            self._cluster_cond.notifyAll()
        logger.info("AXMon exiting")

    def run(self):
        """AXMon main thread"""
        import signal
        import sys

        def signal_handler(signal, frame):
            logger.info("AXMon killed with signal %s", signal)
            sys.exit(0)
        signal.signal(signal.SIGTERM, signal_handler)
        signal.signal(signal.SIGINT, signal_handler)

        # now that axdb is running post events
        cbs = ContainerEventCallbacks()
        from ax.platform.stats import container_oom_cb
        cbs.add_cb(container_oom_cb)

        kube_monitor = AXKubeMonitor()
        kube_monitor.reload_monitors()
        kube_monitor.start()

        logger.info("kube monitor started")

        try:
            while True:
                time.sleep(1)
        except (KeyboardInterrupt, SystemExit):
            self.shutdown()

    def run_test(self, testname, data):
        return {}
