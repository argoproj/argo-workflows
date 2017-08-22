#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
This module represents the service object that is used to define
a unit of work in argo
"""

import json

from future.utils import iteritems

from argo.parser import ObjectParser
from argo.parser import ObjectDefinition as OD
from argo.template.v1.container import ContainerTemplate
from argo.template.v1.common import Volume
from argo.template.v1.deployment import DeploymentTemplate


class ServiceCommon(ObjectParser):

    def __init__(self):

        self.id = None
        self.name = None
        self.labels = []
        self.annotations = {}
        self.service_context = None
        self.costid = None

        self._object_parser_fields = {
            "id": OD(),
            "name": OD(),
            "labels": OD(),
            "annotations": OD(),
            "service_context": OD(class_string=ServiceContext.fqcn()),
            "costid": OD()
        }
        super(ServiceCommon, self).__init__()


class ServiceContext(ObjectParser):

    def __init__(self):
        self.root_workflow_id = None
        self.service_instance_id = None
        self.name = None
        self.leaf_name = None
        self.parent_service_instance_id = None
        self.leaf_full_path = None
        self.artifact_tags = []

        self._object_parser_fields = {
            "root_workflow_id": None,
            "service_instance_id": None,
            "name": None,
            "leaf_name": None,
            "parent_service_instance_id": None,
            "leaf_full_path": None,
            "artifact_tags": None
        }
        super(ServiceContext, self).__init__()


class Service(ServiceCommon):

    def __init__(self):
        super(Service, self).__init__()
        self.volumes = {}
        self.template = None

        fields = {
            "volumes": None, # handle volumes as it is k,v map
            "template": ContainerTemplate.fqcn()
        }
        self.set_fields(fields)

    def parse(self, data, error_on_not_found=False):
        super(Service, self).parse(data, error_on_not_found=error_on_not_found)

        for k,v in iteritems(self.volumes):
            new_v = Volume()
            new_v.parse(v, error_on_not_found=error_on_not_found)
            self.volumes[k] = new_v

    def to_dict(self):
        ret = super(Service, self).to_dict()

        ret["volumes"] = {}
        for k,v in iteritems(self.volumes):
            if isinstance(v, Volume):
                ret["volumes"][k] = v.to_dict()
        return ret


class DeploymentService(ServiceCommon):

    def __init__(self):
        super(DeploymentService, self).__init__()
        self.app_generation = None
        self.deployment_id = None
        self.app_id = None
        self.template = None

        fields = {
            "app_generation": None,
            "deployment_id": None,
            "app_id": None,
            "template": DeploymentTemplate.fqcn()
        }
        self.set_fields(fields)
