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
from ax.platform.routes import ServiceEndpoint, NginxIngressController, IngressRules

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
        token = task_obj.jobname
        super(TaskOperation, self).__init__(token)

    @staticmethod
    def prettyname():
        return "TaskOperation"


class Task(KubeObject):
    """
    The job of this object is to track the creation and deletion of a step in a workflow.
    Callers can create this object with the specification from service template, followed
    by call to create, wait_for_start etc
    """
    def __init__(self):
        self.client = KubernetesApiClient(use_proxy=True)
        self.batchapi = self.client.batchv
        self.kube_namespace = "axuser"
        self.jobname = None

        self.service = None     # this is the argo.services.service.Service object
        self._host_vols = []
        self._name_id = AXClusterId().get_cluster_name_id()
        self._s3_bucket_ax_is_external = AXLogPath(self._name_id).is_external()
        self._s3_bucket_ax = AXLogPath(self._name_id).bucket()
        self._s3_key_prefix_ax = AXLogPath(self._name_id).artifact()
        self._s3_bucket = AXClusterDataPath(self._name_id).bucket()
        self._s3_key_prefix = AXClusterDataPath(self._name_id).artifact()

        self._attribute_map = {
            "uuid": "metadata.uid"
        }
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

        self.jobname = Task.generate_job_name(conf)
        job = swagger_client.V1Job()
        labels = {
            "app": self.jobname,
            "service_instance_id": self.service.service_context.service_instance_id,
            "root_workflow_id": self.service.service_context.root_workflow_id,
            "leaf_full_path": string_to_k8s_label(self.service.service_context.leaf_full_path or "no_path"),
            "tier": "devops",
            "role": "user",
        }

        job_meta = swagger_client.V1ObjectMeta()
        job_meta.name = self.jobname
        job_meta.labels = labels
        job.metadata = job_meta

        job_spec = swagger_client.V1JobSpec()

        job_spec.template = self._container_to_pod(labels)
        job.spec = job_spec
        return job

    def start(self, job):
        """
        Start a Job object
        :param job:
        :return:
        """
        assert isinstance(job, swagger_client.V1Job), "Expect to have a swagger_client.V1Job to create"

        @retry_unless(status_code=[409, 422])
        def create_in_provider():
            return self.batchapi.create_namespaced_job(job, self.kube_namespace)

        with TaskOperation(self):
            return create_in_provider()

    def status(self, name, status_obj=None):
        """
        Return the status of the job with the pass name
        Args:
            name: string that is the unique job name
            status_obj: if passed the job status will be used instead of queried from provider

        Returns: a json dict with task status
        """
        self.jobname = name
        job = None
        if status_obj:
            job = status_obj
        else:
            job = self._get_job(name)

        logger.debug("Task status {} is {}".format(name, json.dumps(job.status.to_dict())))
        active = job.status.active == 1
        completed = job.status.succeeded == 1
        if hasattr(job.status, "failed"):
            failed = job.status.failed
        else:
            failed = 0

        status = {
            "active": active,
            "succeeded": completed,
            "message": "",
            "failed": failed
        }
        if active:
            reason = self.get_last_pod_waiting_reason()
            if reason:
                status['reason'] = reason

            # do some cleanup for pods that have failed due to system errors such as mismatch
            # betweeen kubernetes scheduler and kubelet resources
            try:
                self.delete_pods_with_condition(lambda pod: pod.status.phase == "Failed")
            except Exception as e:
                logger.debug("Exception {} in trying to delete pods with condition for job {}".format(e, name))
        try:
            status["message"] = job.status.conditions[0].message if not active and not completed else ""
        except Exception:
            logger.error("job %s. status is %s", job.status, status)
            pass
        return status

    def delete(self, name, delete_pod=True, force=False):
        """
        Delete the task from kubernetes and returns the final status
        Args:
            name: string that is the unique job name

        Returns: Last status of the job

        """
        logger.debug("Task delete for {}".format(name))
        self.jobname = name

        with TaskOperation(self):
            job_status = self._get_job(name)
            status = self.status(name, status_obj=job_status)

            ax_resources = self._get_ax_resources(job_status)
            self.service_instance_id = job_status.metadata.labels.get("service_instance_id", None)
            self._delete_job(name, delete_pod, force)

        # cleanup other ax_resources
        volumepools = ax_resources.get("volumepools", [])
        for poolname, volname in volumepools:
            vmanager = VolumeManager()
            logger.debug("Return volume {} to volume pool {}".format(volname, poolname))
            try:
                vmanager.put_in_pool(poolname, volname, current_ref=self.service_instance_id)
            except AXVolumeOwnershipException as e:
                logger.debug("Volume {} has already been returned for job {}. This is normal".format(volname, name))

        service_endpoint = ax_resources.get("service_endpoint", None)
        if service_endpoint:
            endpoint_name = service_endpoint["name"]
            nginx_controller = service_endpoint.get("use-nginx", None)
            if nginx_controller is None and "persist" in service_endpoint and service_endpoint["persist"]:
                logger.debug("Do not delete service endpoint {} as persist is set to true".format(endpoint_name))
            else:
                s = ServiceEndpoint(endpoint_name)
                s.delete()

                if nginx_controller:
                    r = IngressRules(endpoint_name, nginx_controller)
                    r.delete()

        return status

    def get_status(self):
        status = self._get_job(self.jobname)
        return status.to_dict()

    def get_pod_list(self):
        result = self.client.api.list_namespaced_pod(self.kube_namespace, label_selector="job-name={}".format(self.jobname))
        assert isinstance(result, swagger_client.V1PodList), "Expect object of type V1PodList"
        return result

    def delete_pods_with_condition(self, func):
        """
        Deletes pod for which the passed function returns true
        Args:
            func: The function should have the following prototype boolean Function(V1Pod)
        """
        def sortfunc(item):
            try:
                return parser.parse(item.status.start_time)
            except Exception as e:
                logger.debug("Could not parse start time for pod {} due to exception {}".format(item.metadata.name, e))
                return parser.parse("9999-12-31T23:59:59Z")

        l = self.get_pod_list()
        sorted_list = sorted(l.items, key=sortfunc)
        for pod in sorted_list:
            if func(pod):
                logger.debug("Pod {} matches condition for deletion".format(pod.metadata.name))
                p = Pod(pod.metadata.name, namespace="axuser")
                p.delete()

    def get_last_pod(self):
        try:
            l = self.get_pod_list()
            if len(l.items) == 0:
                raise ValueError("No Pod found for job {}".format(self.jobname))
            p = None
            for i in l.items:
                if p is None or i.status.startTime > p.status.startTime:
                    p = i
            pod = Pod(p.metadata.name)
            pod.build_attributes()
            return pod

        except swagger_client.rest.ApiException as e:
            details = json.loads(e.body)
            raise AXPlatformException(message=details["message"])

    def get_last_pod_waiting_reason(self):
        try:
            l = self.get_pod_list()
            p = None
            for i in l.items:
                if p is None or i.status.startTime > p.status.startTime:
                    p = i
            if p is not None:
                statuses = json.loads(p.metadata.annotations['pod.alpha.kubernetes.io/init-container-statuses'])
                reason = statuses[0]['state']['waiting']['reason']
                logger.debug("got reason %s", reason)
                return reason
        except Exception:
            pass

        return None

    def get_log_endpoint(self, name):
        self.jobname = name
        url_run, _ = self.get_last_pod().get_log_urls()
        return url_run

    @retry_unless(status_code=[404, 409, 422])
    def _get_job(self, name):
        return self.batchapi.read_namespaced_job_status(self.kube_namespace, name)

    def _get_ax_resources(self, status):
        ax_str = status.spec.template.metadata.annotations.get("ax_resources", "{}")
        return json.loads(ax_str)

    def stop_running_pod(self, name):
        self.jobname = name
        self.stop_all_pods(delete_pod=False, force=False)

    def stop_all_pods(self, delete_pod, force):
        if (not delete_pod) and force:
            logger.warning("stop_running_pod(%s, delete_pod=%s, force=%s) has no effect.",
                           self.jobname, delete_pod, force)
            return

        try:
            pods = self.get_pod_list()
            if not force:
                for p in pods.items:
                    if p.status.phase in ["Pending", "Running"]:
                        pod = Pod(p.metadata.name)
                        pod.stop(self.jobname)
                        continue
                    else:
                        logger.debug("Don't need to stop [%s][%s], status=%s", self.jobname, p.metadata.name, p.status.phase)
            else:
                logger.debug("Force delete pods for [%s]", self.jobname)

            if delete_pod:
                logger.debug("Deleting all pods for [%s]", self.jobname)
                self.client.api.deletecollection_namespaced_pod(namespace=self.kube_namespace,
                                                                label_selector="job-name={}".format(self.jobname))
                logger.debug("Deleted all pods for [%s]", self.jobname)
            else:
                logger.debug("Don't delete pods for [%s]", self.jobname)

            for p in pods.items:
                pod = Pod(p.metadata.name)
                logger.debug("Delete volumes for [%s][%s]", self.jobname, p.metadata.name)
                pod._delete_volumes_for_pod(p)

            time.sleep(DELETE_TASK_GRACE_PERIOD)

        except swagger_client.rest.ApiException as e:
            logger.exception("delete_all_pods")
            details = json.loads(e.body)
            raise AXPlatformException(message=details["message"])

    def _delete_job(self, name, delete_pod, force):
        logger.debug("Deleting job for [%s]", self.jobname)
        options = swagger_client.V1DeleteOptions()
        options.grace_period_seconds = 0

        @retry_unless(swallow_code=[404], status_code=[409, 422])
        def delete_in_provider():
            self.batchapi.delete_namespaced_job(options, self.kube_namespace, name)

        delete_in_provider()
        self.stop_all_pods(delete_pod=delete_pod, force=force)

    def _container_to_pod(self, labels):

        # generate the service environment
        self._gen_service_env()

        pod_spec = PodSpec(self.jobname)
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
        artifacts_container.add_env("AX_JOB_NAME", value=self.jobname)

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

        c.add_env("AX_CONTAINER_NAME", value=self.jobname)
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
        c.add_env("AX_JOB_NAME", value=self.jobname)
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
    def generate_job_name(conf):
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
