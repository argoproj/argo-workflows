#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
This object dynamically updates Argo kubernetes yaml file
"""

import logging
from math import ceil
import os

from ax.cloud import Cloud
from ax.meta import AXClusterId, AXCustomerId
from ax.kubernetes.ax_kube_dict import KubeKindToV1KubeSwaggerObject
from ax.kubernetes.swagger_client import ApiClient, V1Pod, V1beta1StatefulSet, V1beta1Deployment, V1beta1DaemonSet, \
    V1PersistentVolumeClaim, V1Container, V1Service, V1EnvVar, V1EnvVarSource, V1ObjectFieldSelector
from ax.platform.cluster_config import AXClusterConfig, ClusterProvider
from ax.platform.component_config import SoftwareInfo
from ax.platform.resource import AXSYSResourceConfig
from ax.util.resource import ResourceValueConverter
import yaml


logger = logging.getLogger(__name__)


class AXSYSKubeYamlUpdater(object):
    """
    This class loads a kubernetes yaml file, updates resource,
    and generate objects that kube_object.py can consume
    """
    def __init__(self, config_file_path):
        assert os.path.isfile(config_file_path), "Config file {} is not a file".format(config_file_path)
        self._config_file = config_file_path
        self._cluster_name_id = AXClusterId().get_cluster_name_id()
        self._cluster_config = AXClusterConfig(cluster_name_id=self._cluster_name_id)
        if self._cluster_config.get_cluster_provider() != ClusterProvider.USER:
            self.cpu_mult, self.mem_mult, self.disk_mult, \
                self.daemon_cpu_mult, self.daemon_mem_mult = self._get_resource_multipliers()
        else:
            self.cpu_mult = 1
            self.mem_mult = 1
            self.disk_mult = 1
            self.daemon_cpu_mult = 1
            self.daemon_mem_mult = 1
        self._swagger_components = []
        self._yaml_components = []
        self._updated_raw = ""

        # TODO: when we support config software info using a config file, need to figure out how that
        # file gets passed through, since SoftwareInfo is not a singleton
        self._software_info = SoftwareInfo()

        self._load_objects()
        self._load_raw()

    @property
    def updated_raw(self):
        return self._updated_raw

    @property
    def components_in_dict(self):
        return self._yaml_components

    @property
    def components_in_swagger(self):
        return self._swagger_components

    def _load_objects(self):
        with open(self._config_file, "r") as f:
            data = f.read()
        for c in yaml.load_all(data):
            swagger_obj = self._config_yaml(c)
            yaml_obj = ApiClient().sanitize_for_serialization(swagger_obj)
            self._swagger_components.append(swagger_obj)
            self._yaml_components.append(yaml_obj)

    def _load_raw(self):
        self._updated_raw = yaml.dump_all(self._yaml_components)

    def _get_resource_multipliers(self):
        """
        Resources in yaml templates need to be multiplied with these numbers
        :return: cpu_multiplier, mem_multiplier, disk_multiplier
        """
        # Getting cluster size from cluster config, in order to configure resources
        # There are 3 situations we will be using AXClusterConfig
        #   - During install, since the class is a singleton, it has all the values we need
        #     no need to download from s3
        #   - During upgrade, since we are exporting AWS_DEFAULT_PROFILE, we can download
        #     cluster config files from s3 to get the values
        #   - During job creation: the node axmon runs has the proper roles to access s3
        
        try:
            ax_node_max = int(self._cluster_config.get_asxys_node_count())
            ax_node_type = self._cluster_config.get_axsys_node_type()
            usr_node_max = int(self._cluster_config.get_max_node_count()) - ax_node_max
            usr_node_type = self._cluster_config.get_axuser_node_type()
            assert all([ax_node_max, ax_node_type, usr_node_max, usr_node_type])
        except Exception as e:
            logger.error("Unable to read cluster config, skip resource config for %s. Error %s", self._config_file, e)
            return 1, 1, 1, 1, 1

        rc = AXSYSResourceConfig(ax_node_type=ax_node_type,
                                 ax_node_max=ax_node_max,
                                 usr_node_type=usr_node_type,
                                 usr_node_max=usr_node_max,
                                 cluster_type=self._cluster_config.get_ax_cluster_type())
        #logger.info("With %s %s axsys nodes, %s %s axuser nodes, component %s uses multipliers (%s, %s, %s, %s, %s)",
        #            ax_node_max, ax_node_type, usr_node_max, usr_node_type, self._config_file,
        #            rc.cpu_multiplier, rc.mem_multiplier, rc.disk_multiplier,
        #            rc.daemon_cpu_multiplier, rc.daemon_mem_multiplier)
        return rc.cpu_multiplier, rc.mem_multiplier, rc.disk_multiplier, rc.daemon_cpu_multiplier, rc.daemon_mem_multiplier

    def _config_yaml(self, kube_yaml_obj):
        """
        Load dict into swagger object, patch resource,
        sanitize, return a dict
        :param kube_yaml_obj:
        :return: swagger object with resource values finalized
        """
        kube_kind = kube_yaml_obj["kind"]
        (swagger_class_literal, swagger_instance) = KubeKindToV1KubeSwaggerObject[kube_kind]
        swagger_obj = ApiClient()._ApiClient__deserialize(kube_yaml_obj, swagger_class_literal)
        assert isinstance(swagger_obj, swagger_instance), \
            "{} has instance {}, expected {}".format(swagger_obj, type(swagger_obj), swagger_instance)

        if isinstance(swagger_obj, V1beta1Deployment):
            if not self._software_info.registry_is_private():
                swagger_obj.spec.template.spec.image_pull_secrets = None

            node_selector = swagger_obj.spec.template.spec.node_selector
            if node_selector and node_selector.get('ax.tier', 'applatix') == 'master':
                # Skip updating containers on master.
                logger.info("Skip updating cpu, mem multipliers for pods on master: %s", swagger_obj.metadata.name)
            else:
                for container in swagger_obj.spec.template.spec.containers:
                    self._update_container(container)
            return swagger_obj
        elif isinstance(swagger_obj, V1Pod):
            if not self._software_info.registry_is_private():
                swagger_obj.spec.image_pull_secrets = None
            return swagger_obj
        elif isinstance(swagger_obj, V1beta1DaemonSet):
            if not self._software_info.registry_is_private():
                swagger_obj.spec.template.spec.image_pull_secrets = None
            for container in swagger_obj.spec.template.spec.containers:
                # We are special-casing applet DaemonSet to compromise the fact that
                # we are using different node type for compute-intense nodes
                if swagger_obj.metadata.name == "applet":
                    self._update_container(container=container, is_daemon=True, update_resource=True)
                else:
                    self._update_container(container=container, is_daemon=True, update_resource=False)
            return swagger_obj
        elif isinstance(swagger_obj, V1beta1StatefulSet):
            if not self._software_info.registry_is_private():
                swagger_obj.spec.template.spec.image_pull_secrets = None
            return self._update_statefulset(swagger_obj)
        elif isinstance(swagger_obj, V1PersistentVolumeClaim):
            self._update_volume(swagger_obj)
            return swagger_obj
        else:
            # logger.info("Object %s does not need to configure resource", type(swagger_obj))
            # HACK, as the original hook will be messed up
            if isinstance(swagger_obj, V1Service):
                if swagger_obj.metadata.name == "axops":
                    swagger_obj.spec.load_balancer_source_ranges = []
                    if self._cluster_config and self._cluster_config.get_trusted_cidr():
                        for cidr in self._cluster_config.get_trusted_cidr():
                            # Seems swagger client does not support unicode ... SIGH
                            swagger_obj.spec.load_balancer_source_ranges.append(str(cidr))

                # HACK #2: if we don't do this, kubectl will complain about something such as
                #
                # spec.ports[0].targetPort: Invalid value: "81": must contain at least one letter (a-z)
                #
                # p.target_port is defined as string though, but if its really a string, kubectl
                # is looking for a port name, rather than a number
                # SIGH ...
                for p in swagger_obj.spec.ports or []:
                    try:
                        p.target_port = int(p.target_port)
                    except (ValueError, TypeError):
                        pass
            return swagger_obj

    def _update_deployment_or_daemonset(self, kube_obj):
        assert isinstance(kube_obj, V1beta1Deployment) or isinstance(kube_obj, V1beta1DaemonSet)
        for container in kube_obj.spec.template.spec.containers:
            self._update_container(container)
        return kube_obj

    def _update_statefulset(self, kube_obj):
        assert isinstance(kube_obj, V1beta1StatefulSet)
        for container in kube_obj.spec.template.spec.containers:
            self._update_container(container)
        if isinstance(kube_obj.spec.volume_claim_templates, list):
            for vol in kube_obj.spec.volume_claim_templates:
                self._update_volume(vol)
        return kube_obj

    def _update_container(self, container, is_daemon=False, update_resource=True):
        assert isinstance(container, V1Container)

        if update_resource:
            cpulim = container.resources.limits.get("cpu")
            memlim = container.resources.limits.get("memory")
            cpureq = container.resources.requests.get("cpu")
            memreq = container.resources.requests.get("memory")

            def _massage_cpu(orig):
                return orig * self.daemon_cpu_mult if is_daemon else orig * self.cpu_mult

            def _massage_mem(orig):
                return orig * self.daemon_mem_mult if is_daemon else orig * self.mem_mult

            if cpulim:
                rvc = ResourceValueConverter(value=cpulim, target="cpu")
                rvc.massage(_massage_cpu)
                container.resources.limits["cpu"] = "{}m".format(rvc.convert("m"))
            if cpureq:
                rvc = ResourceValueConverter(value=cpureq, target="cpu")
                rvc.massage(_massage_cpu)
                container.resources.requests["cpu"] = "{}m".format(rvc.convert("m"))
            if memlim:
                rvc = ResourceValueConverter(value=memlim, target="memory")
                rvc.massage(_massage_mem)
                container.resources.limits["memory"] = "{}Mi".format(int(rvc.convert("Mi")))
            if memreq:
                rvc = ResourceValueConverter(value=memreq, target="memory")
                rvc.massage(_massage_mem)
                container.resources.requests["memory"] = "{}Mi".format(int(rvc.convert("Mi")))

        if container.liveness_probe and container.liveness_probe.http_get:
            try:
                container.liveness_probe.http_get.port = int(container.liveness_probe.http_get.port)
            except (ValueError, TypeError):
                pass
        if container.readiness_probe and container.readiness_probe.http_get:
            try:
                container.readiness_probe.http_get.port = int(container.readiness_probe.http_get.port)
            except (ValueError, TypeError):
                pass

        # Add resource multiplier to containers in case we need them
        if not container.env:
            container.env = []
        container.env += self._generate_default_envs(is_daemon, update_resource)

    def _update_volume(self, vol):
        assert isinstance(vol, V1PersistentVolumeClaim)
        vol_size = vol.spec.resources.requests["storage"]

        def _massage_disk(orig):
            return orig * self.disk_mult

        if vol_size:
            rvc = ResourceValueConverter(value=vol_size, target="storage")
            rvc.massage(_massage_disk)
            # Since AWS does not support value such as 1.5G, lets round up to its ceil
            vol.spec.resources.requests["storage"] = "{}Gi".format(int(ceil(rvc.convert("Gi"))))

        # Manually patch access mode as swagger client mistakenly interprets this as map
        vol.spec.access_modes = ["ReadWriteOnce"]

    def _generate_default_envs(self, is_daemon, resource_updated):
        """
        Add essential variables to all system containers
        :param is_daemon:
        :return:
        """
        default_envs = [
            # Kubernetes downward APIs
            {"name": "AX_NODE_NAME", "path": "spec.nodeName"},
            {"name": "AX_POD_NAME", "path": "metadata.name"},
            {"name": "AX_POD_NAMESPACE", "path": "metadata.namespace"},
            {"name": "AX_POD_IP", "path": "status.podIP"},

            # Values
            {"name": "DISK_MULT", "value": str(self.disk_mult)},
            {"name": "AX_TARGET_CLOUD", "value": Cloud().target_cloud()},
            {"name": "AX_CLUSTER_NAME_ID", "value": self._cluster_name_id},
            {"name": "AX_CUSTOMER_ID", "value": AXCustomerId().get_customer_id()},
            {"name": "AX_AWS_REGION", "value": os.environ.get("AX_AWS_REGION", None)},
        ]

        # Special cases for daemons
        if is_daemon:
            if resource_updated:
                default_envs += [
                    {"name": "CPU_MULT", "value": str(self.daemon_cpu_mult)},
                    {"name": "MEM_MULT", "value": str(self.daemon_mem_mult)},
                ]
            else:
                default_envs += [
                    {"name": "CPU_MULT", "value": "1.0"},
                    {"name": "MEM_MULT", "value": "1.0"},
                ]
        else:
            default_envs += [
                {"name": "CPU_MULT", "value": str(self.cpu_mult)},
                {"name": "MEM_MULT", "value": str(self.mem_mult)},
            ]


        rst = []
        for d in default_envs:
            var = V1EnvVar()
            var.name = d["name"]

            if d.get("path", None):
                field = V1ObjectFieldSelector()
                field.field_path = d["path"]
                src = V1EnvVarSource()
                src.field_ref = field
                var.value_from = src
            else:
                var.value = d["value"]
            rst.append(var)
        return rst
