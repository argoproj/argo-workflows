# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
This module manages kubernetes secrets
"""
import logging

from retrying import retry
from future.utils import with_metaclass

from ax.util.singleton import Singleton
from ax.kubernetes import swagger_client
from ax.kubernetes.client import KubernetesApiClient, parse_kubernetes_exception, retry_unless

logger = logging.getLogger(__name__)


def reformat_name(name):
    """
    This function returns a new name that fits kubernetes naming format
    https://kubernetes.io/docs/user-guide/identifiers/#names
    """
    return name.replace(":", "-")


class SecretsManager(with_metaclass(Singleton, object)):

    def __init__(self):
        self.client = KubernetesApiClient(use_proxy=True)

    def insert_imgpull(self, name, namespace, token):
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: applatix-registry
        data:
            .dockerconfigjson: XXX
        type: kubernetes.io/dockerconfigjson
        """
        name = reformat_name(name)
        secret = swagger_client.V1Secret()
        secret.metadata = swagger_client.V1ObjectMeta()
        secret.metadata.name = name
        secret.data = {
            ".dockerconfigjson": token
        }
        secret.type = "kubernetes.io/dockerconfigjson"

        # always delete a secret if it exists
        self.delete_imgpull(name, namespace)
        self._create_in_provider(namespace, secret)

    def delete_imgpull(self, name, namespace):

        name = reformat_name(name)

        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def delete(namespace, name):
            options = swagger_client.V1DeleteOptions()
            options.grace_period_seconds = 0
            try:
                logger.debug("Delete secret: {}".format(name))
                self.client.api.delete_namespaced_secret(options, namespace, name)
            except swagger_client.rest.ApiException as e:
                if e.status != 404:
                    raise e

        delete(namespace, name)

    def get_imgpull(self, name, namespace):

        name = reformat_name(name)

        # return false on ApiError else retry
        @retry_unless(swallow_code=[404])
        def exists(namespace, name):
            secret = self.client.api.read_namespaced_secret(namespace, name)
            assert isinstance(secret, swagger_client.V1Secret) , "Expect an instance of V1Secret"
            if secret.type == "kubernetes.io/dockerconfigjson":
                return secret
            return None

        return exists(namespace, name)

    def copy_imgpull(self, secret, to_ns):
        """
        Copy secret to a new namespace
        Args:
            secret: V1Secret
            to_ns: name of namespace
        """
        new_secret = swagger_client.V1Secret()
        new_secret.metadata = swagger_client.V1ObjectMeta()
        new_secret.metadata.name = secret.metadata.name
        new_secret.data = secret.data
        new_secret.type = "kubernetes.io/dockerconfigjson"

        self._create_in_provider(to_ns, new_secret)

    @retry_unless(swallow_code=[409], status_code=[422])
    def _create_in_provider(self, namespace, secret):
        try:
            logger.debug("Creating secret: {}/{}".format(namespace, secret.metadata.name))
            self.client.api.create_namespaced_secret(secret, namespace)
        except swagger_client.rest.ApiException as e:
            if e.status == 409:
                logger.debug("Replacing secret: {}/{}".format(namespace, secret.metadata.name))
                self.client.api.replace_namespaced_secret(secret, namespace, secret.metadata.name)
            else:
                raise e
