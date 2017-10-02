# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
import copy
import os

from ax.cloud import Cloud
from ax.meta import AXClusterId, AXCustomerId
from ax.platform.container import Container, ContainerVolume
from ax.platform.component_config import SoftwareInfo
from ax.platform.secrets import SecretResource

AX_DOCKER_GRAPH_STORAGE_THRESHOLD_DEFAULT = "0.5"

SIDEKICK_WAIT_CONTAINER_NAME = "axsidekickwait"
INIT_CONTAINER_NAME_ARTIFACTS = "axinit"
INIT_CONTAINER_NAME_PULLIMAGE = "axpull"
DIND_CONTAINER_NAME = "axdindhelper"
WFE_CONTAINER_NAME = "axworkflowexecutor"


def is_ax_aux_container(container_name):
    return container_name in [WFE_CONTAINER_NAME, SIDEKICK_WAIT_CONTAINER_NAME,
                              INIT_CONTAINER_NAME_ARTIFACTS, INIT_CONTAINER_NAME_PULLIMAGE,
                              DIND_CONTAINER_NAME]


class ArtifactsContainer(Container):
    """
    This container defines the volumes and environments needed for artifacts management
    """

    ARTIFACTS_CONTAINER_SCRATCH = "/ax-artifacts-scratch"

    def __init__(self, containername, customer_image, namespace, version):
        s = SoftwareInfo()
        super(ArtifactsContainer, self).__init__(
            containername, "{}/{}/artifacts:{}".format(s.registry, namespace, version)
        )

        # artifacts scratch space
        self._artifacts_scratch = ContainerVolume("artifacts-scratch", ArtifactsContainer.ARTIFACTS_CONTAINER_SCRATCH)
        self._artifacts_scratch.set_type("EMPTYDIR")
        self.add_volume(self._artifacts_scratch)

        # create a hostpath for docker-socket-dir. This is used to for running docker inspect
        socket_hostpath = ContainerVolume("docker-socket-file", "/var/run/docker.sock")
        socket_hostpath.set_type("HOSTPATH", "/var/run/docker.sock")
        self.add_volume(socket_hostpath)

        # emptydir for sharing for copying static binaries from init container
        # so that they are available in the main container
        self._static_bins = ContainerVolume("static-bins", "/copyto")
        self._static_bins.set_type("EMPTYDIR")
        self.add_volume(self._static_bins)

        # add environment vars needed for artifacts
        self.add_env("AX_TARGET_CLOUD", value=Cloud().target_cloud())
        self.add_env("AX_CLUSTER_NAME_ID", value=AXClusterId().get_cluster_name_id())
        self.add_env("AX_CUSTOMER_ID", value=AXCustomerId().get_customer_id())
        self.add_env("AX_CUSTOMER_IMAGE_NAME", value=customer_image)
        self.add_env("AX_ARTIFACTS_SCRATCH", value=ArtifactsContainer.ARTIFACTS_CONTAINER_SCRATCH)
        self.add_env("AX_POD_NAME", value_from="metadata.name")
        self.add_env("AX_POD_IP", value_from="status.podIP")
        self.add_env("AX_POD_NAMESPACE", value_from="metadata.namespace")
        self.add_env("AX_NODE_NAME", value_from="spec.nodeName")
        self.add_env("ARGO_LOG_BUCKET_NAME", os.getenv("ARGO_LOG_BUCKET_NAME", ""))
        self.add_env("ARGO_DATA_BUCKET_NAME", os.getenv("ARGO_DATA_BUCKET_NAME", ""))

        annotation_vol = ContainerVolume("annotations", "/etc/axspec")
        annotation_vol.set_type("DOWNWARDAPI", "metadata.annotations")
        self.add_volume(annotation_vol)

        # AA-3175: CPU and memory are set to lowest possible so that pod requests are kept at a minimum
        self.add_resource_constraints("cpu_cores", 0.001)
        self.add_resource_constraints("mem_mib", 4)

    def get_artifacts_volume(self):
        return copy.deepcopy(self._artifacts_scratch)

    def get_static_bins_volume(self):
        return copy.deepcopy(self._static_bins)


