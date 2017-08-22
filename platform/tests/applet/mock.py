#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging
import os
import time
from threading import Thread

logger = logging.getLogger(__name__)


class PodLogManagerMock(object):
    def __init__(self, pod_name, service_id=None, root_id=None, leaf_full_path=None, namespace=None, app_mode=True):
        self._pod_name = pod_name
        self._log_root = "/logs"
        self._container_info = {}
        self._local_log_dirs = {}
        self._collectors = {}

        self._sid = service_id
        self._rid = root_id
        self._leaf = leaf_full_path
        self._am = app_mode
        self._ns = namespace

    def start_log_watcher(self, cname, cid):
        logger.info("Starting log collector for container %s (%s)", cname, cid)
        path = os.path.join(self._log_root, cid)
        if cid in self._collectors:
            logger.info("Log collector for container %s (%s) has already started", cname, cid)
            return
        self._container_info[cid] = cname
        try:
            collector = ContainerLogCollectorMock(
                cname=self._container_info[cid]
            )
            self._collectors[cid] = collector
            collector.start()
            self._local_log_dirs[cid] = path
            logger.info("Watching logs on %s", path)
        except Exception as e:
            logger.exception("%s", e)

    def stop_log_watcher(self, cid):
        """
        Stop a single log watcher
        :param cid:
        :return:
        """
        if not self._collectors.get(cid, None):
            return
        self._collectors[cid].terminate()
        self._collectors[cid].join()
        self._collectors.pop(cid, None)

    def terminate(self):
        for cid in list(self._collectors.keys()):
            self.stop_log_watcher(cid)
        logger.info("All log collectors terminated")

    def is_active(self):
        return len(self._collectors) > 0

    def get_containers(self):
        return self._collectors.keys()


class ContainerLogCollectorMock(Thread):
    def __init__(self, cname):
        super(ContainerLogCollectorMock, self).__init__()
        self.name = "log-collector-mock-{}".format(cname)
        self._terminate = False

    def terminate(self):
        self._terminate = True

    def is_busy(self):
        from random import randint
        if randint(1, 20) <= 1:
            return True
        return False

    def _ok_to_terminate(self):
        """
        Shutdown routine. If return true, main thread is safe to return cleanly
        :return:
        """
        if self._terminate and not self.is_busy():
            return True
        return False

    def run(self):
        while True:
            if self._ok_to_terminate():
                return
            time.sleep(0.1)

