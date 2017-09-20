#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import json

from .base import BaseTemplate
from .common import Inputs, Outputs, DockerSpec
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

        # these fields need to be moved out
        self.once = True
        self.repo = None

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
            "repo": None,

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

    def to_dict(self):
        ret = super(ContainerTemplate, self).to_dict()
        if self.docker_spec:
            ret["annotations"]["ax_ea_docker_enable"] = json.dumps(self.docker_spec.to_dict())
        return ret

    def get_resources(self):
        return self.resources


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

