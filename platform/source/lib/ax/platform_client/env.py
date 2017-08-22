#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module library to identify python running environment.
"""

import os

from future.utils import with_metaclass

from ax.util.singleton import Singleton


class AXEnv(with_metaclass(Singleton, object)):

    in_container = None
    in_pod = None
    kube_host = None

    def __init__(self):
        self.in_container = os.path.isfile('/.dockerenv')
        self.in_pod = bool(os.getenv('KUBERNETES_SERVICE_HOST'))
        self.kube_host = os.path.isfile('/etc/kubernetes/bootstrap')
        self._prepare_path()

    def is_in_pod(self):
        return self.in_pod

    def on_kube_host(self):
        return self.kube_host

    def _is_in_container(self):
        return self.in_container

    def _prepare_path(self):
        if self._is_in_container() or self.is_in_pod():
            # Run in container environment.
            self.axtool_path = "/ax/bin/axtool"
            self.axservice_config_path = "/ax/config/service"
            self.axservice_mvc_config_path = "/ax/config/service/mvc"
        else:
            # Run in source code.
            self.platform_path = os.path.realpath(os.path.join(os.path.dirname(__file__), "../../../.."))
            self.tools_path = os.path.join(self.platform_path, "source", "tools")
            self.axtool_path = os.path.join(self.tools_path, "axtool.py")
            self.axservice_config_path = os.path.join(self.platform_path, "config", "service")
            self.axservice_mvc_config_path = os.path.join(self.platform_path, "config", "service", "mvc")
            self.cloud_config_path = os.path.join(self.platform_path, "config", "cloud")
