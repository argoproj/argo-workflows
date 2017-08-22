#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging
import os
import pytest
from .testdata import *
from .mock import AXClusterIdMock

# Customer id is source of truth so we just set it globally
os.environ["AX_CUSTOMER_ID"] = TEST_CUSTOMER_ID
os.environ["AX_TARGET_CLOUD"] = "aws"

logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S")
logging.getLogger("botocore").setLevel(logging.WARNING)
logging.getLogger("boto3").setLevel(logging.WARNING)
logging.getLogger("ax").setLevel(logging.DEBUG)
logging.getLogger("requests").setLevel(logging.WARNING)


# This only happens when a user first installs a cluster
# we should not be able to upload / get cluster name id
def test_cluster_name_id_without_bucket():
    idobj = AXClusterIdMock(name=TEST_CLUSTER_NAME, aws_profile=TEST_AWS_PROFILE)

    # Simply passing cluster name without creation should not be able to operate
    # all of those operations
    with pytest.raises(Exception):
        idobj.get_cluster_name()
    with pytest.raises(Exception):
        idobj.get_cluster_id()
    with pytest.raises(Exception):
        idobj.get_cluster_name_id()

    # Without a bucket, AXClusterId object should be able to parse "name-id" format
    # after create, but should not be able to upload cluster id
    idobj.reinit(name=TEST_CLUSTER_NAME_ID_AWS, aws_profile=TEST_AWS_PROFILE)
    idobj.create_cluster_name_id()
    assert idobj.get_cluster_name() == TEST_CLUSTER_NAME
    assert idobj.get_cluster_id() == TEST_CLUSTER_ID_AWS
    assert idobj.get_cluster_name_id() == TEST_CLUSTER_NAME_ID_AWS
    with pytest.raises(Exception):
        idobj.upload_cluster_name_id()


# After a bucket is created: install/uninstall/pause/restart/upgrade case
# Noticeably, during creation, we ONLY upload cluster name-id record after
# buckets are ensured
def test_cluster_name_id_with_bucket(cluster_bucket):
    idobj = AXClusterIdMock()
    idobj.reinit(name=TEST_CLUSTER_NAME_ID_AWS, aws_profile=TEST_AWS_PROFILE)

    # When passing "name-id" format, we are able to creat/upload, get name/id/name-id
    idobj.create_cluster_name_id()
    idobj.upload_cluster_name_id()
    assert idobj.get_cluster_name() == TEST_CLUSTER_NAME
    assert idobj.get_cluster_id() == TEST_CLUSTER_ID_AWS
    assert idobj.get_cluster_name_id() == TEST_CLUSTER_NAME_ID_AWS

    # After name id is uploaded, when we just initialize the class with cluster name
    # we should still be able to get all these information
    idobj.reinit(name=TEST_CLUSTER_NAME, aws_profile=TEST_AWS_PROFILE)
    assert idobj.get_cluster_name() == TEST_CLUSTER_NAME
    assert idobj.get_cluster_id() == TEST_CLUSTER_ID_AWS
    assert idobj.get_cluster_name_id() == TEST_CLUSTER_NAME_ID_AWS


# All applatix services should be provided with Applatix cluster name id env,
# by the provisioner, i.e. axinstaller, and applatix service do not need to
# initialize the class with anything to get cluster name / id / name-id
def test_cluster_name_id_with_env():
    os.environ["AX_CLUSTER_NAME_ID"] = TEST_CLUSTER_NAME_ID_AWS
    idobj = AXClusterIdMock()
    idobj.reinit()

    assert idobj.get_cluster_name() == TEST_CLUSTER_NAME
    assert idobj.get_cluster_id() == TEST_CLUSTER_ID_AWS
    assert idobj.get_cluster_name_id() == TEST_CLUSTER_NAME_ID_AWS



