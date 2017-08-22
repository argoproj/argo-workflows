# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Code for creating executing a step in a workflow
"""
import logging
import os
import re
import shlex

from ax.kubernetes import swagger_client
from ax.kubernetes.kube_object import KubeObject
from ax.util.docker_command import docker_options_to_envvar, docker_options_to_ports
from ax.exceptions import AXIllegalArgumentException
from ax.platform.component_config import SoftwareInfo
from ax.platform.secrets import SecretsManager
from ax.platform.volumes import VolumeManager
from ax.util.docker_image import DockerImage
from ax.devops.axdb.axops_client import AxopsClient


logger = logging.getLogger(__name__)

axops_client = AxopsClient()


class ContainerImagePullPolicy:
    PullAlways = "Always"
    PullNever = "Never"
    PullIfNotPresent = "IfNotPresent"


class ContainerVolume(object):
    """
    This class is used to hold information about
    volumes that can be attached to a container.
    A pod spec for volumes can then be generated
    using the union of all container volumes that
    are attached to all the containers in the pod
    """

    def _unset(self):
        pass

    def _emptydir(self):
        # use host space for empty dir
        self.vol.empty_dir = swagger_client.V1EmptyDirVolumeSource()

    def _hostpath(self, hostpath):
        self.vol.host_path = swagger_client.V1HostPathVolumeSource()
        self.vol.host_path.path = hostpath

    def _downward_api(self, field):
        self.vol.downward_api = swagger_client.V1DownwardAPIVolumeSource()
        item = swagger_client.V1DownwardAPIVolumeFile()
        item.path = "annotations"
        item.field_ref = swagger_client.V1ObjectFieldSelector()
        item.field_ref.field_path = field
        self.vol.downward_api.items = [item]

    def _docker_graph_storage(self, size_in_mb):
        flexvol = swagger_client.V1FlexVolumeSource()
        flexvol.driver = "ax/vol_plugin"
        flexvol.fs_type = "ext4"
        flexvol.options = {
            "size_mb": str(size_in_mb),
            "ax_vol_type": "ax-docker-graph-storage"
        }
        self.vol.flex_volume = flexvol

    def _pvc(self, claim_name):
        pvc_spec = swagger_client.V1PersistentVolumeClaimVolumeSource()
        pvc_spec.claim_name = claim_name
        self.vol.persistent_volume_claim = pvc_spec

    def _aws_ebs(self, volume_name, resource_id, filesystem):
        volume_spec = swagger_client.V1AWSElasticBlockStoreVolumeSource()
        volume_spec.volume_id = resource_id
        volume_spec.fs_type = filesystem
        volume_spec.name = volume_name
        self.vol.aws_elastic_block_store = volume_spec

    def __init__(self, name, mount_path):
        self.name = name
        self.vol = swagger_client.V1Volume()
        self.vol.name = name
        self.typename = None
        self.volmount = swagger_client.V1VolumeMount()
        self.volmount.name = name
        self.volmount.mount_path = mount_path

        self.typedict = {
            "UNSET": self._unset,
            "EMPTYDIR": self._emptydir,
            "HOSTPATH": self._hostpath,
            "DOWNWARDAPI": self._downward_api,
            "DOCKERGRAPHSTORAGE": self._docker_graph_storage,
            "PVC": self._pvc,
            "AWS_EBS": self._aws_ebs,
        }
        self.set_type("UNSET")

    def __str__(self):
        return "ContainerVolume:: Name: {}, Mount Path: {}".format(self.vol.name, self.volmount.mount_path)

    def set_type(self, typename, *args, **kwargs):
        func = self.typedict.get(typename, None)
        if func is not None and callable(func):
            func(*args, **kwargs)
            self.typename = typename

    def set_mount_path(self, mount_path):
        self.volmount.mount_path = mount_path

    def get_container_spec(self):
        assert self.typename is not None, "set_type must be called before getting container spec"
        return self.volmount

    def pod_spec(self):
        assert self.typename is not None, "set_type must be called before getting pod spec"
        return self.vol


class Container(KubeObject):
    """
    Class for creating container specifications
    """

    LIVENESS_PROBE = 1
    READINESS_PROBE = 2

    def __init__(self, name, image, pull_policy=None):
        """
        Construct a container that will provide the spec for a kubernetes container
        http://kubernetes.io/docs/api-reference/v1/definitions/#_v1_container
        Args:
            name: name of a container. must be conformant to kubernetes container name
            image: image for container
            pull_policy: pull policy based on kubernetes. If None then kubernetes default is used
        """
        self.name = name
        self.image = image
        self.image_pull_policy = pull_policy

        self.command = None
        self.args = None

        self.vmap = {}
        self.env_map = {}
        self.ports = []

        self.resources = None
        self.privileged = None

        self.software_info = SoftwareInfo()
        self.probes = {}

    def generate_spec(self):
        c = swagger_client.V1Container()
        c.name = self.name
        c.image = self.image

        if self.resources is not None:
            c.resources = swagger_client.V1ResourceRequirements()
            c.resources.requests = {}
            c.resources.limits = {}
            if "cpu_cores" in self.resources:
                c.resources.requests["cpu"] = str(self.resources["cpu_cores"][0])
                if self.resources["cpu_cores"][1] is not None:
                    c.resources.limits["cpu"] = str(self.resources["cpu_cores"][1])

            if "mem_mib" in self.resources:
                c.resources.requests["memory"] = "{}Mi".format(self.resources["mem_mib"][0])
                if self.resources["mem_mib"][1] is not None:
                    c.resources.limits["memory"] = "{}Mi".format(self.resources["mem_mib"][1])

        # Kubernetes 1.5 requires init container must specify image pull policy. Since we are setting
        # a pull policy for all containers, we want to replicate the kubernetes default behavior of pulling
        # the image if tag is "latest"
        if self.image.endswith(':latest'):
            c.image_pull_policy = ContainerImagePullPolicy.PullAlways
        else:
            c.image_pull_policy = self.image_pull_policy or ContainerImagePullPolicy.PullIfNotPresent

        if self.command:
            c.command = self.command
        if self.args:
            c.args = self.args

        c.volume_mounts = []
        for _, vol in self.vmap.iteritems():
            c.volume_mounts.append(vol.get_container_spec())

        c.env = []
        for _, env in self.env_map.iteritems():
            c.env.append(env)

        if self.privileged is not None:
            c.security_context = swagger_client.V1SecurityContext()
            c.security_context.privileged = self.privileged

        for probe in self.probes:
            probe_spec = self.probes[probe]
            probe_k8s_spec = Container._generate_probe_spec(probe_spec)
            if probe == Container.LIVENESS_PROBE:
                c.liveness_probe = probe_k8s_spec
            elif probe == Container.READINESS_PROBE:
                c.readiness_probe = probe_k8s_spec
            else:
                raise AXIllegalArgumentException("Unexpected probe type {} found with spec {}".format(probe, probe_spec))

        return c

    def add_resource_constraints(self, resource, request, limit=None):
        if self.resources is None:
            self.resources = {}
        self.resources[resource] = (request, limit)

    def add_volume(self, volume):
        self.vmap[volume.name] = volume

    def add_volumes(self, volumes):
        for vol in volumes or []:
            self.add_volume(vol)

    def get_volume(self, name):
        return self.vmap.get(name, None)

    def add_env(self, name, value=None, value_from=None):
        env = swagger_client.V1EnvVar()
        env.name = name
        if value is not None:
            env.value = value
        else:
            assert value_from is not None, "value and value_from both cannot be None for env {}".format(name)
            env.value_from = swagger_client.V1EnvVarSource()
            env.value_from.field_ref = swagger_client.V1ObjectFieldSelector()
            env.value_from.field_ref.field_path = value_from
            # Some 1.5 requires this. https://github.com/kubernetes/kubernetes/issues/39189
            env.value_from.field_ref.api_version = "v1"

        self.env_map[name] = env

    def add_probe(self, probe_type, probe_spec):
        self.probes[probe_type] = probe_spec

    def parse_probe_spec(self, container_template):
        """
        @type container_template: argo.template.v1.container.ContainerTemplate
        """
        if container_template.liveness_probe:
            probe_type = Container.LIVENESS_PROBE
            self.add_probe(probe_type, container_template.liveness_probe)
        if container_template.readiness_probe:
            probe_type = Container.READINESS_PROBE
            self.add_probe(probe_type, container_template.readiness_probe)

    def get_registry(self, namespace="axuser"):
        """
        This function returns the name of the secrets file that needs to be
        used in the pod specification image_pull_secrets array
        """
        (reg, _, _) = DockerImage(fullname=self.image).docker_names()
        if reg == self.software_info.registry:
            if self.software_info.registry_is_private():
                return "applatix-registry"
            else:
                return None
        else:
            try:
                smanager = SecretsManager()
                secret = smanager.get_imgpull(reg, namespace)
                if secret:
                    return secret.metadata.name

                # Code for copying the registry to the app namespace if
                # it does not exist. We do not copy to axuser as secrets
                # are always created there.
                secret_axuser = smanager.get_imgpull(reg, "axuser")
                if secret_axuser and namespace != "axuser":
                    smanager.copy_imgpull(secret_axuser, namespace)
                    return secret_axuser.metadata.name
            except Exception as e:
                logger.debug("Did not find a secret for registry {} due to exception {}".format(reg, e))
            return None

    def volume_iterator(self):
        for _, vol in self.vmap.iteritems():
            yield vol

    @staticmethod
    def _generate_probe_spec(spec):
        """
        @type spec argo.template.v1.container.ContainerProbe
        """
        try:
            probe = swagger_client.V1Probe()
            probe.initial_delay_seconds = spec.initial_delay_seconds
            probe.timeout_seconds = spec.timeout_seconds
            probe.period_seconds = spec.period_seconds
            probe.failure_threshold = spec.failure_threshold
            probe.success_threshold = spec.success_threshold

            if spec.exec_probe:
                action = swagger_client.V1ExecAction()
                action.command = shlex.split(spec.exec_probe.command)
                probe._exec = action
                return probe
            elif spec.http_get:
                action = swagger_client.V1HTTPGetAction()
                action.path = spec.http_get.path
                action.port = spec.http_get.port
                headers = spec.http_get.http_headers
                action.http_headers = []
                for header in headers or []:
                    h = swagger_client.V1HTTPHeader()
                    h.name = header["name"]
                    h.value = header["value"]
                    action.http_headers.append(h)
                probe.http_get = action
                return probe
            else:
                logger.debug("Cannot handle probe {}".format(spec))
        except Exception as e:
            raise AXIllegalArgumentException("Probe {} cannot be processed due to error {}".format(spec, e))

        return None
