# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Pod management
"""
import json
import logging
import os
import time

from retrying import retry

from ax.cloud import Cloud
from ax.kubernetes import swagger_client
from ax.kubernetes.client import KubernetesApiClient, parse_kubernetes_exception, retry_unless
from ax.kubernetes.kube_object import KubeObject
from ax.meta import AXClusterId, AXClusterDataPath
from ax.platform.exceptions import AXPlatformException
from ax.platform.ax_monitor import AXKubeMonitor
from ax.platform.ax_monitor_helper import KubeObjWaiter, KubeObjStatusCode
from ax.platform.container import ContainerVolume
from ax.platform.container_specs import InitContainerPullImage, InitContainerTask, SidecarDockerDaemon, InitContainerSetup
from ax.platform.container_specs import SIDEKICK_WAIT_CONTAINER_NAME, DIND_CONTAINER_NAME
from ax.util.ax_artifact import AXArtifacts

DELETE_POD_GRACE_PERIOD = 2
DELETE_WAITER_WAIT_TIMEOUT = 60 * 5
ARTIFACTS_CONTAINER_SCRATCH_PATH = "/ax-artifacts-scratch"

logger = logging.getLogger(__name__)


# TODO: Need to refactor KubeObject a bit so that it understands
#       kubernetes objects that are programmatically generated rather
#       than just from files
class Pod(KubeObject):

    def __init__(self, name, namespace="axuser"):
        self.name = name
        self.namespace = namespace
        self.client = KubernetesApiClient(use_proxy=True)
        self._attribute_map = {
            "nodename": "spec.node_name",
            "nodeip": "status.host_ip",
            "containers": "spec.containers"
        }

    def get_status(self):
        return self._get_status_obj().to_dict()

    def delete(self):

        @parse_kubernetes_exception
        @retry(wait_exponential_multiplier=100,
               stop_max_attempt_number=10)
        def delete_from_provider():
            options = swagger_client.V1DeleteOptions()
            options.grace_period_seconds = 0
            try:
                logger.debug("Trying to delete pod {} from namespace {}".format(self.name, self.namespace))
                self.client.api.delete_namespaced_pod(options, self.namespace, self.name)
                logger.debug("Deleted pod {} from namespace {}".format(self.name, self.namespace))
            except swagger_client.rest.ApiException as e:
                logger.debug("Got the following exception {}".format(e))
                if e.status != 404:
                    raise e

        delete_from_provider()

    def stop(self, jobname=None):
        """
        NOTE: This function assumes that a pod is already running.
        This process kills the user command so that artifacts collection can occur
        Once this is done, the pod will be completed. This call will return when
        pod is completed. Note: pod is not deleted (just completed)
        """

        def get_container_status(s, container_name, name):
            if isinstance(s, dict):
                try:
                    c_status = s.get("containerStatuses", None)
                    for c in c_status or []:
                        n = c.get("name", None)
                        if n == container_name:
                            return c
                except Exception:
                    logger.exception("cannot get_container_status for [%s] [%s]", name, container_name)

            return None

        def get_container_state(s, container_name, name):
            container_status = get_container_status(s, container_name, name=name)
            container_states = ["waiting", "running", "terminated"]
            if isinstance(container_status, dict):
                if "state" in container_status:
                    for state_string in container_states:
                        if state_string in container_status["state"]:
                            # wait if state in state_strings
                            logger.debug("state=%s for [%s] [%s]", state_string, name, container_name)
                            return state_string
                    logger.error("unknown state for [%s] [%s]: %s", name, container_name, s)
                    return None
                else:
                    # No state
                    logger.error("no state for [%s] [%s]: %s", name, container_name, s)
                    return None
            else:
                # no status
                logger.error("no status for [%s] [%s]: %s", name, container_name, s)
                return None

        def get_pod_phase(s):
            if isinstance(s, dict):
                return s.get("phase", None)
            else:
                return None

        def validator_func(pod_status):
            # always return true for any event
            return True

        def send_kill_signal_to_main_container():
            ax_command_path = "/ax-execu-host/art"
            busybox_command_path = os.path.join(ax_command_path, "busybox-i686")
            bash_path = os.path.join(ax_command_path, "ax_bash_ax")
            touch_command = "{} {}".format(busybox_command_path, "touch")
            pgrep_command = "{} {}".format(busybox_command_path, "pgrep")
            xargs_command = "{} {}".format(busybox_command_path, "xargs")
            kill_command = os.path.join(ax_command_path, "ax_kill_ax")
            cat_command = os.path.join(ax_command_path, "ax_cat_ax")

            # execute command to initiate user command kill
            # This command may or may not execute properly if the container is already dying or dead
            # but it does not matter to us here since we will have a waiter. This command will ensure
            # that if a container is running, it will start the process of terminating
            # TODO: we may have pods that are started programmatically that do not have artifacts later
            # HACK HACK
            cmd = [
                bash_path,
                "-c",
                "{touch} {scratch_path}/.ax_delete ;  {kill} -9 `{cat} {scratch_path}/.ax_pid` ".format(
                    touch=touch_command,
                    scratch_path=ARTIFACTS_CONTAINER_SCRATCH_PATH,
                    pgrep=pgrep_command,
                    xargs=xargs_command,
                    kill=kill_command,
                    cat=cat_command
                )
            ]
            logger.debug("Try gracefully stop main container in [%s][%s]. cmd=%s", jobname, self.name, cmd)
            try:
                output = self.exec_commands(cmd)
                logger.debug("Kill output:\n%s", output)
            except Exception:
                logger.exception("exception:")

        main_name = self.get_main_container_name()
        wait_name = SIDEKICK_WAIT_CONTAINER_NAME

        logger.debug("About to stop pod [%s][%s]", jobname, self.name)

        count = 0
        while True:
            count += 1
            if count > 180:
                logger.warning("Pod [%s][%s] too many lopps, abort. count=%s",
                               jobname, self.name, count)
                return False
            obj = {
                "kind": "pods",
                "name": jobname if jobname else self.name,
                "validator": validator_func
            }
            waiter = KubeObjWaiter()
            monitor = AXKubeMonitor()
            monitor.wait_for_kube_object(obj, timeout=DELETE_WAITER_WAIT_TIMEOUT, waiter=waiter)

            # read status here
            read_count = 0
            while True:
                read_count += 1
                if read_count > 180:
                    logger.warning("Pod [%s][%s] too many retry, abort. count=%s",
                                   jobname, self.name, count)
                    return False
                try:
                    status = self._get_status_obj().status
                    assert isinstance(status, swagger_client.V1PodStatus)
                    status_dict = swagger_client.ApiClient().sanitize_for_serialization(status)
                    break
                except Exception:
                    # xxx todo: what if self.name is not there?
                    logger.exception("exception in get status for Pod [%s][%s] retry=%s count=%s",
                                     jobname, self.name, read_count, count)
                    time.sleep(10)
                    continue

            main_container_state = get_container_state(status_dict, main_name, self.name)
            wait_container_state = get_container_state(status_dict, wait_name, self.name)
            pod_phase = get_pod_phase(status_dict)
            logger.debug("Pod [%s][%s] phase=%s. main=%s, wait=%s count=%s",
                         jobname, self.name, pod_phase,
                         main_container_state, wait_container_state, count)

            if main_container_state == "waiting":
                logger.debug("Pod [%s][%s] main in %s count=%s",
                             jobname, self.name, main_container_state, count)
            elif main_container_state == "running":
                logger.debug("Pod [%s][%s] main in %s count=%s",
                             jobname, self.name, main_container_state, count)
                send_kill_signal_to_main_container()
            elif main_container_state is None:
                if pod_phase == "Pending":
                    logger.debug("Pod [%s][%s] in %s phase count=%s",
                                 jobname, self.name, pod_phase, count)
                else:
                    logger.warning("Pod [%s][%s] unknown main container state, abort. %s count=%s",
                                   jobname, self.name, status_dict, count)
                    return False
            else:
                assert main_container_state == "terminated", "bad state {}".format(main_container_state)
                if wait_container_state in ["waiting", "running"]:
                    logger.debug("Pod [%s][%s] wait in %s count=%s",
                                 jobname, self.name, wait_container_state, count)
                    pass
                elif wait_container_state == "terminated":
                    logger.debug("Pod [%s][%s] all containers are terminated. stop() done. count=%s",
                                 jobname, self.name, count)
                    return True
                else:
                    logger.warning("Pod [%s][%s] unknown wait container state, abort. %s. count=%s",
                                   jobname, self.name, status_dict, count)
                    return False

            logger.debug("Pod [%s][%s] wait for new event. count=%s",
                         jobname, self.name, count)
            waiter.wait()
            if waiter.result != KubeObjStatusCode.OK:
                logger.info("Pod [%s][%s] waiter return %s, events: %s",
                            jobname, self.name,
                            waiter.result, waiter.details)
            else:
                logger.debug("Pod [%s][%s] waiter return ok count=%s",
                             jobname, self.name, count)

    def exec_commands(self, commands):
        """
        Return a generator with the output of the command
        """
        cname = self.get_main_container_name()
        client = KubernetesApiClient()
        return client.exec_cmd(self.namespace, self.name, commands, container=cname)

    def get_log_urls(self, service_instance_id):
        cname = self.get_main_container_name()
        url_run = "/api/v1/namespaces/{}/pods/{}/log?container={}&follow=true".format(self.namespace, self.name, cname)

        docker_id = None
        pod = self._get_status_obj()
        cstats = pod.status.container_statuses
        for cstat in cstats:
            if cstat.name != cname:
                continue
            if cstat.container_id is None:
                # Running: The pod has been bound to a node, and all of the containers have been created.
                # At least one container is still running, or is in the process of starting or restarting.
                raise AXPlatformException("log urls can only be obtained after pod {} has started. Current status of container is {}".format(self.name, cstat))
            docker_id = cstat.container_id[len("docker://"):]

        assert docker_id is not None, "Docker ID of created container {} in pod {} was not found".format(self.name, cname)

        name_id = AXClusterId().get_cluster_name_id()
        bucket = AXClusterDataPath(name_id).bucket()
        prefix = AXClusterDataPath(name_id).artifact()
        url_done = "/{}/{}/{}/{}.{}.log".format(bucket, prefix, service_instance_id, cname, docker_id)

        return url_run, url_done

    def get_main_container_name(self):
        if not hasattr(self, "containers"):
            self.build_attributes()
        for c in self.containers:
            if c["name"] not in [SIDEKICK_WAIT_CONTAINER_NAME, DIND_CONTAINER_NAME]:
                return c["name"]

        raise AXPlatformException("Pod for a task needs to have a non-wait container")

    @retry_unless(status_code=[404, 409, 422])
    def _get_status_obj(self):
        status = self.client.api.read_namespaced_pod_status(self.namespace, self.name)
        assert isinstance(status, swagger_client.V1Pod), "Status object should be of type V1Pod"
        return status

    @staticmethod
    def massage_pod_status(v1pod):
        pod_field_map = {
            "name": "metadata.name",
            "generation": "metadata.annotations.ax_generation",
            "phase": "status.phase",
            "startTime": "status.start_time",
            "message": "status.message",
            "reason": "status.reason",
            "labels": "metadata.labels"
        }

        container_field_map = {
            "name": "name",
            "container_id": "container_id",
            "state": "state",
            "ready": "ready",
            "restart_count": "restart_count",
            "image": "image",
            "image_id": "image_id"
        }

        # Get information from pod_field_map
        pstatus = KubeObject.swagger_obj_extract(v1pod, pod_field_map, serializable=True)

        # fill containers in pod
        pstatus["containers"] = []
        for container_status in v1pod.status.container_statuses or []:
            cstatus = KubeObject.swagger_obj_extract(container_status, container_field_map, serializable=True)
            if cstatus["name"] == DIND_CONTAINER_NAME:
                continue
            # fix container_id field
            if cstatus["container_id"]:
                cstatus["container_id"] = cstatus["container_id"][len("docker://"):]
            pstatus["containers"].append(cstatus)
        return pstatus


"""
All the code for creating Pod Specifications and Pod Resources
"""


class PodSpec(object):
    """
    Create a Pod spec
    """

    def __init__(self, name, namespace="axuser"):
        self.name = name
        self.namespace = namespace
        self.cmap = {}
        self.annotations = {}
        self.labels = {}
        self.restart_policy = None
        self._artifact_vols = []
        self._tier = "user"
        self.hostname = None

    def set_tier(self, t):
        self._tier = t

    def enable_docker(self, size_in_mb):
        if "main" not in self.cmap:
            raise AXPlatformException("Pod needs to have main container before enabling docker")

        # create the dind sidecar container
        dind_c = SidecarDockerDaemon(size_in_mb)
        if Cloud().in_cloud_aws():
            dind_c.args = ["--storage-driver=overlay2"]
        elif Cloud().in_cloud_gcp():
            # Current GKE defaults to overlay.
            dind_c.args = ["--storage-driver=overlay"]
        self.cmap["dind"] = dind_c
        self.cmap["main"].add_env("DOCKER_HOST", value="tcp://localhost:2375")

        return dind_c

    def enable_artifacts(self, namespace, version, sid, in_artifacts_spec):
        if "main" not in self.cmap:
            raise AXPlatformException("Pod needs to have main and wait container before enabling artifacts")

        # Add an init container that gets artifacts and creates mappings in the main container
        c_setup = InitContainerSetup()
        customer_image = self.cmap["main"].image
        c_pullimage = InitContainerPullImage(customer_image)
        c_artifacts = InitContainerTask(customer_image, namespace, version)

        self.add_init_container(c_setup)
        self.add_init_container(c_pullimage)
        self.add_init_container(c_artifacts)

        # set the command in the main container and add volume mount
        self.cmap["main"].command = ["{}/executor.sh".format(ARTIFACTS_CONTAINER_SCRATCH_PATH)]
        artifacts_vol = c_artifacts.get_artifacts_volume()
        artifacts_vol.set_mount_path(ARTIFACTS_CONTAINER_SCRATCH_PATH)
        self.cmap["main"].add_volume(artifacts_vol)

        static_bins_vol = c_artifacts.get_static_bins_volume()
        static_bins_vol.set_mount_path("/ax-execu-host")
        self.cmap["main"].add_volume(static_bins_vol)

        def generate_volumes_for_artifacts():

            test_mode = False
            if AXArtifacts.is_test_service_instance(sid):
                test_mode = True

            art_volumes = AXArtifacts.get_extra_artifact_in_volume_mapping(
                in_artifacts_spec, ARTIFACTS_CONTAINER_SCRATCH_PATH, "in", test_mode=test_mode, self_sid=sid)

            ret_vols = []
            initc_vols = []
            i = 0
            already_mapped = {}
            for initc_path, mount_path in art_volumes or []:
                name = "ax-art-{}".format(i)
                c = ContainerVolume(name, mount_path)
                c.set_type("EMPTYDIR")
                c_init = ContainerVolume(name, initc_path)
                c_init.set_type("EMPTYDIR")
                i += 1
                if mount_path not in already_mapped:
                    ret_vols.append(c)
                    initc_vols.append(c_init)
                    already_mapped[mount_path] = True

            return ret_vols, initc_vols

        # Add artifacts to main container
        (self._artifact_vols, initc_vols) = generate_volumes_for_artifacts()
        self.cmap["main"].add_volumes(self._artifact_vols)
        c_artifacts.add_volumes(initc_vols)

        return c_artifacts

    def get_artifact_vols(self):
        return self._artifact_vols

    def add_init_container(self, c):
        if "init" not in self.cmap:
            self.cmap["init"] = []
        self.cmap["init"].append(c)

    def add_main_container(self, c):
        self.cmap["main"] = c

    def add_wait_container(self, c):
        self.cmap["wait"] = c

    def add_annotation(self, key, val):
        if isinstance(val, dict):
            val = json.dumps(val)
        self.annotations[key] = val

    def add_label(self, key, val):
        self.labels[key] = val

    def get_spec(self):

        # generate the metadata
        metadata = swagger_client.V1ObjectMeta()
        metadata.name = self.name
        metadata.annotations = {
            "pod.beta.kubernetes.io/init-containers": self._init_containers_spec()
        }
        for a in self.annotations:
            metadata.annotations[a] = self.annotations[a]

        metadata.labels = {}
        for l in self.labels:
            metadata.labels[l] = self.labels[l]

        # generate the pod specification
        pspec = swagger_client.V1PodSpec()
        if self.hostname:
            pspec.hostname = self.hostname
        pspec.containers = []

        if "wait" in self.cmap:
            pspec.containers.append(self.cmap["wait"].generate_spec())

        assert "main" in self.cmap, "Pod specification cannot be generated without a main container"
        pspec.containers.append(self.cmap["main"].generate_spec())

        if "dind" in self.cmap:
            pspec.containers.append(self.cmap["dind"].generate_spec())

        pspec.image_pull_secrets = self._build_image_pull_secrets()
        pspec.volumes = self._volume_spec()

        if self.restart_policy is not None:
            pspec.restart_policy = self.restart_policy

        pspec.node_selector = {
            "ax.tier": self._tier
        }

        # finalize the pod template spec
        spec = swagger_client.V1PodTemplateSpec()
        spec.metadata = metadata
        spec.spec = pspec

        return spec

    """
    Helper routines used by get_spec() function of this class
    """

    def _volume_spec(self):
        vmap = {}
        vols = []
        for c in self._container_iterator():
            for v in c.volume_iterator():
                if v.name in vmap:
                    continue
                vols.append(v.pod_spec())
                vmap[v.name] = v

        return vols

    def _init_containers_spec(self):
        init_c = []

        if "init" not in self.cmap:
            return json.dumps(init_c)

        for c in self.cmap["init"] or []:
            c_spec = c.generate_spec()
            c_formatted = swagger_client.ApiClient().sanitize_for_serialization(c_spec)
            init_c.append(c_formatted)

        return json.dumps(init_c)

    def _build_image_pull_secrets(self):
        secrets_arr = []
        regmap = {}

        def append_to_secrets(container):
            registry = container.get_registry(namespace=self.namespace)
            if registry is not None and registry not in regmap:
                reg = swagger_client.V1LocalObjectReference()
                reg.name = registry
                secrets_arr.append(reg)
                regmap[registry] = True

        for ctype, c in self.cmap.iteritems():
            if ctype == "init":
              for cinit in c or []:
                  append_to_secrets(cinit)
            else:
                append_to_secrets(c)

        return secrets_arr

    def _container_iterator(self):
        for ctype, c in self.cmap.iteritems():
            if ctype == "init":
              for cinit in c or []:
                  yield cinit
            else:
                yield c
