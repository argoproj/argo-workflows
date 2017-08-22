#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
ApplicationManager client
"""

import logging
import requests
from retrying import retry


from .consts import CUR_HB_VERSION
from ax.platform.pod import Pod

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


class ApplicationManagerClient(object):
    def __init__(self):
        self._hb_url = "http://axam.{app}:8968/v1/heartbeats"

    @retry(
           wait_exponential_multiplier=1000,
           stop_max_attempt_number=3
    )
    def send_heart_beat(self, app_name, pod_name, dep_id, hb_type, timestamp, pod, fail_reason=None, fail_message=None):
        """
        Sends one heart beat to the am of the given pod/application
        :param app_name:
        :param pod_name:
        :param dep_id:
        :param hb_type:
        :param timestamp:
        :param pod: V1Pod
        :param fail_reason:
        :param fail_message:
        :return:
        :return:
        """
        status_to_send = self._form_status_to_send(pod_name, pod, fail_reason, fail_message)
        logger.info("Sending HeartBeat: NS(%s), PodName(%s), DepID(%s), Type(%s), TS(%s), Status: %s",
                    app_name, pod_name, dep_id, hb_type, timestamp, status_to_send)

        hb_data = {
            "key": dep_id,
            "date": timestamp,
            "data": {
                "version": CUR_HB_VERSION,
                "type": hb_type,
                "podStatus": status_to_send
            }
        }

        try:
            requests.post(
                url=self._hb_url.format(app=app_name),
                json=hb_data,
                timeout=5
            )
        except requests.ConnectionError as ce:
            if "Name does not resolve" in str(ce):
                logger.warning("Not sending heartbeat as AM address is not resolvable. Application might have been deleted already.")
                return
            else:
                raise ce

    @staticmethod
    def _form_status_to_send(pod_name, pod, fail_reason, fail_message):
        failure_info = None
        if fail_message or fail_reason:
            failure_info = {
                "reason": fail_reason,
                "message": fail_message
            }

        if pod:
            status_to_send = Pod.massage_pod_status(pod)
        else:
            status_to_send = {
                "name": pod_name,
                "phase": None,
                "start_time": None,
                "reason": None,
                "message": None,
                "containers": []
            }
        status_to_send["failure"] = failure_info
        return status_to_send


