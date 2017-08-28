# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Code for creating executing a step in a workflow
"""
import base64
import json
import logging
import time
import os

from future.utils import iteritems
from pyparsing import Word, nums, alphanums
from dateutil import parser

from ax.kubernetes import swagger_client
from ax.kubernetes.kube_object import KubeObject
from ax.kubernetes.client import KubernetesApiClient, retry_unless
from ax.meta import AXLogPath, AXClusterDataPath, AXClusterId, AXCustomerId
from ax.platform.component_config import SoftwareInfo
from ax.platform.exceptions import AXPlatformException, AXVolumeOwnershipException
from ax.platform.operations import Operation
from ax.platform.volumes import VolumeManager
from ax.platform.container import ContainerVolume
from ax.platform.cluster_config import AXClusterConfig
from ax.util.ax_random import random_string
from ax.util.converter import string_to_dns_label, string_to_k8s_label
from .pod import Pod, PodSpec
from .container_specs import SidecarTask
from .container import Container

from argo.services.service import Service

logger = logging.getLogger(__name__)

DELETE_TASK_GRACE_PERIOD = 2

# minimum amount of memory that is required by docker for any container
MEM_MIB_MIN = 4

# minimum amount of memory that is needed for docker in docker.
# I experimentally found this to be 16 MB so I doubled that value here.
MEM_MIB_MIN_DOCKER_ENABLE = 32

# The maximum length of job name
MAX_JOB_NAME = 55

# URL for user pods to access cluster metadata
CLUSTER_META_URL_V1 = os.getenv("AX_CLUSTER_META_URL_V1")


class TaskOperation(Operation):

    def __init__(self, task_obj):
        token = task_obj.name
        super(TaskOperation, self).__init__(token)

    @staticmethod
    def prettyname():
        return "TaskOperation"


class Task(object):
    """
    The job of this object is to track the creation and deletion of a step in a workflow.
    Callers can create this object with the specification from service template, followed
    by call to create, wait_for_start etc
    """
    def __init__(self, name, namespace="axuser"):
        self.name = name
        self.namespace = namespace
        self.client = KubernetesApiClient(use_proxy=True)

        self.service = None     # this is the argo.services.service.Service object
        self._host_vols = []
        self._name_id = AXClusterId().get_cluster_name_id()
        self._s3_bucket_ax_is_external = AXLogPath(self._name_id).is_external()
        self._s3_bucket_ax = AXLogPath(self._name_id).bucket()
        self._s3_key_prefix_ax = AXLogPath(self._name_id).artifact()
        self._s3_bucket = AXClusterDataPath(self._name_id).bucket()
        self._s3_key_prefix = AXClusterDataPath(self._name_id).artifact()

        self.software_info = SoftwareInfo()
        self._ax_resources = {}

    def create(self, conf):
        """
        Create a Kubernetes Job object
        :param conf: conf data from DevOps
        :return:
        """
        logger.debug("Task create for {}".format(json.dumps(conf)))
        self.service = Service()
        self.service.parse(conf)

        labels = {
            "app": self.name,
            "service_instance_id": self.service.service_context.service_instance_id,
            "root_workflow_id": self.service.service_context.root_workflow_id,
            "leaf_full_path": string_to_k8s_label(self.service.service_context.leaf_full_path or "no_path"),
            "tier": "devops",
            "role": "user",
        }

        template_spec = self._container_to_pod(labels)

        # convert V1PodTemplateSpec to V1Pod
        pod_spec = swagger_client.V1Pod()
        pod_spec.metadata = template_spec.metadata
        pod_spec.spec = template_spec.spec
        return pod_spec

    def start(self, spec):
        """
        Start a task
        :param spec: The swagger specification for Pod
        :type spec: swagger_client.V1Pod
        :return:
        """
        assert isinstance(spec, swagger_client.V1Pod), "Unexpected object {} in Task.start".format(type(spec))

        @retry_unless(status_code=[409, 422])
        def create_in_provider():
            return self.client.api.create_namespaced_pod(spec, self.namespace)

        with TaskOperation(self):
            return create_in_provider()

    def status(self, status_obj=None):
        """
        Return the status of the job with the pass name
        Args:
            status_obj: if passed the job status will be used instead of queried from provider

        Returns: a json dict with task status
        """
        if status_obj:
            assert isinstance(status_obj, swagger_client.V1Pod), "Unexpected status object {} in Task.status".format(type(status_obj))
            status = status_obj
        else:
            status = Pod(self.name, self.namespace)._get_status_obj()

        # pkg/api/v1/types.go in kubernetes source code describes the following PodPhase
        # const (
        #     // PodPending means the pod has been accepted by the system, but one or more of the containers
        #     // has not been started. This includes time before being bound to a node, as well as time spent
        #     // pulling images onto the host.
        #     PodPending PodPhase = "Pending"
        #     // PodRunning means the pod has been bound to a node and all of the containers have been started.
        #     // At least one container is still running or is in the process of being restarted.
        #     PodRunning PodPhase = "Running"
        #     // PodSucceeded means that all containers in the pod have voluntarily terminated
        #     // with a container exit code of 0, and the system is not going to restart any of these containers.
        #     PodSucceeded PodPhase = "Succeeded"
        #     // PodFailed means that all containers in the pod have terminated, and at least one container has
        #     // terminated in a failure (exited with a non-zero exit code or was stopped by the system).
        #     PodFailed PodPhase = "Failed"
        #     // PodUnknown means that for some reason the state of the pod could not be obtained, typically due
        #     // to an error in communicating with the host of the pod.
        #     PodUnknown PodPhase = "Unknown"
        # )
        pending = True if status.status.phase == "Pending" else False
        active = True if status.status.phase == "Running" or pending else False
        failed = True if status.status.phase == "Failed" else False
        completed = True if status.status.phase == "Succeeded" or failed else False

        if status.status.phase == "Unknown":
            raise AXPlatformException("Status of task {} could not be found due to some temporary problem. Please retry".format(self.name))

        ret_status = {
            "active": active,
            "succeeded": completed,
            # TODO: Should this be removed
            "message": "",
            "reason": "",
            "failed": failed
        }

        if pending:
            # If the pod is pending it is likely stuck on container image pull failure
            try:
                for init_status in status.status.init_container_statuses or []:
                    if init_status.state.waiting is not None:
                        # find the first init container that is stuck on waiting and stuff the reason and message
                        ret_status["reason"] = init_status.state.waiting.reason or ""
                        ret_status["message"] = init_status.state.waiting.message or ""
                        break
            except Exception:
                pass

        return ret_status

    def delete(self, force=False):
        """
        Delete the task from kubernetes and returns the final status
        Returns: Last status of the job
        """
        logger.debug("Task delete for {}".format(self.name))
        with TaskOperation(self):
            status = self.status()
            p = Pod(self.name, self.namespace)
            if not force:
                p.stop()
            p.delete()
            return status

    def get_log_endpoint(self):
        url_run, _ =  Pod(self.name, self.namespace).get_log_urls()
        return url_run

    def _get_ax_resources(self, status):
        ax_str = status.spec.template.metadata.annotations.get("ax_resources", "{}")
        return json.loads(ax_str)

    def stop(self):
        Pod(self.name, self.namespace).stop()

    def _container_to_pod(self, labels):

        # generate the service environment
        self._gen_service_env()

        pod_spec = PodSpec(self.name)
        pod_spec.restart_policy = "Never"

        main_container = self._container_spec()

        for vol_tag, vol in iteritems(self.service.template.inputs.volumes):
            # sanitize name for kubernetes
            vol_tag = string_to_dns_label(vol_tag)
            cvol = ContainerVolume(vol_tag, vol.mount_path)
            assert "resource_id" in vol.details and "filesystem" in vol.details, "resource_id and filesystem are required fields in volume details"
            cvol.set_type("AWS_EBS", vol_tag, vol.details["resource_id"], vol.details["filesystem"])
            main_container.add_volume(cvol)
            logger.info("Mounting volume {} {} in {}".format(vol_tag, vol.details, vol.mount_path))

        pod_spec.add_main_container(main_container)
        wait_container = self._generate_wait_container_spec()
        pod_spec.add_wait_container(wait_container)

        (cpu, mem, d_cpu, d_mem) = self._container_resources()
        main_container.add_resource_constraints("cpu_cores", cpu, limit=None)
        main_container.add_resource_constraints("mem_mib", mem, limit=mem)

        # handle artifacts
        self_sid = None
        if self.service.service_context:
            self_sid = self.service.service_context.service_instance_id

        # TODO: This function calls ax_artifact and needs to be rewritten. Ugly code.
        artifacts_container = pod_spec.enable_artifacts(self.software_info.image_namespace, self.software_info.image_version, self_sid, self.service.template.to_dict())
        artifacts_container.add_env("AX_JOB_NAME", value=self.name)

        if self.service.template.docker_spec:
            dind_c = pod_spec.enable_docker(self.service.template.docker_spec.graph_storage_size_mib)
            dind_c.add_volumes(pod_spec.get_artifact_vols())
            dind_c.add_resource_constraints("cpu_cores", d_cpu, limit=None)
            dind_c.add_resource_constraints("mem_mib", d_mem, limit=d_mem)

        service_id = None
        if self.service.service_context:
            service_id = self.service.service_context.service_instance_id
        pod_spec.add_annotation("ax_serviceid", service_id)
        pod_spec.add_annotation("ax_costid", json.dumps(self.service.costid))
        pod_spec.add_annotation("ax_resources", json.dumps(self._ax_resources))
        pod_spec.add_annotation("AX_SERVICE_ENV", self._gen_service_env())

        for k in labels or []:
            pod_spec.add_label(k, labels[k])

        return pod_spec.get_spec()

    def _container_spec(self):
        """
        Converts service template to V1Container
        """
        container = self.service.template
        c = Container(container.name, container.image, pull_policy=container.image_pull_policy)

        c.add_env("AX_CONTAINER_NAME", value=self.name)
        c.add_env("AX_ROOT_SERVICE_INSTANCE_ID", value=self.service.service_context.root_workflow_id)
        c.add_env("AX_SERVICE_INSTANCE_ID", value=self.service.service_context.service_instance_id)

        # Envs introduced to user
        c.add_env("AX_POD_NAME", value_from="metadata.name")
        c.add_env("AX_POD_IP", value_from="status.podIP")
        c.add_env("AX_POD_NAMESPACE", value_from="metadata.namespace")
        c.add_env("AX_NODE_NAME", value_from="spec.nodeName")
        c.add_env("AX_CLUSTER_META_URL_V1", value=CLUSTER_META_URL_V1)

        for env in container.env:
            c.add_env(env.name, value=env.value)

        return c

    def _container_resources(self):

        container = self.service.template

        cpu = float(container.resources.cpu_cores)
        mem = float(container.resources.mem_mib)
        main_cpu = cpu
        main_mem = mem
        dind_cpu = 0.0
        dind_mem = 0.0

        if container.docker_spec:
            dind_cpu = float(container.docker_spec.cpu_cores)
            dind_mem = float(container.docker_spec.mem_mib)
            main_cpu = cpu
            main_mem = mem

            if dind_mem < MEM_MIB_MIN_DOCKER_ENABLE:
                raise ValueError("mem_mib must have a minimum value of {} for docker support".format(MEM_MIB_MIN_DOCKER_ENABLE))

            if main_mem < MEM_MIB_MIN:
                raise ValueError("mem_mib must have a minimum value of {}MB".format(MEM_MIB_MIN))

        return main_cpu, main_mem, dind_cpu, dind_mem

    def _generate_wait_container_spec(self):

        main_container_name = self.service.template.name

        c = SidecarTask(main_container_name, self.software_info.image_namespace, self.software_info.image_version)
        c.add_env("AX_MAIN_CONTAINER_NAME", value=main_container_name)
        c.add_env("AX_JOB_NAME", value=self.name)
        c.add_env("AX_CUSTOMER_ID", AXCustomerId().get_customer_id())
        c.add_env("AX_REGION", AXClusterConfig().get_region())
        c.add_env("AX_CLUSTER_NAME_ID", self._name_id)

        return c

    def _gen_service_env(self):
        service_env = {
            "container": {"docker": {}},
            "s3_bucket": self._s3_bucket,
            "s3_key_prefix": self._s3_key_prefix,
            "s3_bucket_ax_is_external": self._s3_bucket_ax_is_external,
            "s3_bucket_ax": self._s3_bucket_ax,
            "s3_key_prefix_ax": self._s3_key_prefix_ax,
            "docker_enable": self.service.template.docker_spec is not None
        }

        container = self.service.template
        # if container is not 'once', i.e. need to be restarted if failed,
        # set keep_return_code to be True so the inner_executor will pass-through the return code
        service_env["keep_return_code"] = not container.once

        service_env["container"]["docker"]["commands"] = container.command
        service_env["container"]["docker"]["args"] = container.args
        if container.inputs.count() > 0:
            service_env["container"]["inputs"] = container.inputs.to_dict()
        if container.outputs.count() > 0:
            service_env["container"]["outputs"] = container.outputs.to_dict()
        if self.service.service_context:
            service_env["container"]["service_context"] = self.service.service_context.to_dict()
            service_env["container"]["service_context"]["name"] = self.service.name

        # use base64 encode then decode to accommodate all chars in json
        # xxx todo: which unicode encode to use?
        return base64.b64encode(json.dumps(service_env))

    @staticmethod
    def generate_name(conf):
        """
        This function generates a kubernetes job name from a service template and also
        ensures that the generated name has some relationship to human readable job names 
        while also being unique.
        :param conf: service template 
        :return: job name string
        """
        if not conf["template"].get("once", True):
            # name is fully specified by caller. This is currently only used by
            # workflow executor. No user jobs are expected to use this code path
            # Workflow executor generates a unique name for the workflow so we
            # do not have to worry about generating one for it.
            name = conf.get("name", None)
            if name is None:
                raise ValueError("name is a required field in service object for once=false.")
            return string_to_dns_label(name)
        else:
            return string_to_dns_label(conf["id"])

    @staticmethod
    def insert_defaults(conf):
        """
        This function inserts default that are required for Task processing
        :param conf: input conf
        :return: output conf
        """
        if conf["template"].get("name", None) is None:
            conf["template"]["name"] = "main"
        else:
            conf["template"]["name"] = string_to_dns_label(conf["template"]["name"])
        return conf
