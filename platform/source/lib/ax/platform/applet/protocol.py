#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Protocol applet will use when dealing with user deployment
sends handshakes.

Protocol:
    User container sends Request-To-Start message:
        "V1##RTS##<PodName>##<AppName>"
    After processing RTS, applet sends back Okay-To-Start message:
        "OTS"
"""

import logging
import time

from retrying import retry
from ax.kubernetes.kubelet import KubeletClient
from ax.kubernetes.pod_status import PodStatus
from ax.kubernetes.swagger_client import V1Pod
from ax.kubernetes.client import retry_unless

from .appdb import ApplicationRecord
from .plm_pool import PodLogManagerPool
from .handshake import DefaultHandshakeProtocol
from .consts import CUR_RECORD_VERSION, HeartBeatType
from .amclient import ApplicationManagerClient

logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

REQUEST_TO_START = "RTS"
OK_TO_START = "OTS"


class DeploymentNannyProtocol(DefaultHandshakeProtocol):

    def generate_response_from_data(self, data):
        assert isinstance(data, str), "Unexpected data format {}".format(type(data))
        decoded = data.split("##")
        if len(decoded) != 4:
            logger.error("Cannot decode handshake message: %s", data)
            return ""

        version = decoded[0]
        msg = decoded[1]
        pod = decoded[2]
        app = decoded[3]

        if version != CUR_RECORD_VERSION:
            logger.warning("Cannot process msg with version %s", version)
            return ""

        if msg == REQUEST_TO_START:
            try:
                self.__process_rts(app, pod)
                return OK_TO_START
            except Exception as e:
                logger.exception("Failed to process RTS: %s", e)
                return ""
        else:
            logger.warning("Received invalid handshake: %s", data)
            return ""

    @retry(
        wait_fixed=2000,
        stop_max_attempt_number=5
    )
    def __process_rts(self, app_name, pod_name):
        logger.info("Processing RTS. Application: %s; Pod %s", app_name, pod_name)

        pod = self._get_pod_from_kubelet_with_retry(app_name, pod_name)
        timestamp = int(time.time())

        # this can throw exception as container info is not yet available to kubelet
        # retry wrapper will handle this
        ps = PodStatus(pod_status=pod)
        running = ps.list_current_containers()
        _, app_id, dep_id, dep_name = ps.get_app_meta()

        logger.info("Get running containers %s", running)

        # Update DB
        db = ApplicationRecord()
        db.update_application(
            app_name=app_name,
            app_id=app_id,
            deployment_name=dep_name,
            deployment_id=dep_id,
            pod_name=pod_name,
            cur_containers=running
        )
        db.close_connection()

        # Update pod log manager
        PodLogManagerPool().create_or_update_pod_log_manager(
            app_name=app_name,
            app_id=app_id,
            deployment_name=dep_name,
            deployment_id=dep_id,
            pod_name=pod_name,
            to_add=running
        )

        # Send heart beat
        ApplicationManagerClient().send_heart_beat(
            app_name=app_name,
            pod_name=pod_name,
            dep_id=dep_id,
            hb_type=HeartBeatType.BIRTH_CRY,
            timestamp=timestamp,
            pod=pod
        )

    @staticmethod
    @retry_unless()
    def _get_pod_from_kubelet_with_retry(app_name, pod_name):
        logger.info("Listing pod with namespace: %s, podname: %s", app_name, pod_name)
        rst = []
        for p in KubeletClient().list_namespaced_pods(namespace=app_name, name=pod_name):
            rst.append(p)
        # by the time handshake is received, kubelet must have this pod's information
        assert len(rst) == 1, "More than 1 pods received: {}".format(rst)
        assert isinstance(rst[0], V1Pod)
        assert rst[0].metadata, "Pod metadata missing"
        return rst[0]


