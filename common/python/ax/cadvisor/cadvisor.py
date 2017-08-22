#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

'''
cAdvisor CLI. Used by axstats temporarily before moving to Heapster
'''

import requests
import logging
import time

logger = logging.getLogger(__name__)
CHECK_LIVELINESS_INTERVAL = 5
CONNECTION_TIMEOUT = 5


class AXCadvisorClient(object):
    def __init__(self, ip):
        self._wait_interval = 60

        # Using Kubernetes default cadvisor port
        self._url_prefix = "http://{ip}:{port}/api/v2.0/".format(ip=ip, port=4194)
        self.wait_for_cadvisor_up()

    def wait_for_cadvisor_up(self):
        """
        Poll cadvisor endpoint till there is a response.
        Note it was calling /api/v2.0/version before, but this api in Kubernetes returns empty string
        :param url:
        :return:
        """
        ping = None
        while ping is None:
            ping = requests.get(self._url_prefix, timeout=CONNECTION_TIMEOUT)
            if ping is None:
                logger.debug("Unable to connect to cadvisor %s. Will sleep for %s sec",
                             self._url_prefix, CHECK_LIVELINESS_INTERVAL)
                time.sleep(CHECK_LIVELINESS_INTERVAL)
        logger.info("cAdvisor client is up for endpoint %s", self._url_prefix)

    def get_machine_info(self):
        url = self._url_prefix + "machine"
        return self._get_response(url)

    def get_spec_info(self):
        url = self._url_prefix + "spec"
        data = {
            "recursive": "true"
        }
        return self._get_response(url, data)

    def get_events(self, event_start):
        url = self._url_prefix + "events"
        data = {
            "all_events": "true",
            "subcontainers": "true",
            "start_time": event_start
        }
        return self._get_response(url, data)

    def get_docker_stats(self):
        url = self._url_prefix + "stats"
        data = {
            "recursive": "true",
            "count": str(self._wait_interval)
        }
        return self._get_response(url, data)

    @staticmethod
    def _get_response(url, params=None):
        out = None
        try:
            response = requests.get(url=url, params=params, timeout=CONNECTION_TIMEOUT)
            if response.status_code == requests.codes.ok:
                out = response.json()
        except requests.exceptions.RequestException as e:
            logger.error('Unexpected exception occurred during request: %s', e)
        return out
