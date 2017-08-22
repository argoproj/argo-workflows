#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

import json
import logging

import requests


logger = logging.getLogger(__name__)

class PrometheusClient(object):
    """Prometheus client."""

    def get_all_volume_free(self):
        try:
            base_url = PrometheusClient.get_prometheus_url()
            # TODO (shri): Make this url more specific about AWS volumes.
            response = requests.get(base_url + 'query?query=node_filesystem_free{mountpoint=~%22.*aws-ebs.*%22}')
            if response.text:
                data = json.loads(response.text)
                return data
        except Exception as e:
            logger.info("Failed while querying Prometheus: " + str(e))
        return

    @staticmethod
    def get_prometheus_url():
        """Get prometheus url.

        :return:
        """
        return "http://prometheus.axsys:9090/api/v1/"
