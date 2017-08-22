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


class AxArtifactManagerClient(BaseClient):
    """Artifact Manager Client"""

    def __init__(self, host=None, port=None, version=None, timeout=60):
        """Initialize the artifact manager client

        :param host:
        :param port:
        :param version:
        :returns:
        """
        host = host or AxSettings.AXARTIFACTMANAGER_HOSTNAME
        port = port or AxSettings.AXARTIFACTMANAGER_PORT
        version = version or AxSettings.AXARTIFACTMANAGER_VERSION
        BaseClient.__init__(self, host=host, port=port, version=version, timeout=timeout)

    def create_artifact(self, artifact, **kwargs):
        """Create an artifact

        :param artifact:
        :return:
        """
        return self.retry_function(self.create_query, '/artifacts', artifact, **kwargs)

    def query_artifacts(self, conditions, **kwargs):
        """Create an artifact

        :param conditions:
        :return:
        """
        return self.retry_function(self.get_query, '/artifacts', conditions, **kwargs)
