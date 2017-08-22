#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from ax.meta import AXClusterId


class AXClusterIdMock(AXClusterId):
    def __init__(self, name=None, aws_profile=None):
        super(AXClusterIdMock, self).__init__(name, aws_profile)

    # For testing purposes, as cluster name/id need to be
    # reloaded in different test scenarios
    def reinit(self, name=None, aws_profile=None):
        self._cluster_name = None
        self._cluster_id = None
        self._cluster_name_id = None
        self._bucket = None

        self._input_name = name
        self._aws_profile = aws_profile

