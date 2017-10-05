#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Library for accessing AWS S3.

This is organized at bucket level.
"""

import os
import logging
import json
import time

import boto3
import requests
from ax.cloud import Cloud
from botocore.client import Config
from botocore.exceptions import ClientError
from retrying import retry

from .util import default_aws_retry

logger = logging.getLogger(__name__)

"""
TODO: need to handle exceptions. Typical exception looks like this.
{
    'response': {
        'ResponseMetadata': {
            'HTTPStatusCode': 400,
            'HostId': 'kZJI+LI/VP4cP8NSNgyh11x/71qmhL+ZkwTo8pNjrMVSlFakow9t+pqjp2EANMwq',
            'RequestId': '64B1BAE2E12998DE'
        },
        'Error': {
            'Message': 'The specified bucket is not valid.',
            'Code': 'InvalidBucketName',
            'BucketName': 'ax_lcj_test'
        }
    }
}
"""

AWS_S3_BATCH_SIZE = 1000

# In Argo use case, bucket clean means there is NO cluster using bucket. In this situation,
# there should be at most objects with prefix `BUCKET_CLEAN_KEYWORD` in the bucket.
# TODO: I don't like this constant name, any better suggestions?
BUCKET_CLEAN_KEYWORD = "kubernetes-staging"

CORS_CONFIG_KEY = "ax-bucket-attributes/cors-config"

# S3 objects in us-east-1 uses different endpoint as other regions
# see http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html
S3_ENDPOINT_COMMON = "https://s3-{region}.amazonaws.com/{bucket_name}"
S3_ENDPOINT_VIRGINIA = "https://s3.amazonaws.com/{bucket_name}"


# Getting bucket region is tricky. The method that can work reliably if to start boto3 session
# with a random region, and call head_bucket but enforce signature V4. In addition, the random
# region should be inside the same aws partition as the actual region of the bucket.
#
# If this code runs on an aws instance then perfect, we can get this initial region from their
# metadata server, assuming we don't access buckets in different partition. But if this code
# is invoked in a normal user environment, i.e. laptop, we need to guess partitions.
#
# The following list provides region in each partition we can use to get bucket's actual
# region.
PARTITION_DEFAULT_REGIONS = ["us-east-1", "us-gov-west-1", "cn-north-1"]


def head_bucket_retry(exception):
    # Retrying on head bueckt is tricky, as HTTP code can change
    # from time to time after the bucket is created:
    #   - HTTP/1.1 307 Temporary Redirect
    #   - HTTP/1.1 403 Forbidden
    # For not we only retry on network errors
    return isinstance(exception, requests.exceptions.Timeout) or \
           isinstance(exception, requests.exceptions.ConnectionError)


class AXS3Bucket(object):
    """
    Wrapper around S3 API.

    TODO: There is problem with this module.
    It does extra list_bucket call, which can be problematic with IAM roles.
    """
    def __init__(self, bucket_name, aws_profile=None, region=None):
        """
        Initialize a bucket object.

        :param bucket_name: Bucket name.
        :return:
        """
        self._name = bucket_name
        self._aws_profile = aws_profile
        self._region = region if region else self._get_bucket_region_from_aws()
        logger.info("Using region %s for bucket %s", self._region, self._name)

        session = boto3.Session(profile_name=aws_profile, region_name=self._region)
        self._s3 = session.resource("s3", aws_access_key_id=os.environ.get("ARGO_S3_ACCESS_KEY_ID", None),
                aws_secret_access_key=os.environ.get("ARGO_S3_ACCESS_KEY_SECRET", None),
                endpoint_url=os.environ.get("ARGO_S3_ENDPOINT", None))
        self._s3_client = session.client("s3", aws_access_key_id=os.environ.get("ARGO_S3_ACCESS_KEY_ID", None),
                aws_secret_access_key=os.environ.get("ARGO_S3_ACCESS_KEY_SECRET", None),
                endpoint_url=os.environ.get("ARGO_S3_ENDPOINT", None))
        self._bucket = self._s3.Bucket(self._name)
        self._policy = self._s3.BucketPolicy(self._name)

    def __repr__(self):
        return "{}".format(self._name)

    def get_bucket_name(self):
        return self._name

    @retry(
        retry_on_exception=head_bucket_retry,
        wait_exponential_multiplier=1000,
        stop_max_attempt_number=3
    )
    def _get_bucket_region_from_aws(self):

        def _do_get_region(start_region):
            s3 = boto3.Session(
                profile_name=self._aws_profile,
                region_name=start_region
            ).client("s3", aws_access_key_id=os.environ.get("ARGO_S3_ACCESS_KEY_ID", None),
                    aws_secret_access_key=os.environ.get("ARGO_S3_ACCESS_KEY_SECRET", None),
                    endpoint_url=os.environ.get("ARGO_S3_ENDPOINT", None),
                    config=Config(signature_version='s3v4'))
            logger.debug("Finding region for bucket %s from with initial region %s", self._name, start_region)
            try:
                response = s3.head_bucket(Bucket=self._name)
                logger.debug("Head_bucket returned OK %s", response)
            except ClientError as e:
                response = getattr(e, "response", {})
                logger.debug("Head_bucket returned error %s, inspecting headers", response)
                return None
            headers = response.get("ResponseMetadata", {}).get("HTTPHeaders", {})
            region = headers.get("x-amz-bucket-region", headers.get("x-amz-region", None))
            logger.debug("Found region %s from head_bucket for %s, headers %s", region, self._name, headers)
            return region

        if Cloud().own_cloud() == Cloud.CLOUD_AWS:
            # When running on AWS instance, we query metadata server for initial region to get bucket region
            return _do_get_region(Cloud().meta_data().get_region())
        else:
            # Assume we don't have AWS metadata server access
            for r in PARTITION_DEFAULT_REGIONS:
                logger.debug("Trying partition default region %s to get bucket region.", r)
                bucket_region = _do_get_region(r)
                if bucket_region:
                    return bucket_region
        return None

    def region(self):
        return self._region

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def create(self, **kwargs):
        """
        Create a new bucket in S3.

        :param region: AWS region for bucket.
        :param kwargs: All other creation args for S3.
                       http://boto3.readthedocs.org/en/latest/reference/services/s3.html#S3.Bucket.create
        :return: True or False
        """
        if self.exists():
            logger.info("Bucket %s already exist, don't create.", self._name)
            return True

        assert not kwargs.get("region", None), "Region should not be specified in create method"
        if self._region != "us-east-1":
            kwargs.update({
                "CreateBucketConfiguration": {
                    "LocationConstraint": self._region}})

        logger.info("Creating bucket %s in %s, with arguments %s", self._name, self._region, kwargs)
        try:
            self._bucket.create(**kwargs)
        except ClientError as ce:
            if "BucketAlreadyOwnedByYou" not in str(ce):
                raise ce

        # Head bucket has glitches right after bucket is created, so we return after 3 consecutive exists()
        logger.info("Waiting for bucket %s to stably exist ...", self._name)
        exists_count = 0
        while True:
            if self.exists():
                exists_count += 1
            else:
                exists_count = 0
            if exists_count == 3:
                break
            time.sleep(2)
        logger.info("Bucket %s created", self._name)
        return True

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def delete(self, force=False):
        """
        Delete a bucket and optionally all object inside it.
        :param force: Auto delete all objects inside.
        :return:
        """
        if not self.exists():
            logger.debug("Bucket %s doesn't exist, don't delete.", self._name)
            return True
        logger.info("Deleting bucket %s. Force: %s", self._name, force)
        try:
            if force:
                for key in self._bucket.objects.all():
                    key.delete()
            self._bucket.delete()
        except ClientError as ce:
            if "NoSuchBucket" not in str(ce):
                raise ce

        # Head bucket has glitches right after bucket is deleted, so we return after 3 consecutive not exists()
        logger.info("Waiting for bucket %s to stably disappear ...", self._name)
        not_exists_count = 0
        while True:
            if self.exists():
                not_exists_count = 0
            else:
                not_exists_count += 1
            if not_exists_count == 3:
                break
            time.sleep(2)
        logger.info("Bucket %s deleted.", self._name)
        return True

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def put_policy(self, policy):
        if not self.exists():
            logger.error("Failed to upload policy as bucket %s does not exist", self._name)
            return False

        try:
            self._policy.put(Policy=policy)
            logger.info("Successfully added policy for bucket %s", self._name)
            return True
        except ClientError as ce:
            if "MalformedPolicy" in str(ce):
                logger.error("Failed to add policy for bucket %s. Error: %s", self._name, ce)
                return False
            else:
                raise ce

    def get_policy(self):
        try:
            return self._policy.policy
        except Exception as e:
            logger.debug("Bucket policy does not exist. Msg: %s", e)
            return False

    def exists(self):
        """
        Whether a bucket exists.
        :return: True or False
        """
        return self._exists()

    def empty(self):
        return self._empty()

    def clean(self):
        return self._clean()

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def put_cors(self, cors, force=False):
        # TODO: Possible race when two cluster install/upgrade run at same time.
        version = 0
        existing = self.get_object(CORS_CONFIG_KEY)
        if existing is not None:
            version = json.loads(existing)["version"]
        if version < cors["version"] or force:
            logger.info("Adding CORS config %s to %s, old version %s.", cors, self._name, version)
            self._s3_client.put_bucket_cors(Bucket=self._name, CORSConfiguration=cors["config"])
            assert self.put_object(CORS_CONFIG_KEY, json.dumps(cors))
        else:
            logger.info("CORS config version %s already exists for %s.", version, self._name)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def delete_cors(self):
        # Both delete_object and delete_bucket_cors() would return silently if not found.
        self.delete_object(CORS_CONFIG_KEY)
        self._s3_client.delete_bucket_cors(Bucket=self._name)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_cors(self):
        try:
            return self._s3_client.get_bucket_cors(Bucket=self._name)
        except ClientError as e:
            if "NoSuchCORSConfiguration" in str(e):
                logger.info("CORS not found for %s.", self._name)
            else:
                raise

    def list_objects(self, keyword="", list_all=False):
        """
        List all objects inside a bucket.
        :return: List of object keys in string
        """
        if not list_all:
            assert keyword is not None, "No keyword provided when not listing all objects"
        if not self.exists():
            logger.info("Bucket does not exist")
            return []
        keys = []
        logger.info("Retrieving objects with keyword %s ...", keyword)
        for key in self._bucket.objects.all():
            if list_all:
                keys.append(key.key)
            else:
                if keyword in key.key:
                    keys.append(key.key)

        logger.info("Retrieving objects with keyword %s ... DONE", keyword)
        return keys

    def list_objects_by_prefix(self, prefix):
        """
        List objects in the bucket by the given prefix.
        :param prefix: Prefix string
        :return: iterable of s3.ObjectSummary
        """
        assert prefix is not None, "Prefix cannot be none."
        return self._bucket.objects.filter(Prefix=prefix)

    def get_object(self, key, **kwargs):
        """
        Get object data specified by key.
        :param key: Key string
        :param kwargs: Other arguments for S3 API.
                       http://boto3.readthedocs.org/en/latest/reference/services/s3.html#S3.Object.get
        :return: actual object or None
        """
        try:
            return self._s3.Object(self._name, key).get(**kwargs)["Body"].read().decode("utf-8")
        except Exception as e:
            if not self.exists():
                return None
            if "NoSuchKey" not in str(e):
                raise
        return None

    def get_object_url_from_key(self, key):
        if self._region == "us-east-1":
            url_base = S3_ENDPOINT_VIRGINIA.format(bucket_name=self._name)
        else:
            url_base = S3_ENDPOINT_COMMON.format(region=self._region, bucket_name=self._name)
        url = "{base}/{obj_key}".format(base=url_base, obj_key=key)
        logger.debug("URL for object %s in bucket %s is %s", key, self._name, url)
        return url

    def put_object(self, key, data, **kwargs):
        """
        Put object into S3, overwrite by default now.
        :param key: key string
        :param data: data blob
        :param kwargs: Other S3 arguments.
                       http://boto3.readthedocs.org/en/latest/reference/services/s3.html#S3.Object.put
        :return:
        """
        assert "Key" not in kwargs, "Can't pass Key in kwargs"
        assert "Body" not in kwargs, "Can't pass Body in kwargs"
        if not self.exists():
            return True

        try:
            self._s3.Object(self._name, key).put(Body=data, **kwargs)
            return True
        except Exception as e:
            logger.warning("Failed to put s3 object to bucket %s, key: %s, data: %s, with error %s", self._bucket, key, data, e)
            return False

    def put_file(self, local_file_name, s3_key, **kwargs):
        """
        Similar to object but this uploads a file
        :param local_file_name:
        :param s3_key:
        :param kwargs:
        :return:
        """
        if not self.exists():
            return False
        self._bucket.upload_file(Filename=local_file_name, Key=s3_key, **kwargs)
        return True

    def copy_object(self, source_key, dest_key, **kwargs):
        """
        Copy object within bucket
        :param source_key:
        :param dest_key:
        :return:
        """
        dest_obj = self._s3.Object(self._name, dest_key)
        copy_source = {
            "Bucket": self._name,
            "Key": source_key
        }
        dest_obj.copy_from(CopySource=copy_source, **kwargs)

    def download_file(self, key, file_name, **kwargs):
        self._s3_client.download_file(self._name, key, file_name, **kwargs)

    def delete_object(self, key, **kwargs):
        """
        Delete object from S3. As we are not using versioning, this is a simple delete of object
        :param key:
        :param kwargs:
        :return:
        """
        if not self.exists():
            return True
        try:
            self._s3.Object(self._name, key).delete(**kwargs)
            return True
        except Exception:
            logger.exception("Failed to delete s3 object %s, %s", self._bucket, key)
            return False

    def delete_all(self, obj_prefix="", use_prefix=True, exempt=None):
        """
        Delete all s3 objects with given prefix
        :param obj_prefix:
        :param use_prefix:
        :param exempt: a list of objects (exact key) that should not be deleted
        :return:
        """
        if not self.exists():
            logger.warning("Trying to delete all object in bucket %s with prefis \"%s\", and exemption %s, but bucket does not exist", self._name, obj_prefix, exempt)
            return

        if use_prefix:
            assert obj_prefix, "No object prefix is provided when using prefix"

        if exempt:
            assert isinstance(exempt, list), "exempt objects should be a list of object keys"

        if not use_prefix:
            logger.warning("No object prefix provided, deleting all objects in bucket %s. Exempted: %s ...",
                           self._name, exempt)
        else:
            logger.info("Deleting all objects in bucket %s with prefix \"%s\". Exempted: %s ...",
                        self._name, obj_prefix, exempt)

        total_deleted = 0
        while True:
            to_delete = []
            try:
                obj_batch = self._s3_client.list_objects(
                    Bucket=self._name,
                    MaxKeys=AWS_S3_BATCH_SIZE,
                    Prefix=obj_prefix
                )
            except ClientError as ce:
                logger.error("Failed to list object with prefix %s. Error: %s", obj_prefix, ce)
                raise ce
            if exempt:
                if "Contents" not in obj_batch:
                    logger.info("All objects with prefix %s in bucket %s have been deleted. Exempted objects don't exist",
                                obj_prefix, self._name)
                    return
                # Use "<=" in case any of the exempted object does not exist
                if len(obj_batch["Contents"]) <= len(exempt):
                    logger.info("All objects with prefix %s in bucket %s have been deleted. Exempted: %s",
                                obj_prefix, self._name, exempt)
                    return
            else:
                if "Contents" not in obj_batch:
                    logger.info("All objects with prefix %s in bucket %s have been deleted", obj_prefix, self._name)
                    return
            for obj in obj_batch["Contents"]:
                if exempt and obj["Key"] in exempt:
                    continue
                info = {
                    "Key": obj["Key"]
                }
                to_delete.append(info)

            try:
                deleted_object_batch = self._s3_client.delete_objects(
                    Bucket=self._name,
                    Delete={
                        "Objects": to_delete
                    }
                )
            except ClientError as ce:
                logger.error("Failed to delete object batch %s. Error: %s", to_delete, ce)
                raise ce

            # If we send something to delete, there should be "Deleted" key in response, provide
            # that there is no exception thrown
            assert "Deleted" in deleted_object_batch, "Does not have \"Deleted\" key in response. {}".format(deleted_object_batch)

            if "Errors" in deleted_object_batch:
                logger.warning("Failed to delete objects %s", deleted_object_batch["Errors"])

            deleted_num = len(deleted_object_batch["Deleted"])
            total_deleted += deleted_num
            logger.info("Deleted %s objects, %s in total", deleted_num, total_deleted)

    def generate_signed_url(self, key):
        """
        Generate a pre signed URL for a object.

        :param key: Object key in string
        :return: Signed URL in string
        """
        return self._s3_client.generate_presigned_url(ClientMethod="get_object",
                                                      Params={"Bucket": self._name, 'Key': key})

    def _clean(self):
        # for support bucket and upgrade bucket, bucket empty is same as bucket clean
        # for cluster bucket, we should expect only "BUCKET_CLEAN_KEYWORD/..." in all remaining objects
        try:
            for key in self._bucket.objects.all():
                if BUCKET_CLEAN_KEYWORD not in key.key:
                    return False
        except ClientError as ce:
            # It happens right after bucket deletion: head_bucket would show some glitch
            # this is more reliable
            if "NoSuchBucket" in str(ce):
                return True
            raise ce
        return True

    @retry(
        wait_exponential_multiplier=1000,
        stop_max_attempt_number=3
    )
    def _exists(self):
        try:
            self._s3_client.head_bucket(Bucket=self._name)
            return True
        except ClientError as ce:
            if "Not Found" in str(ce):
                return False
            else:
                raise ce

    def _empty(self):
        try:
            for _ in self._bucket.objects.all():
                return False
        except ClientError as ce:
            if "NoSuchBucket" in str(ce):
                return True
            raise ce
        return True