class WaitContainer(ArtifactsContainer):

    """
    The wait container adds volumes and environments needed for monitoring main container
    """

    def __init__(self, containername, customer_image, namespace, version):
        super(WaitContainer, self).__init__(containername, customer_image, namespace, version)


class SidecarTask(WaitContainer):
    """
    Class for sidecar of task
    """
    def __init__(self, customer_image, namespace, version):
        super(SidecarTask, self).__init__(
            SIDEKICK_WAIT_CONTAINER_NAME, customer_image, namespace, version
        )

        # Sidecar needs to manage logs so add the log path here
        logpath = ContainerVolume("containerlogs", "/logs")
        logpath.set_type("HOSTPATH", "/var/lib/docker/containers")
        self.add_volume(logpath)
        self.add_env("LOGMOUNT_PATH", "/logs")
        self.add_env("AX_CLUSTER_NAME_ID", os.getenv("AX_CLUSTER_NAME_ID"))

        # set the arguments
        self.args = ["post"]


class SidecarDeployment(Container):
    """
    Class for sidecar of deployment
    """
    def __init__(self):
        super(SidecarDeployment, self).__init__(
            SIDEKICK_WAIT_CONTAINER_NAME, "ubuntu:latest"
        )
        self.command = ["/bin/bash", "-c", "tail -f /dev/null"]


class InitContainerTask(ArtifactsContainer):
    def __init__(self, customer_image, namespace, version):
        super(InitContainerTask, self).__init__(
            INIT_CONTAINER_NAME_ARTIFACTS, customer_image, namespace, version
        )

        # set the arguments
        self.args = ["pre"]

    def add_configs_as_vols(self, configs, step_name, step_ns):
        """
        Some configs will be passed as secrets and these need to be loaded
        into the init container so that init container can convert the params
        in command and args to the necessary secret
        :param configs: A list of tuples of (config_namespace, config_name)
        """
        res_list = []
        for (cfg_ns, cfg_name) in configs or []:
            res = SecretResource(cfg_ns, cfg_name, step_name, step_ns)
            res.create()
            vol = ContainerVolume(res.get_resource_name(), "/ax_secrets/{}/{}".format(cfg_ns, cfg_name))
            vol.set_type("SECRET", res.get_resource_name())
            self.add_volume(vol)
            res_list.append(res)
        return res_list


class InitContainerSetup(Container):

    def __init__(self):
        super(InitContainerSetup, self).__init__("axsetup", "busybox:1.27.2-musl")
        c = ContainerVolume("static-bin-share", "/copyout")
        c.set_type("EMPTYDIR")
        self.add_volume(c)
        self.command = ["/bin/cp", "-f", "/bin/true", "/copyout"]


class InitContainerPullImage(Container):

    def __init__(self, customer_image):
        super(InitContainerPullImage, self).__init__(INIT_CONTAINER_NAME_PULLIMAGE, customer_image)
        c = ContainerVolume("static-bin-share", "/staticbin")
        c.set_type("EMPTYDIR")
        self.add_volume(c)
        self.command = ["/staticbin/true"]

        # AA-3175: CPU and memory are set to lowest possible so that pod requests are kept at a minimum
        self.add_resource_constraints("cpu_cores", 0.001)
        self.add_resource_constraints("mem_mib", 4)


class SidecarDockerDaemon(Container):
    """
    Spec for dind daemon
    """
    def __init__(self, size_in_mb):
        super(SidecarDockerDaemon, self).__init__(DIND_CONTAINER_NAME, "argoproj/dind:1.12.6")

        # Add lib modules for dind to load aufs module.
        libmodule_hostpath = ContainerVolume("kernel-lib-module", "/lib/modules")
        libmodule_hostpath.set_type("HOSTPATH", "/lib/modules")
        self.add_volume(libmodule_hostpath)

        # Add per node dgs to sidecar
        dgs_vol = ContainerVolume("docker-graph-storage", "/var/lib/docker")
        if Cloud().target_cloud_aws():
            dgs_vol.set_type("DOCKERGRAPHSTORAGE", size_in_mb)
        elif Cloud().target_cloud_gcp():
            dgs_vol.set_type("EMPTYDIR")
        self.add_volume(dgs_vol)

        # dind daemon needs to be privileged!
        self.privileged = True
