# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
This module manages kubernetes secrets
"""
import base64
import json
import logging

from retrying import retry
from future.utils import with_metaclass, iteritems

from ax.platform.resources import AXResource
from ax.util.hash import generate_hash
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


class SecretResource(AXResource):

    @staticmethod
    def create_object_from_info(info):
        cfg_ns = info["config_namespace"]
        cfg_name = info["config_name"]
        workflow_step = info["step_name"]
        namespace_step = info["step_namespace"]
        return SecretResource(cfg_ns, cfg_name, workflow_step, namespace_step)

    def __init__(self, cfg_ns, cfg_name, workflow_step, namespace_step):
        self.namespace_step = namespace_step
        self._orig_secret_name = ConfigToSecret(cfg_ns, cfg_name).gen_name()
        self._step_secret_name = generate_hash("config-{}-{}".format(workflow_step, self._orig_secret_name))
        self._manager = SecretsManager()
        self.ax_meta = {
            "config_namespace": cfg_ns,
            "config_name": cfg_name,
            "step_name": workflow_step,
            "step_namespace": namespace_step
        }

    def create(self):
        self._manager.copy_generic_to(self._orig_secret_name, self._step_secret_name, to_namespace=self.namespace_step)

    def delete(self):
        self._manager.delete_generic(self._step_secret_name, namespace=self.namespace_step)

    def status(self):
        return {}

    def get_resource_name(self):
        return self._step_secret_name

    def get_resource_info(self):
        return self.ax_meta

    def __str__(self):
        return "ConfigSecret {}".format(json.dumps(self.ax_meta))


class ConfigToSecret(object):

    def __init__(self, cfg_ns, cfg_name):
        self.namespace = cfg_ns
        self.name = cfg_name
        self._manager = SecretsManager()
        self._secret_name = self.gen_name()

    def create(self, data, metadata=None):
        meta = metadata or {}
        meta["config-name"] = self.name
        meta["config-namespace"] = self.namespace
        self._manager.create_generic(self._secret_name, data, metadata=meta)

    def delete(self):
        self._manager.delete_generic(self._secret_name)

    def get(self):
        return self._manager.get_generic(self._secret_name)

    def gen_name(self):
        return generate_hash("config-{}-{}".format(self.namespace, self.name))


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

    #
    # CRUD operations for generic secrets
    #
    def create_generic(self, name, data, metadata=None, namespace="axsys"):
        """
        Create/update a generic kubernetes secret from configuration
        :param name: name of secret
        :param data: key, value pairs
        :param metadata: key, value pairs of metadata to store with secret
                         metadata is not loaded with secret into the container
        :param namespace: The namespace to create secret in
        NOTE: update only if secret already exists in the namespace.
        """
        logger.debug("Create secret {}/{}".format(name, namespace))
        obj = swagger_client.V1Secret()
        obj.metadata = swagger_client.V1ObjectMeta()
        obj.metadata.name = name
        obj.metadata.annotations = {
            "user_metadata": json.dumps(metadata)
        }
        obj.type = "Opaque"
        enc_data = {}
        for k,v in iteritems(data):
            enc_data[k] = base64.b64encode(v)
        obj.data = enc_data
        self._create_in_provider(namespace, obj)
        logger.debug("Create secret {}/{} complete".format(name, namespace))

    def delete_generic(self, name, namespace="axsys"):
        """
        Delete the secret
        :param name: secret name
        :param namespace: namespace of secret
        """
        logger.debug("Delete secret {}/{}".format(name, namespace))
        self._delete_in_provider(namespace, name)
        logger.debug("Delete secret {}/{} complete".format(name, namespace))

    def copy_generic_to(self, name, new_name, from_namespace="axsys", to_namespace="axuser"):
        """
        This create a copy of the secret with a new name to a new namespace
        :param secret: secret to copy
        :param new_secret: new secret to create
        :param from_namespace: the namespace of the original secret
        :param to_namespace: the namespace to copy to
        """
        logger.debug("copy secret {}/{} => {}/{}".format(name, from_namespace, new_name, to_namespace))
        (data, meta) = self.get_generic(name, from_namespace)
        self.create_generic(new_name, data, metadata=meta, namespace=to_namespace)
        logger.debug("copy secret {}/{} => {}/{} complete".format(name, from_namespace, new_name, to_namespace))

    def get_generic(self, name, namespace="axsys"):
        """
        Return the metadata and data for the secret
        :param secret: name of secret
        :param namespace: namespace of secret
        :return: tuple of data and metadata
        """
        logger.debug("Get secret {}/{}".format(name, namespace))
        obj = self._get_from_provider(namespace, name)
        dec_data = {}
        for k,v in iteritems(obj.data):
            dec_data[k] = base64.b64decode(v)
        metadata = obj.metadata.annotations.get("user_metadata", None)
        if metadata:
            metadata = json.loads(metadata)
        logger.debug("Get secret {}/{} complete".format(name, namespace))
        return dec_data, metadata

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

    @retry_unless(swallow_code=[404], status_code=[409, 422])
    def _delete_in_provider(self, namespace, secret):
        options = swagger_client.V1DeleteOptions()
        options.grace_period_seconds = 0
        self.client.api.delete_namespaced_secret(options, namespace, secret)

    @retry_unless(status_code=[404, 409, 422])
    def _get_from_provider(self, namespace, secret):
        return self.client.api.read_namespaced_secret(namespace, secret)