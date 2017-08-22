#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

import logging

from ax.devops.settings import AxSettings
from ax.devops.client.base_client import BaseClient

logger = logging.getLogger(__name__)


class AxAMMClient(BaseClient):
    """AX Application Manager Client"""

    def __init__(self, host=None, port=None, version=None, timeout=60):
        """Initialize the application manager client

        :param host:
        :param port:
        :param version:
        :returns:
        """
        host = host or AxSettings.AXAMM_HOSTNAME
        port = port or AxSettings.AXAMM_PORT
        version = version or AxSettings.AXAMM_VERSION
        BaseClient.__init__(self, host=host, port=port, version=version, timeout=timeout)

    def query_deployments(self, conditions, **kwargs):
        """List deployments with conditions

        :param conditions:
        :param kwargs:
        :return:
        """
        return self.retry_function(self.get_query, '/deployments', conditions, **kwargs)
