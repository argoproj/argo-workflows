# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import pytest

from ax.cloud.aws import AXS3Bucket
from ax.meta import AXCustomerId
from .testdata import *


@pytest.fixture(scope="module")
def cluster_bucket():
    bucket = AXS3Bucket(
        bucket_name="applatix-cluster-{}-0".format(AXCustomerId().get_customer_id()),
        region=TEST_AWS_REGION,
        aws_profile=TEST_AWS_PROFILE
    )
    bucket.create()
    yield bucket
    bucket.delete(force=True)
