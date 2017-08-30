#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import random
import logging

from ax.cloud import Cloud
from ax.aws.meta_data import AWSMetaData
from .testdata import *

logger = logging.getLogger(__name__)


class AWSMetaDataMock(AWSMetaData):
    def __init__(self):
        pass

    def get_region(self):
        return random.choice(AWS_REGIONS)

    def get_security_groups(self):
        raise NotImplementedError()

    def get_zone(self):
        raise NotImplementedError()

    def get_public_ip(self):
        raise NotImplementedError()

    def get_instance_id(self):
        raise NotImplementedError()

    def get_instance_type(self):
        raise NotImplementedError()

    def get_private_ip(self):
        raise NotImplementedError()

    def get_user_data(self, attr=None, plain_text=False):
        raise NotImplementedError()


class CloudAWSMock(Cloud):
    def __init__(self):
        super(CloudAWSMock, self).__init__(target_cloud=self.CLOUD_AWS)
        self._own_cloud = self.AX_CLOUD_AWS

    def meta_data(self):
        return AWSMetaDataMock()

