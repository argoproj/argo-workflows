#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os

from ax.util.singleton import Singleton
from future.utils import with_metaclass


"""
Customer id is the source of truth of s3 bucket that stores import cluster information.
Argo customer id should be provided as env AX_CUSTOMER_ID to all Argo software
that need to work with S3 buckets
"""

CUSTOMER_ID_ENV_NAME = "AX_CUSTOMER_ID"


class AXCustomerId(with_metaclass(Singleton, object)):
    def __init__(self):
        self._customer_id = os.getenv(CUSTOMER_ID_ENV_NAME)
        assert self._customer_id, "Cannot find customer id. Please provide env AX_CUSTOMER_ID"

    def get_customer_id(self):
        return self._customer_id
