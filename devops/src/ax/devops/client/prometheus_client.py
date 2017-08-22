#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

import logging

from ax.devops.settings import AxSettings
from ax.devops.client.base_client import BaseClient

logger = logging.getLogger(__name__)


class PrometheusClient(BaseClient):
    """Artifact Manager Client"""

    def __init__(self, host=None, port=None, timeout=60):
        """Initialize the artifact manager client

        :param host:
        :param port:
        :param version:
        :returns:
        """
        host = host or AxSettings.PROMETHEUS_HOSTNAME
        port = port or AxSettings.PROMETHEUS_PORT
        BaseClient.__init__(self, host=host, port=port, timeout=timeout)

    def delete_series(self, match_string):
        return self.retry_function(self.delete_query, '/api/v1/series?match[]={}'.format(match_string), payload=None)
