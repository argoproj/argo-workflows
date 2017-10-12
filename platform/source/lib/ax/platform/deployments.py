# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
import base64
import copy
import json
import logging
import os
import re

from future.utils import iteritems
from retrying import retry

from ax.cloud.aws.elb import visibility_to_elb_addr, visibility_to_elb_name
from ax.exceptions import AXNotFoundException, AXApiForbiddenReq
from ax.kubernetes import swagger_client
from ax.kubernetes.client import KubernetesApiClient, retry_unless
from ax.kubernetes.kube_object import KubeObject
from ax.meta import AXClusterId
from ax.platform.applet import CUR_RECORD_VERSION
from ax.platform.application import Application
from ax.platform.cluster_config import AXClusterConfig
from ax.platform.container import Container, ContainerVolume
from ax.platform.component_config import SoftwareInfo
from ax.platform.operations import Operation
from ax.platform.pod import PodSpec, Pod
from ax.platform.resources import AXResources
from ax.platform.routes import InternalRoute, ExternalRoute, ExternalRouteVisibility
from ax.platform.secrets import SecretResource
from ax.platform.volumes import AXNamedVolumeResource
from ax.util.converter import string_to_dns_label


logger = logging.getLogger(__name__)

# URL for user pods to access cluster metadata
CLUSTER_META_URL_V1 = os.getenv("AX_CLUSTER_META_URL_V1")


class DeploymentOperation(Operation):

    def __init__(self, obj):
        token = "{}/{}".format(obj.application, obj.name)
        super(DeploymentOperation, self).__init__(token=token)

    @staticmethod
    def prettyname():
        return "DeploymentOperation"


