#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

import logging
import json

from retrying import retry

from ax.devops.axrequests.axrequests import AxRequests

logger = logging.getLogger(__name__)


class BaseClient(object):
    """Base Client for all relevant components in DevOps."""

    def __init__(self, host=None, port=None, version=None, url=None, timeout=None):
        """Initialize the base client

        :param host:
        :param port:
        :param version:
        :param url:
        :param timeout:
        :returns:
        """
        self.ax_request = AxRequests(host=host, port=port, version=version, protocol='http', url=url, timeout=timeout)

    @staticmethod
    def retry_function(f, *args, **kwargs):
        """Retry a function call

        :param f:
        :param args:
        :param kwargs:
        :returns:
        """
        wait_exponential_multiplier = kwargs.pop('wait_exponential_multiplier', 1000)
        wait_exponential_max = kwargs.pop('wait_exponential_max', 60000)
        stop_max_attempt_number = kwargs.pop('max_retry', 20)
        retry_on_exception = kwargs.pop('retry_on_exception', None)

        @retry(wait_exponential_multiplier=wait_exponential_multiplier, wait_exponential_max=wait_exponential_max,
               stop_max_attempt_number=stop_max_attempt_number, retry_on_exception=retry_on_exception)
        def _f():
            return f(*args, **kwargs)

        return _f()

    def get_query(self, path, payload=None, **kwargs):
        """Get query"""
        return self.ax_request.get(path=path, params=payload, data=None, **kwargs)

    def create_query(self, path, payload, **kwargs):
        """Create query"""
        return self.ax_request.post(path=path, params=None, data=json.dumps(payload), **kwargs)

    def update_query(self, path, payload, **kwargs):
        """Update query"""
        return self.ax_request.put(path=path, params=None, data=json.dumps(payload), **kwargs)

    def delete_query(self, path, payload=None, **kwargs):
        """Delete query"""
        return self.ax_request.delete(path=path, params=payload, data=None, **kwargs)
