# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import pytest
import subprocess
import logging
from ax.kubernetes.client import KubernetesApiClient
from ax.platform.ax_monitor import AXKubeMonitor
from ax.util.kube_poll import KubeObjPoll
from ax.platform_client.env import AXEnv

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)


@pytest.fixture(scope="module")
def kubectl():

    assert AXEnv().is_in_pod(), "Please run this test inside a pod"
    kubectl = KubernetesApiClient()
    yield kubectl


@pytest.fixture(scope="module")
def monitor(kubectl):
    logger.info("Starting AXKubeMonitor")

    monitor = AXKubeMonitor(kubectl=kubectl)
    monitor.reload_monitors(namespace="default")
    monitor.start()
    yield monitor

    logger.info("Tearing down AXKubeMonitor")
    monitor.stop(force=True)


@pytest.fixture(scope="module")
def kubepoll(kubectl):
    logger.info("Starting KubeObjPoll")

    kubepoll = KubeObjPoll(kubectl=kubectl)
    yield kubepoll

    logger.info("Tearing down KubeObjPoll")
    kubepoll.stop()
