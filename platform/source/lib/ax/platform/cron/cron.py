#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import time
import logging
import subprocess
import os
import tempfile
import shutil

import boto3
from retrying import retry
from future.utils import with_metaclass

from ax.util.monotonic_time import get_monotonic_time
from ax.util.singleton import Singleton
from ax.util.const import SECONDS_PER_DAY, SECONDS_PER_HOUR, SECONDS_PER_MINUTE

from ax.meta import AXSupportConfigPath, AXClusterId, AXCustomerId

logger = logging.getLogger(__name__)


class AXCron(with_metaclass(Singleton, object)):

    def __init__(self):
        self._cluster_name_id = AXClusterId().get_cluster_name_id()
        self._cluster_name = AXClusterId().get_cluster_name()
        self._cluster_id = AXClusterId().get_cluster_id()
        self._account = AXCustomerId().get_customer_id()

        self._sleep_interval = SECONDS_PER_MINUTE

        self._hourly = SECONDS_PER_HOUR
        self._daily = SECONDS_PER_DAY
        self._last_hourly = -self._hourly
        self._last_daily = -self._daily

        self._elasticsearch_host = "elasticsearch"
        logger.debug("AX account: %s cluster_id: %s", self._account, self._cluster_name_id)

    def start(self):
        while True:
            current = get_monotonic_time()
            if current > self._last_hourly + self._hourly:
                logger.debug("Run hourly tasks")
                self.run_hourly()
                self._last_hourly = current
            if current > self._last_daily + self._daily:
                logger.debug("Run daily tasks")
                self.run_daily()
                self._last_daily = current
            time.sleep(self._sleep_interval)

    def run_once(self):
        self.run_daily()
        self.run_hourly()

    def run_daily(self):
        pass

    def run_hourly(self):
        self.run_support()

    def run_support(self):
        logger.info("Collecting support ...")
        self.kube_cluster()
        logger.info("Done collecting support.")

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def upload_file(self, s3, local_path, s3_path):
        s3.upload_file(local_path, AXSupportConfigPath(self._cluster_name_id).bucket(), s3_path, ExtraArgs={"ACL": "bucket-owner-full-control"})

    def kube_cluster(self):
        logger.info("Collecting kubectl cluster-info ...")
        tmp = None
        try:
            tmp = tempfile.mkdtemp(prefix="support-collect-")
            cmd = ["kubectl", "cluster-info", "dump", "--namespaces", "kube-system,axsys", "--output-directory", tmp]
            subprocess.call(cmd)
            with open(os.path.join(tmp, "user_jobs.txt"), "w") as f:
                subprocess.call(["kubectl", "get", "jobs", "--namespace", "axuser", "-o", "yaml"], stdout=f)
            with open(os.path.join(tmp, "user_pods.txt"), "w") as f:
                subprocess.call(["kubectl", "get", "pods", "--namespace", "axuser", "-o", "yaml"], stdout=f)
            with open(os.path.join(tmp, "pvcs.txt"), "w") as f:
                subprocess.call(["kubectl", "get", "pvc", "--all-namespaces", "-o", "yaml"], stdout=f)
            with open(os.path.join(tmp, "volumes.txt"), "w") as f:
                subprocess.call(["kubectl", "get", "pv", "-o", "yaml"], stdout=f)
            s3 = boto3.Session().client("s3", aws_access_key_id=os.environ.get("ARGO_S3_ACCESS_KEY_ID", None),
                aws_secret_access_key=os.environ.get("ARGO_S3_ACCESS_KEY_SECRET", None),
                endpoint_url=os.environ.get("ARGO_S3_ENDPOINT", None))
            prefix = AXSupportConfigPath(self._cluster_name_id).support() + "/" + time.strftime("%Y-%m-%d/%H.%M.%S")
            for dir, _, names in os.walk(tmp):
                for f in names:
                    src_path = "{}/{}".format(dir, f)
                    dst_path = src_path.replace(tmp, prefix)
                    self.upload_file(s3, src_path, dst_path)
        except Exception:
            logger.exception("Failed to upload cluster-info.")
        finally:
            if tmp is not None:
                shutil.rmtree(tmp)
        logger.info("Done with kubectl cluster-info.")
