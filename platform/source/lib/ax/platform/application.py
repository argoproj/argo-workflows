# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
import json
import logging
import os
import time
import uuid

from ax.exceptions import AXTimeoutException
from ax.kubernetes import swagger_client
from ax.kubernetes.client import KubernetesApiClient, retry_not_exists, retry_unless_not_found, retry_unless
from ax.kubernetes.kube_object import KubeObjectConfigFile
from ax.platform.cluster_config import AXClusterConfig, ClusterProvider
from ax.platform.exceptions import AXPlatformException
from ax.platform.component_config import AXPlatformConfigDefaults, SoftwareInfo
from ax.platform.routes import InternalRoute


logger = logging.getLogger(__name__)

DEFAULT_SECRET_YAML_PATH = os.path.join(AXPlatformConfigDefaults.DefaultManifestRoot, "registry-secrets.yml.in")
DEFAULT_AM_YAML_PATH = os.path.join(AXPlatformConfigDefaults.DefaultManifestRoot, "axam-svc.yml.in")


class Application(object):
    """
    Create an Application which maps to a kubernetes namespace
    """
    def __init__(self, name, client=None):
        self.name = name
        if client is None:
            self._client = KubernetesApiClient(use_proxy=True)
        else:
            self._client = client

        self._registry_spec = None
        self._software_info = SoftwareInfo()
        if self._software_info.registry_is_private():
            secret = KubeObjectConfigFile(DEFAULT_SECRET_YAML_PATH, {"REGISTRY_SECRETS": self._software_info.registry_secrets})
            for obj in secret.get_swagger_objects():
                if isinstance(obj, swagger_client.V1Secret):
                    self._registry_spec = obj
            assert self._registry_spec, "Argo registry specification is missing"

        self._am_service_spec = None
        self._am_deployment_spec = None

        # AA-2471: Hack to add AXOPS_EXT_DNS to Application Manager
        elb = InternalRoute("axops", "axsys", client=self._client)
        elb_status = elb.status(with_loadbalancer_info=True)["loadbalancer"][0]
        if not elb_status:
            raise AXPlatformException("Could not get axops elb address {}".format(elb_status))

        replacements = {"NAMESPACE": self._software_info.image_namespace,
                        "VERSION": self._software_info.image_version,
                        "REGISTRY": self._software_info.registry,
                        "APPLICATION_NAME": self.name,
                        "AXOPS_EXT_DNS": elb_status}
        cluster_name_id = os.getenv("AX_CLUSTER_NAME_ID", None)
        assert cluster_name_id, "Cluster name id is None!"
        cluster_config = AXClusterConfig(cluster_name_id=cluster_name_id)
        if cluster_config.get_cluster_provider() != ClusterProvider.USER:
            axam_path = DEFAULT_AM_YAML_PATH
        else:
            axam_path = "/ax/config/service/argo-all/axam-svc.yml.in"
            replacements["ARGO_DATA_BUCKET_NAME"] = os.getenv("ARGO_DATA_BUCKET_NAME")

        logger.info("Using replacements: %s", replacements)

        k = KubeObjectConfigFile(axam_path, replacements)
        for obj in k.get_swagger_objects():
            if isinstance(obj, swagger_client.V1Service):
                self._am_service_spec = obj
            elif isinstance(obj, swagger_client.V1beta1Deployment):
                self._am_deployment_spec = obj
                self._add_pod_metadata("deployment", self._am_deployment_spec.metadata.name, is_label=True)
                self._add_pod_metadata("ax_costid", json.dumps({
                    "app": self.name,
                    "service": "axam-deployment",
                    "user": "system"
                }))
            else:
                logger.debug("Ignoring specification of type {}".format(type(obj)))
        assert self._am_service_spec and self._am_deployment_spec, "Application monitor specification is missing"

    def _add_pod_metadata(self, key, value, is_label=False):
        """
        Helper function to add metadata to deployment pod spec for AXAM
        """
        pod_meta = self._am_deployment_spec.spec.template.metadata
        if is_label:
            if pod_meta.labels is None:
                pod_meta.labels = {}
            pod_meta.labels[key] = value
        else:
            if pod_meta.annotations is None:
                pod_meta.annotations = {}
            pod_meta.annotations[key] = value

    def create(self, force_recreate=False):
        """
        Create a kubernetes namespace and populate it with argo registry

        Idempotency: This function will be idempotent as long as the content
        of the secret is not changed. If create is called with a registry secret
        that has been updated and the namespace with the secret already exists
        then it will not update the secret for now.
        """

        @retry_not_exists
        def create_ns_in_provider():
            namespace = swagger_client.V1Namespace()
            namespace.metadata = swagger_client.V1ObjectMeta()
            namespace.metadata.name = self.name
            self._client.api.create_namespace(namespace)

        # NOTE: 403 is not retried as application is getting deleted in parallel
        # 422 is unprocessable object (aka error in spec)
        @retry_unless(status_code=[403, 422])
        def create_reg_in_provider():
            if self._registry_spec is None:
                return
            try:
                self._client.api.create_namespaced_secret(self._registry_spec, self.name)
            except swagger_client.rest.ApiException as e:
                if e.status == 409:
                    self._client.api.patch_namespaced_secret(self._registry_spec.to_dict(), self.name, self._registry_spec.metadata.name)
                else:
                    raise e

        @retry_unless(status_code=[403, 422])
        def create_app_monitor_service_in_provider():
            try:
                self._client.api.create_namespaced_service(self._am_service_spec, self.name)
            except swagger_client.rest.ApiException as e:
                if e.status == 409:
                    self._client.api.patch_namespaced_service(self._am_service_spec.to_dict(), self.name, self._am_service_spec.metadata.name)
                else:
                    raise e

        @retry_unless(status_code=[403, 422])
        def create_app_monitor_deployment_in_provider():
            try:
                self._client.apisappsv1beta1_api.create_namespaced_deployment(self._am_deployment_spec, self.name)
            except swagger_client.rest.ApiException as e:
                if e.status == 409:
                    if force_recreate:
                        # add a new metadata in pod spec to force the recreation of pods
                        self._add_pod_metadata("applatix.io/force-recreate-salt", str(uuid.uuid4()))

                    self._client.apisappsv1beta1_api.replace_namespaced_deployment(self._am_deployment_spec, self.name, self._am_deployment_spec.metadata.name)
                else:
                    raise e

        try:
            logger.debug("Creating application {}".format(self.name))
            create_ns_in_provider()
            logger.debug("Created namespace {}".format(self.name))
            create_reg_in_provider()
            create_app_monitor_service_in_provider()
            logger.debug("Created application monitor service {}".format(self._am_service_spec.metadata.name))
            create_app_monitor_deployment_in_provider()
            logger.debug("Created application monitor deployment {}".format(self._am_deployment_spec.metadata.name))
        except Exception as e:
            logger.exception(e)

    def delete(self, timeout=None):
        """
        Delete a kubernetes namespace and image secret for Argo

        Idempotency: Can be repeatedly called
        """
        delete_grace_period = 1
        options = swagger_client.V1DeleteOptions()
        options.grace_period_seconds = delete_grace_period
        options.orphan_dependents = False

        @retry_unless(swallow_code=[404, 409])
        def delete_ns_in_provider():
            """
            The retry is not done for 404 (not found) and also for 409 (conflict)
            The 404 case is for simple retry. 409 happens when application delete was
            requested but not complete and another request came in.
            """
            logger.debug("Deleting application {}".format(self.name))
            self._client.api.delete_namespace(options, self.name)

        delete_ns_in_provider()

        start_time = time.time()
        while self.exists():
            logger.debug("Application {} still exists".format(self.name))
            time.sleep(delete_grace_period+1)
            wait_time = int(time.time() - start_time)
            if timeout is not None and wait_time > timeout:
                raise AXTimeoutException("Could not delete namespace {} in {} seconds".format(self.name, timeout))

    def exists(self):

        @retry_unless_not_found
        def get_ns_in_provider():
            try:
                stat = self._client.api.read_namespace(self.name)
                return True
            except swagger_client.rest.ApiException  as e:
                if e.status == 404:
                    return False
                else:
                    raise e

        return get_ns_in_provider()

    def status(self):
        """
        This function checks the following:
        1. Namespace exists?
        2. Argo Registry exists?
        3. TODO: Application Monitor exists
        Returns:
            A json dict with the status of each
            {
                'namespace': True/False,
                'registry': True/False,
                'monitor': True/False
            }
        """
        ret = {
            'namespace': False,
            'registry': False,
            'monitor': False
        }

        if not self.exists():
            return ret

        ret['namespace'] = True
        ns = self._get_registry_from_provider()
        if ns is None:
            return ret

        ret['registry'] = True
        srv = self._get_am_service_from_provider()
        if srv is None:
            return ret

        am_dep = self._get_am_deployment_from_provider()
        if am_dep is not None and am_dep.status.available_replicas == am_dep.status.replicas:
            ret["monitor"] = True

        return ret

    def healthy(self):
        """
        If all components are present/healthy then return True
        else return False
        """
        d = self.status()
        for component in d:
            if not d[component]:
                return False
        return True

    def events(self, name=None):
        return self._get_events_from_provider(name).items

    @retry_unless(swallow_code=[404])
    def _get_registry_from_provider(self):
        if self._registry_spec is not None:
            return self._client.api.read_namespaced_secret(self.name, self._registry_spec.metadata.name)
        else:
            return "NotNeeded"

    @retry_unless(swallow_code=[404])
    def _get_am_service_from_provider(self):
        return self._client.api.read_namespaced_service(self.name, self._am_service_spec.metadata.name)

    @retry_unless(swallow_code=[404])
    def _get_am_deployment_from_provider(self):
        return self._client.apisappsv1beta1_api.read_namespaced_deployment(self.name, self._am_deployment_spec.metadata.name)

    @retry_unless(swallow_code=[404])
    def _get_events_from_provider(self, name):
        # XXX: For some reason list_namespaced_event does not take a namespace but the _21 version
        #      of the function does. Hopefully this gets fixed in swagger soon
        field_selector = None
        if name is not None:
            field_selector = "involvedObject.name={}".format(name)
        return self._client.api.list_namespaced_event(self.name, field_selector=field_selector)


class Applications(object):
    """
    This class is for aggregate operations on a group of applications
    """
    def __init__(self, client=None):
        if client is None:
            self.client = KubernetesApiClient(use_proxy=True)
        else:
            self.client = client
        self.ignored_namespaces = frozenset(["kube-system", "default", "axsys", "axuser", "kube-public"])

    def list(self):
        return [x.metadata.name for x in self._get_namespaces().items if x.metadata.name not in self.ignored_namespaces]

    @retry_unless()
    def _get_namespaces(self):
        return self.client.api.list_namespace()