class Deployment(object):
    """
    This class creates and manages a single deployment object
    A deployment consists of the following specifications in kubernetes
    1. A kubernetes deployment spec
    2. Zero or more kubernetes service specs
    3. Zero or more ingress rules

    All functions in the object need to be idempotent.
    """

    def __init__(self, name, application):
        """
        Each deployment has a name and needs to be part of an application
        Application maps to a kubernetes namespace and the deployment will
        be created in this namespace.

        Args:
            name: deployment name
            application: the application that this deployment runs under
        """
        self.name = name
        self.application = application
        self.client = KubernetesApiClient(use_proxy=True)
        self._nameid = AXClusterId().get_cluster_name_id()
        self._software_info = SoftwareInfo()

        self._app_obj = Application(application)

        self._resources = AXResources()
        self.spec = None

        self._cluster_config = AXClusterConfig()

    def create(self, spec):
        """
        Create a deployment from the template specified

        Idempotency: This function is idempotent. A create of identical spec will
        have no impact if the deployment already exists. If the spec is different
        then the existing deployment will be updated.
        """
        @retry_unless(status_code=[404, 422])
        def create_in_provider(k8s_spec):
            try:
                logger.info("Creating deployment %s in Kubernetes namespace %s", self.name, self.application)
                self.client.apisappsv1beta1_api.create_namespaced_deployment(k8s_spec, self.application)
                logger.info("Done creating deployment %s in Kubernetes namespace %s", self.name, self.application)
            except swagger_client.rest.ApiException as e:
                if e.status == 409:
                    self.client.apisappsv1beta1_api.replace_namespaced_deployment(k8s_spec, self.application, self.name)
                else:
                    raise e

        with DeploymentOperation(self):

            self.spec = spec

            # Do some template checks
            self._template_checks()

            # First create supplemental resources such as routes, ingress rules etc
            self._create_deployment_resources()

            # Now create the deployment spec
            d_spec = self._create_deployment_spec()

            # Store the resources in the deployment spec
            self._resources.finalize(d_spec)

            # Create the deployment object in kubernetes
            create_in_provider(d_spec)

    def delete(self, timeout=None):
        """
        Delete the deployment.

        Idempotency: This function is idempotent. If deployment does not exist then
        delete will silently fail without raising any exceptions.
        Args:
            timeout: In seconds or None for infinite
        """
        options = swagger_client.V1DeleteOptions()
        options.grace_period_seconds = 1
        options.orphan_dependents = False

        def check_result(result):
            # True for retry False for done
            return not result

        @retry(retry_on_result=check_result, wait_fixed=2000, stop_max_delay=timeout)
        def wait_for_scale_to_zero():
            logger.debug("Wait for scale of deployment to 0 for {} {}".format(self.application, self.name))

            @retry_unless(swallow_code=[404])
            def get_scale_from_provider():
                return self.client.apisappsv1beta1_api.read_namespaced_scale_scale(self.application, self.name)

            scale = get_scale_from_provider()
            if scale is None:
                return True
            if scale.status.replicas == 0:
                return True

            return False

        @retry_unless(swallow_code=[404, 409])
        def delete_in_provider():
            logger.debug("Deleting deployment for {} {}".format(self.application, self.name))
            self.client.apisappsv1beta1_api.delete_namespaced_deployment(options, self.application, self.name)

        def delete_rs_in_provider():
            logger.debug("Deleting replica set for {} {}".format(self.application, self.name))
            self.client.extensionsv1beta1.deletecollection_namespaced_replica_set(self.application, label_selector="deployment={}".format(self.name))

        # now delete deployment object and replication set
        with DeploymentOperation(self):
            dep_obj = self._deployment_status()
            self._scale_to(0)
            wait_for_scale_to_zero()

            if dep_obj:
                resources = AXResources(existing=dep_obj)
                resources.delete_all()

            delete_in_provider()
            delete_rs_in_provider()

    def status(self):
        """
        Get the status of the deployment.
        Returns: Returns the entire V1Deployment as a dict.
        If deployment is not found then this will raise an AXNotFoundException (404)
        """
        # STEP 1: Get status of deployment
        stat = self._deployment_status()
        if stat is None:
            raise AXNotFoundException("Deployment {} not found in application {}".format(self.name, self.application))

        dep_field_map = {
            "name": "metadata.name",
            "generation": "metadata.annotations.ax_generation",
            "desired_replicas": "status.replicas",
            "available_replicas": "status.available_replicas",
            "unavailable_replicas": "status.unavailable_replicas"
        }
        ret = KubeObject.swagger_obj_extract(stat, dep_field_map, serializable=True)

        # STEP 2: Get the pods for the deployment and events associated
        podlist = self._deployment_pods().items
        dep_events = self._app_obj.events(name=self.name)
        event_field_map = {
            "message": "message",
            "reason": "reason",
            "source": "source.component",
            "host": "source.host",
            "firstTS": "first_timestamp",
            "lastTS": "last_timestamp",
            "count": "count",
            "container": "involved_object.field_path",
            "type": "type"
        }
        ret["events"] = []
        for event in dep_events:
            ret["events"].append(KubeObject.swagger_obj_extract(event, event_field_map, serializable=True))

        ret["pods"] = []
        for pod in podlist or []:
            # fill pod status and containers
            pstatus = Pod.massage_pod_status(pod)

            # fill events for pod
            pstatus["events"] = []

            events = self._app_obj.events(name=pod.metadata.name)
            for event in events:
                pstatus["events"].append(KubeObject.swagger_obj_extract(event, event_field_map, serializable=True))

            # fill pod failure information for pod based on events
            pstatus["failure"] = Deployment._pod_failed_info(pstatus)

            ret["pods"].append(pstatus)

        # STEP 3: From the deployment spec get the resources created by deployment
        # TODO: Add this when services are created by deployment

        return ret

    def get_labels(self):
        """
        Get a dict of labels used for this deployment
        """
        state = self._deployment_status()
        if state is None:
            raise AXNotFoundException("Did not find deployment {} in application {}".format(self.name, self.application))

        return KubeObject.swagger_obj_extract(state, {"labels": "spec.selector.match_labels"})['labels']

    @staticmethod
    def _pod_failed_info(pod_status):
        if pod_status["phase"] != "Pending":
            return None

        for ev in pod_status["events"] or []:
            if ev["reason"] == "Failed" and ev["source"] == "kubelet" and ev["type"] == "Warning" and \
                            "Failed to pull image" in ev["message"] and ev["count"] > 5:
                return {
                    "reason": "ImagePullFailure",
                    "message": ev["message"]
                }
        return None

    def scale(self, replicas):
        with DeploymentOperation(self):
            # Deployments with volumes can't be scaled to > 1.
            if replicas > 1:
                dep_obj = self._deployment_status()
                if dep_obj:
                    resources = AXResources(existing=dep_obj)
                    for type in resources.get_all_types():
                        if type.startswith("ax.platform.volumes"):
                            raise AXApiForbiddenReq("Deployments with volumes can't be scaled to > 1 ({})".format(replicas))

            self._scale_to(replicas)

    @retry_unless(swallow_code=[404])
    def _deployment_status(self):
        return self.client.apisappsv1beta1_api.read_namespaced_deployment(self.application, self.name)

    @retry_unless(swallow_code=[404])
    def _deployment_pods(self):
        return self.client.api.list_namespaced_pod(self.application, label_selector="deployment={}".format(self.name))

    def _create_deployment_spec(self):

        pod_spec = PodSpec(self.name, namespace=self.application)
        main_container = self.spec.template.get_main_container()

        main_container_spec = self._create_main_container_spec(main_container)
        pod_spec.add_main_container(main_container_spec)

        container_vols = self._get_main_container_vols()
        main_container_spec.add_volumes(container_vols)

        hw_res = main_container.get_resources()
        main_container_spec.add_resource_constraints("cpu_cores", hw_res.cpu_cores, limit=None)
        main_container_spec.add_resource_constraints("mem_mib", hw_res.mem_mib, limit=None)

        artifacts_container = pod_spec.enable_artifacts(self._software_info.image_namespace, self._software_info.image_version,
                                                        None, main_container.to_dict())
        secret_resources = artifacts_container.add_configs_as_vols(main_container.get_all_configs(), self.name, self.application)
        self._resources.insert_all(secret_resources)

        # Set up special circumstances based on annotations
        # Check if we need to circumvent the executor script. This is needed for containers that run
        # special init processes such as systemd as these processes like to be pid 1
        if main_container.executor_spec:
            main_container_spec.command = None
            if main_container.docker_spec is not None:
                raise ValueError("We do not support ax_ea_docker_enable with ax_ea_executor")

        # Does this container need to be privileged
        main_container_spec.privileged = main_container.privileged

        # Check if docker daemon sidecar needs to be added
        if main_container.docker_spec:
            # graph storage size is specified in GiB
            dind_container_spec = pod_spec.enable_docker(main_container.docker_spec.graph_storage_size_mib)
            dind_container_spec.add_volumes(pod_spec.get_artifact_vols())
            dind_container_spec.add_resource_constraints("cpu_cores", main_container.docker_spec.cpu_cores, limit=None)
            dind_container_spec.add_resource_constraints("mem_mib", main_container.docker_spec.mem_mib, limit=None)
            dind_container_spec.add_volumes(container_vols)

        # Do we only need docker graph storage volume for the main container
        if main_container.graph_storage:
            dgs_vol = ContainerVolume("graph-storage-vol-only", main_container.graph_storage.mount_path)
            dgs_vol.set_type("DOCKERGRAPHSTORAGE", main_container.graph_storage.graph_storage_size_mib)
            main_container_spec.add_volume(dgs_vol)

        # set the pod hostname to value provided in main container spec
        pod_spec.hostname = main_container.hostname

        # TODO: This needs fixup. job name is used in init container to ask permission to start
        # TODO: Don't know if this is needed in deployment or not?
        artifacts_container.add_env("AX_JOB_NAME", value=self.application)
        artifacts_container.add_env("AX_DEPLOYMENT_NEW", value="True")

        if len(container_vols) > 0:
            tmp_container_vols = copy.deepcopy(container_vols)
            volume_paths = []
            for v in tmp_container_vols:
                v.set_mount_path("/ax/fix" + v.volmount.mount_path)
                volume_paths.append(v.volmount.mount_path)
            artifacts_container.add_volumes(tmp_container_vols)
            logger.info("Volumes to chmod: %s", volume_paths)
            artifacts_container.add_env("AX_VOL_MOUNT_PATHS", value=str(volume_paths))

        # add annotation for service env which will show up in artifacts container
        pod_spec.add_annotation("AX_SERVICE_ENV", self._generate_service_env(self.spec.template))
        pod_spec.add_annotation("AX_IDENTIFIERS", self._get_identifiers())
        if self.spec.costid:
            pod_spec.add_annotation("ax_costid", json.dumps(self.spec.costid))

        pod_spec.add_label("deployment", self.name)
        pod_spec.add_label("application", self.application)
        pod_spec.add_label("tier", "user")
        pod_spec.add_label("deployment_id", self.spec.id)

        # now that pod is ready get its spec and wrap it in a deployment
        k8s_spec = self._generate_deployment_spec_for_pod(pod_spec.get_spec())

        logger.info("Generated Kubernetes spec for deployment %s", self.name)
        return k8s_spec

    def _create_main_container_spec(self, container_template):
        """
        :type container_template: argo.template.v1.container.ContainerTemplate
        :rtype Container
        """
        logger.debug("Container template is {}".format(container_template))

        name = string_to_dns_label(container_template.name)
        container_spec = Container(name, container_template.image, pull_policy=container_template.image_pull_policy)
        container_spec.parse_probe_spec(container_template)

        # Necessary envs for handshake
        container_spec.add_env("AX_HANDSHAKE_VERSION", value=CUR_RECORD_VERSION)

        # Envs introduced to user
        container_spec.add_env("AX_POD_NAME", value_from="metadata.name")
        container_spec.add_env("AX_POD_IP", value_from="status.podIP")
        container_spec.add_env("AX_POD_NAMESPACE", value_from="metadata.namespace")
        container_spec.add_env("AX_NODE_NAME", value_from="spec.nodeName")
        container_spec.add_env("AX_CLUSTER_META_URL_V1", value=CLUSTER_META_URL_V1)

        # envs from user spec
        for env in container_template.env:
            (cfg_ns, cfg_name, cfg_key) = env.get_config()
            if cfg_ns is not None:
                secret = SecretResource(cfg_ns, cfg_name, self.name, self.application)
                secret.create()
                self._resources.insert(secret)
                container_spec.add_env(env.name, value_from_secret=(secret.get_resource_name(), cfg_key))
            else:
                container_spec.add_env(env.name, value=env.value)

        # Unix socket for applet
        applet_sock = ContainerVolume("applet", "/tmp/applatix.io/")
        applet_sock.set_type("HOSTPATH", "/var/run/")
        container_spec.add_volume(applet_sock)

        return container_spec

    @staticmethod
    def _get_valid_name_from_axrn(axrn):
        # AXRN's will have non-alphanumeric characters such as : / @, etc which K8S doesn't
        # like in its PVC name. Replace all non-alphanumeric characters with -.
        name_regex = re.compile(r"\W+")
        return name_regex.sub("-", axrn).replace("_", "-")

    def _get_main_container_vols(self):
        container_template = self.spec.template.get_main_container()
        ret = []

        for vol_name, vol in  iteritems(container_template.inputs.volumes):
            # sanitize the volume name for kubernetes
            vol_name = string_to_dns_label(vol_name)
            cvol = ContainerVolume(vol_name, vol.mount_path)
            assert "resource_id" in vol.details, "Volume resource-id absent in volume details"
            assert "filesystem" in vol.details, "Volume filesystem absent in volume details"
            cvol.set_type("AWS_EBS", vol_name, vol.details["resource_id"], vol.details["filesystem"])
            logger.debug("Volume {} {} mounted at {}".format(vol_name, vol.details, vol.mount_path))
            ret.append(cvol)

        return ret

    def _generate_service_env(self, template):
        return base64.b64encode(json.dumps(template.to_dict()))

    def _get_identifiers(self):
        return {
            "application_id": self.spec.app_generation,
            "deployment_id": self.spec.id,
            "static": {
                "application_id": self.spec.app_id,
                "deployment_id": self.spec.deployment_id
            }
        }

    def _generate_deployment_spec_for_pod(self, pod_spec):

        metadata = swagger_client.V1ObjectMeta()
        metadata.name = self.name

        dspec = swagger_client.V1beta1DeploymentSpec()
        dspec.strategy = self._get_strategy()
        if self.spec.template.min_ready_seconds:
            dspec.min_ready_seconds = self.spec.template.min_ready_seconds
        dspec.selector = swagger_client.V1LabelSelector()
        dspec.selector.match_labels = {
            "deployment": self.name
        }

        dspec.replicas = self.spec.template.scale.min
        dspec.template = pod_spec

        deployment_obj = swagger_client.V1beta1Deployment()
        deployment_obj.metadata = metadata
        deployment_obj.spec = dspec
        return deployment_obj

    def _create_deployment_resources(self):

        for route in self.spec.template.internal_routes:
            # ignore empty port spec
            if len(route.ports) == 0:
                logger.debug("Skipping internal route {} as port spec is empty".format(route.name))
                continue
            ir = InternalRoute(route.name, self.application)
            ir.create(route.to_dict()["ports"], selector={"deployment": self.name}, owner=self.name)
            self._resources.insert(ir)
            logger.debug("Created route {}".format(ir))

        for route in self.spec.template.external_routes:

            dns_name = route.dns_name()
            if dns_name.endswith("."):
                dns_name = dns_name[:-1]

            r = ExternalRoute(dns_name, self.application, {"deployment": self.name}, route.target_port, route.ip_white_list, route.visibility)

            elb_addr = None
            elb_name = None

            if not self._cluster_config.get_cluster_provider().is_user_cluster():
                try:
                    elb_addr = visibility_to_elb_addr(route.visibility)
                    elb_name = visibility_to_elb_name(route.visibility)
                except AXNotFoundException:
                    if route.visibility == ExternalRouteVisibility.VISIBILITY_WORLD:
                        raise AXNotFoundException("Could not find the public ELB. Please report this error to Applatix Support at support@applatix.com")
                    else:
                        assert route.visibility == ExternalRouteVisibility.VISIBILITY_ORGANIZATION, "Only world and organization are currently supported as visibility attributes"
                        raise AXNotFoundException("Please create a private ELB using the template named 'ax_private_elb_creator_workflow' before using 'visibility=organization'")

            name = r.create(elb_addr,elb_name=elb_name)
            self._resources.insert(r)
            logger.debug("Created external route {} for {}/{}/{}".format(name, self.application, self.name, dns_name))

        main_container = self.spec.template.get_main_container()
        for key_name, vol in iteritems(main_container.inputs.volumes):
            assert "resource_id" in vol.details, "Volume resource_id absent in volume details"
            name = vol.details.get("axrn", None)
            resource_id = vol.details.get("resource_id", None)
            assert name is not None and resource_id is not None, "axrn and resource_id are required details for volume {}".format(key_name)
            nv_res = AXNamedVolumeResource(name, resource_id)
            nv_res.create()
            self._resources.insert(nv_res)
            logger.debug("Using named volume resource {} in application {}".format(name, self.application))

    @retry_unless(status_code=[422], swallow_code=[400, 404])
    def _scale_to(self, replicas):
        logger.debug("Scaling deployment to {} for {} {}".format(replicas, self.application, self.name))
        scale = swagger_client.V1beta1Scale()
        scale.spec = swagger_client.V1beta1ScaleSpec()
        scale.spec.replicas = replicas
        scale.metadata = swagger_client.V1ObjectMeta()
        scale.metadata.name = self.name
        scale.metadata.namespace = self.application
        self.client.apisappsv1beta1_api.replace_namespaced_scale_scale(scale, self.application, self.name)

    def _template_checks(self):
        if self.spec.template.scale and self.spec.template.scale.min > 1 and len(self.spec.template.volumes) >= 1:
            raise ValueError("Deployments with volumes can't have scale > 1 ({})".format(self.spec.template.scale.min))

    def _get_strategy(self):
        s = swagger_client.V1beta1DeploymentStrategy()
        s.type = "RollingUpdate" if self.spec.template.strategy.type == "rolling_update" else "Recreate"
        if s.type == "RollingUpdate":
            rolling_update = swagger_client.V1beta1RollingUpdateDeployment()
            rolling_update.max_unavailable = self.spec.template.strategy.rolling_update.max_unavailable
            rolling_update.max_surge = self.spec.template.strategy.rolling_update.max_surge
            s.rolling_update = rolling_update
        return s
