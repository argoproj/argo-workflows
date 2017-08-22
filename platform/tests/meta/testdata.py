# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import random
import string
import uuid

TEST_AWS_REGION = "us-west-2"
TEST_AWS_PROFILE = "dev"

TEST_CUSTOMER_ID = str(uuid.uuid4())
TEST_CLUSTER_NAME = "cluster-" + "".join(random.SystemRandom().choice(
    string.ascii_uppercase + string.digits + string.ascii_lowercase
) for _ in range(7))
TEST_CLUSTER_ID_AWS = str(uuid.uuid1())
TEST_CLUSTER_NAME_ID_AWS = "{}-{}".format(TEST_CLUSTER_NAME, TEST_CLUSTER_ID_AWS)
