#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Library for accessing Google Cloud Storage.

This is organized at bucket level.
"""

import json
import logging

from google.cloud import storage
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import NotFound
from retrying import retry


logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

# In Argo use case, bucket clean means there is NO cluster using bucket. In this situation,
# there should be at most objects with prefix `BUCKET_CLEAN_KEYWORD` in the bucket.
# TODO: I don't like this constant name, any better suggestions?
BUCKET_CLEAN_KEYWORD = "kubernetes-staging"

CORS_CONFIG_KEY = "ax-bucket-attributes/cors-config"


class AXGCSBucket(object):
    """
    Wrapper around GCS.
    """
    def __init__(self, bucket_name, aws_profile=None, region=None):
        """
        Initialize a bucket object.

        :param bucket_name: Bucket name.
        :return:
        """
        self._name = bucket_name
        # TODO(shri): will the project change?
        self._gs_client = storage.Client(project="ax-random-project")

        logger.info("Creating object of bucket %s", bucket_name)
        try:
            self._bucket = self._gs_client.get_bucket(bucket_name)
        except NotFound as ne:
            self._bucket = None
            self.create()

    def __repr__(self):
        return "{}".format(self._name)

    def get_bucket_name(self):
        return self._name

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def create(self, **kwargs):
        """
        Create a new bucket.

        :param kwargs: All other creation args.
        :return: True or False
        """
        if self._bucket is not None:
            logger.info("Bucket %s already exists", self._name)
            return True

        self._bucket = self._gs_client.create_bucket(self._name)
        logger.info("Created bucket %s", self._name)
        return True

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def delete(self, force=False):
        """
        Delete a bucket and optionally all object inside it.
        :param force: Auto delete all objects inside.
        :return:
        """
        if not self._exists():
            logger.debug("Bucket %s doesn't exist, don't delete.", self._name)
            return True

        self._bucket.delete(force=force)
        return True

    def put_policy(self, policy):
        # TODO(shri): Implement this!
        return

    def get_policy(self):
        # TODO(shri): Implement this!
        return

    def exists(self):
        """
        Whether a bucket exists.
        :return: True or False
        """
        return self._exists()

    def empty(self):
        return self._empty()

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def put_cors(self, cors, force=False):
        # TODO(shri): Implement this!
        return

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def delete_cors(self):
        # TODO(shri): Implement this!
        return

    def get_cors(self):
        # TODO(shri): Implement this!
        return

    def list_objects(self, keyword="", list_all=False):
        """
        List all objects inside a bucket.
        :return: List of object keys in string
        """
        if not list_all:
            assert keyword is not None, "No keyword provided when not listing all objects"
        if not self._exists():
            logger.info("Bucket does not exist")
            return []
        blobs = []
        logger.info("Retrieving objects with keyword %s ...", keyword)
        for blob in self._bucket.list_blobs():
            if list_all:
                blobs.append(blob.name)
            else:
                if keyword in blob.name:
                    blobs.append(blob.name)

        logger.info("Retrieving objects with keyword %s ... DONE", keyword)
        return keys

    def list_objects_by_prefix(self, prefix):
        """
        List objects in the bucket by the given prefix.
        :param prefix: Prefix string
        :return: List of objects or None
        """
        assert prefix is not None, "Prefix cannot be none."
        return self.list_objects(keyword=prefix)

    def get_object(self, key, **kwargs):
        """
        Get object data specified by key.
        :param key: Key string
        :return: actual object or None
        """
        if not self._exists():
            return None
        try:
            logger.info("Getting object %s", key)
            blob = self._bucket.get_blob(key)
            if blob is None:
                return None
            return blob.download_as_string()
        except Exception as e:
            if "NoSuchKey" not in str(e):
                raise
        return None

    def get_object_url_from_key(self, key):
        return ""

    def put_object(self, key, data, **kwargs):
        """
        Put object into GCS, overwrite by default now.
        :param key: key string
        :param data: data blob
        :return:
        """
        assert "Key" not in kwargs, "Can't pass Key in kwargs"
        assert "Body" not in kwargs, "Can't pass Body in kwargs"
        if not self._exists():
            return True

        try:
            self._bucket.blob(key).upload_from_string(data)
            return True
        except Exception as e:
            logger.warning("Failed to put object into GCS bucket %s, key: %s, data: %s, with error %s", self._bucket, key, data, e)
            return False

    def put_file(self, local_file_name, s3_key, **kwargs):
        """
        # TODO(shri): Implement this!
        """
        return True

    def copy_object(self, source_key, dest_key, **kwargs):
        """
        # TODO(shri): Implement this!
        """
        return True

    def download_file(self, key, file_name, **kwargs):
        """
        # TODO(shri): Implement this!
        """
        return True

    def delete_object(self, key, **kwargs):
        """
        Delete object from GCS.
        :param key:
        :param kwargs:
        :return:
        """
        if not self._exists():
            return True
        try:
            self._bucket.delete_blob(key)
            return True
        except NotFound as ne:
            return True
        except Exception:
            logger.exception("Failed to delete GCS object %s, %s", self._bucket, key)
            return False

    def delete_all(self, obj_prefix="", use_prefix=True, exempt=None):
        """
        # Deletes all the blobs in the bucket!
        """
        num_objs = 0
        for page in self._bucket.list_blobs(prefix=obj_prefix).pages:
            num_objs = num_objs + page.num_items
            self._bucket.delete_blobs(page)
        logger.info("Deleted %s objects", num_objs)
        return True

    def generate_signed_url(self, key):
        """
        Generate a pre signed URL for a object.

        :param key: Object key in string
        :return: Signed URL in string
        """
        # TODO: Find correct way to signed URL.
        # Google's recommended way is to use a service account with downloaded key.
        # URL would be signed locally with private key without consulting GCP servers.
        # This requires us to manage customer's private key, which is not desired.
        # https://googlecloudplatform.github.io/google-cloud-python/stable/storage-blobs.html#
        # https://github.com/GoogleCloudPlatform/google-cloud-python/issues/922
        #
        # Google's alternative approach is to talk to GCP's IAM API and let GCP server
        # sign URL using its managed system key for service account.
        # https://github.com/GoogleCloudPlatform/google-auth-library-python/blob/master/google/auth/iam.py
        # This is preferred by AX but it didn't work with 403 error.
        # $ gcloud iam service-accounts sign-blob --iam-account lcj-zzz@ax-random-project.iam.gserviceaccount.com sign-blob.txt sign-out.txt
        # ERROR: (gcloud.iam.service-accounts.sign-blob) PERMISSION_DENIED: Permission iam.serviceAccounts.signBlob isrequired
        # to perform this operation on service account projects/-/serviceAccounts/lcj-zzz@ax-random-project.iam.gserviceaccount.com.
        #
        # For now, hard code service accoount key for AX account and use that credential to sign URL.
        return NotImplementedError("TODO")

    def _clean(self):
        """
        # TODO(shri): Implement this!
        """
        return True

    def _exists(self):

        @retry(wait_exponential_multiplier=1000,
               stop_max_attempt_number=3)
        def _call_head_bucket_with_retry(gscli, name):
            gscli.lookup_bucket(name)

        try:
            _call_head_bucket_with_retry(self._gs_client, self._name)
            return True
        except Exception as e:
            logger.warning("head_bucket has exception %s", e)
            return False

    def _empty(self):
        if not self._exists():
            logger.info("Bucket does not exist during empty() check")
            return True
        for _ in self._bucket.objects.all():
            return False
        return True

