#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
The Broker object takes a cloud provider name as input and returns the
appropriate object on which subsequent methods can be called.
"""

from ax.cloud import Cloud

from ax.platform.minion_manager.cloud_provider.aws.aws_minion_manager import AWSMinionManager


class Broker(object):
    """ Create and return cloud provider specific objects """

    @staticmethod
    def get_impl_object(provider, scaling_groups, region, **kwargs):
        """
        Given a cloud provider name, return the cloud provider specific
        implementation.
        """
        if provider.lower() == Cloud.CLOUD_AWS:
            return AWSMinionManager(scaling_groups, region, **kwargs)

        raise NotImplementedError
