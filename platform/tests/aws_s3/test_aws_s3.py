#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import json
import pytest
import logging
import requests
import time

import ax.cloud
from .mock import CloudAWSMock
ax.cloud.Cloud = CloudAWSMock

from ax.cloud.aws import AXS3Bucket, BUCKET_CLEAN_KEYWORD
from .testdata import *

logger = logging.getLogger(__name__)

logging.basicConfig(
    format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S"
)
logging.getLogger("botocore").setLevel(logging.WARNING)
logging.getLogger("boto3").setLevel(logging.WARNING)
logging.getLogger("ax").setLevel(logging.DEBUG)
logging.getLogger("requests").setLevel(logging.WARNING)


def test_s3_get_region():
    for region in AWS_REGIONS:
        bucket_name = TEST_BUCKET_NAME_TEMPLATE.format(region=region)
        logger.info("Testing GetRegion for bucket %s", bucket_name)

        bucket = AXS3Bucket(bucket_name=bucket_name, aws_profile=TEST_AWS_PROFILE, region=region)
        bucket.create()

        assert bucket._get_bucket_region_from_aws() == region

        # Need to cool down a bit as bucket creation / deletion is very heavy weighted
        bucket.delete()
        time.sleep(5)


def test_put_policy_no_bucket():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE, region=TEST_AWS_REGION)
    assert not bucket.put_policy(policy="")


def test_delete_nonexist_bucket():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE, region=TEST_AWS_REGION)
    assert bucket.delete(force=True)


def test_bucket_create():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE, region=TEST_AWS_REGION)
    assert bucket.create()
    assert bucket.exists()
    assert bucket.empty()
    assert bucket.clean()

    # Recreate should return True
    assert bucket.create()


# From this test on, bucket is already created, so we don't need to pass `region` to the class
# which is what most of our use cases are
def test_put_policy_invalid_format():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    assert not bucket.put_policy(policy=TEST_INVALID_POLICY_FORMAT)


def test_put_policy_invalid_content():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    assert not bucket.put_policy(policy=TEST_INVALID_POLICY_CONTENT)


def test_cors_operation():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)

    # Do the operations twice to ensure idempotency
    bucket.put_cors(TEST_CORS_CONFIG)
    bucket.put_cors(TEST_CORS_CONFIG)
    bucket.delete_cors()
    bucket.delete_cors()


def test_single_object_operations():
    file_name = "test_file"
    file_content = "test_content"
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    assert bucket.put_object(file_name, file_content)

    assert not bucket.clean()
    assert not bucket.empty()

    file_content_s3 = bucket.get_object(file_name)
    if isinstance(file_content_s3, bytes):
        assert file_content == file_content_s3.decode("utf-8")
    else:
        assert file_content == file_content_s3

    file_name_cpy = "test_file_copy"
    bucket.copy_object(file_name, file_name_cpy)

    file_content_s3_cpy = bucket.get_object(file_name_cpy)
    assert file_content_s3 == file_content_s3_cpy

    assert bucket.delete_object(file_name)
    assert bucket.get_object(file_name) is None

    assert bucket.delete_object(file_name_cpy)
    assert bucket.get_object(file_name_cpy) is None

    assert bucket.clean()
    assert bucket.empty()


def test_generate_object_url():
    file_name = "test_file_url"
    file_content = "test_content_url"
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.put_object(file_name, file_content, ACL="public-read")

    url = bucket.get_object_url_from_key(key=file_name)
    data = requests.get(url, timeout=5).text
    assert data == file_content

    assert bucket.delete_object(file_name)


def test_bucket_clean():
    file_name = BUCKET_CLEAN_KEYWORD + "{:05d}".format(random.randint(1, 99999))
    file_content = "test_content"
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    assert bucket.put_object(file_name, file_content)
    assert bucket.clean()

    if not bucket.delete_object(file_name):
        pytest.fail("Failed to delete object {}".format(file_name))


def test_list_objects():
    file_name_prefix = "test_file-"
    file_content = "test_content"
    file_name_set = set()
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    for i in range(50):
        file_name = file_name_prefix + "{:03d}".format(i)
        file_name_set.add(file_name)
        assert bucket.put_object(key=file_name, data=file_content)
    file_name_set_s3 = set(bucket.list_objects(keyword=file_name_prefix))
    assert file_name_set_s3 == file_name_set


def test_delete_all_without_prefix():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    file_name_prefix = "test_file-"
    file_content = "test_content"
    for i in range(50):
        file_name = file_name_prefix + "{:03d}".format(i)
        assert bucket.put_object(key=file_name, data=file_content)

    bucket.delete_all(use_prefix=False)

    assert bucket.clean()
    assert bucket.empty()


def test_delete_all_with_prefix_and_exemption():
    file_name_prefix = "test_file-"
    file_content = "test_content"
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    for i in range(50):
        file_name = file_name_prefix + "{:03d}".format(i)
        assert bucket.put_object(key=file_name, data=file_content)

    bucket.put_object(key="special_file", data=file_content)
    bucket.delete_all(obj_prefix=file_name_prefix, exempt=["test_file-015"])

    remaining_file_s3 = set(bucket.list_objects(list_all=True))
    remaining_file = {"special_file", "test_file-015"}

    assert remaining_file_s3 == remaining_file

    bucket.delete_all(use_prefix=False)
    assert bucket.clean()
    assert bucket.empty()


def test_bucket_delete():
    bucket = AXS3Bucket(bucket_name=TEST_BUCKET_NAME, aws_profile=TEST_AWS_PROFILE)
    assert bucket.create()
    assert bucket.delete(force=True)

    assert bucket.clean()
    assert bucket.empty()

    # Re-delete should return True
    assert bucket.delete(force=True)

