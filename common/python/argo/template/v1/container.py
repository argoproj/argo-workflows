#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import json
import re

from .base import BaseTemplate
from .common import Inputs, Outputs, DockerSpec, ExecutorSpec, GraphStorageVolumeSpec
from argo.parser.parser import ObjectParser, ObjectDefinition as OD

from ax.platform.annotations import Annotations


class EnvVar(ObjectParser):

    def __init__(self):
        self.name = None
        self.value = ""         # set default value to empty string

        self._object_parser_fields = {
            "name": OD(required=True),
            "value": OD()
        }
        super(EnvVar, self).__init__()

    def get_config(self):
        secret = re.match(r"^%%config\.(.+?)\.([A-Za-z0-9-]+)\.([A-Za-z0-9-]+)%%$", self.value)
        if secret:
            cfg_ns = secret.group(1)
            cfg_name = secret.group(2)
            cfg_key = secret.group(3)
            return cfg_ns, cfg_name, cfg_key
        else:
            return None, None, None


class ContainerTemplate(BaseTemplate):

    def __init__(self):
        super(ContainerTemplate, self).__init__()
        self.inputs = Inputs()
        self.outputs = Outputs()
        self.resources = ContainerResources()
        self.image = None
        self.command = None
        self.args = None
        self.env = []
        self.liveness_probe = None
        self.readiness_probe = None
        self.image_pull_policy = None
        self.name = None
        self.annotations = {}

        # derived field from annotations
        self.docker_spec = None
        self.executor_spec = None
        self.privileged = False
        self.graph_storage = None
        self.hostname = None

        # these fields need to be moved out
        self.once = True

        fields = {
            "inputs": Inputs.fqcn(),
            "outputs": Outputs.fqcn(),
            "resources": OD(class_string=ContainerResources.fqcn(), required=True),
            "image": OD(required=True),
            "command": None,
            "args": None,
            "env": OD(class_string=EnvVar.fqcn()),
            "liveness_probe": ContainerProbe.fqcn(),
            "readiness_probe": ContainerProbe.fqcn(),
            "image_pull_policy": None,
            "name": None,
            "annotations": None,

            # these fields need to be moved out
            "once": None
        }

        self.set_fields(fields)

    def parse(self, data, error_on_not_found=False):
        super(ContainerTemplate, self).parse(data, error_on_not_found=error_on_not_found)

        annotations_obj = Annotations()
        dspec = self.annotations.get("ax_ea_docker_enable", None)
        if dspec is not None:
            # load dict from json, validate it and then conver to DockerSpec
            dspec = json.loads(dspec)
            annotations_obj.parse("ax_ea_docker_enable", dspec)
            self.docker_spec = DockerSpec()
            self.docker_spec.parse(dspec)

        espec = self.annotations.get("ax_ea_executor", None)
        if espec is not None:
            espec = json.loads(espec)
            annotations_obj.parse("ax_ea_executor", espec)
            self.executor_spec = ExecutorSpec()
            self.executor_spec.parse(espec)

        priv_bool = self.annotations.get("ax_ea_privileged", None)
        if priv_bool is not None:
            priv_bool = json.loads(priv_bool)
            annotations_obj.parse("ax_ea_privileged", priv_bool)
            self.privileged = priv_bool

        gs_spec = self.annotations.get("ax_ea_graph_storage_volume", None)
        if gs_spec:
            gs_spec = json.loads(gs_spec)
            annotations_obj.parse("ax_ea_graph_storage_volume", gs_spec)
            self.graph_storage = GraphStorageVolumeSpec()
            self.graph_storage.parse(gs_spec)

        hostname_spec = self.annotations.get("ax_ea_hostname", None)
        if hostname_spec:
            annotations_obj.parse("ax_ea_hostname", hostname_spec)
            self.hostname = hostname_spec

    def to_dict(self):
        ret = super(ContainerTemplate, self).to_dict()
        if self.docker_spec:
            ret["annotations"]["ax_ea_docker_enable"] = json.dumps(self.docker_spec.to_dict())

        if self.executor_spec:
            ret["annotations"]["ax_ea_executor"] = json.dumps(self.executor_spec.to_dict())

        if self.privileged:
            ret["annotations"]["ax_ea_privileged"] = json.dumps(self.privileged)

        if self.graph_storage:
            ret["annotations"]["ax_ea_graph_storage_volume"] = json.dumps(self.graph_storage.to_dict())

        if self.hostname:
            ret["annotations"]["ax_ea_hostname"] = self.hostname

        return ret

    def get_resources(self):
        return self.resources

    def get_all_configs(self):
        configs = []
        for cmd in self.command or []:
            matches = re.findall(r"%%config\.(.+?)\.([A-Za-z0-9-]+)\.[A-Za-z0-9-]+%%", cmd)
            configs.extend(matches)
        for arg in self.args or []:
            matches = re.findall(r"%%config\.(.+?)\.([A-Za-z0-9-]+)\.[A-Za-z0-9-]+%%", arg)
            configs.extend(matches)
        return configs


class ContainerResources(ObjectParser):

    def __init__(self):
        self.mem_mib = 0.0
        self.cpu_cores = 0.0
        self._object_parser_fields = {
            "mem_mib": OD(required=True),
            "cpu_cores": OD(required=True)
        }
        super(ContainerResources, self).__init__()


class ContainerProbe(ObjectParser):
    """
    :type exec_probe: ContainerProbeExec
    :type http_get: ContainerProbeHttpRequest
    """

    def __init__(self):
        self.initial_delay_seconds = None
        self.timeout_seconds = None
        self.period_seconds = None
        self.failure_threshold = None
        self.success_threshold = None
        self.exec_probe = None
        self.http_get = None

        self._object_parser_fields = {
            "initial_delay_seconds": None,
            "timeout_seconds": None,
            "period_seconds": None,
            "failure_threshold": None,
            "success_threshold": None,
            "exec": (ContainerProbeExec.fqcn(), "exec_probe"),
            "http_get": ContainerProbeHttpRequest.fqcn(),
        }

        super(ContainerProbe, self).__init__()


class ContainerProbeExec(ObjectParser):

    def __init__(self):
        self.command = None
        self._object_parser_fields = {
            "command": None
        }
        super(ContainerProbeExec, self).__init__()


class ContainerProbeHttpRequest(ObjectParser):

    def __init__(self):
        self.path = None
        self.port = None
        self.http_headers = []

        self._object_parser_fields = {
            "path": None,
            "port": None,
            "http_headers": None
        }
        super(ContainerProbeHttpRequest, self).__init__()

