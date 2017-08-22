#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import re

from future.utils import iteritems

from .base import BaseTemplate
from .common import Inputs, TerminationPolicy
from .container import ContainerTemplate
from argo.parser import ObjectParser
from argo.parser import ObjectDefinition as OD
from argo.template.v1.common import VolumeDetails


class DeploymentTemplate(BaseTemplate):

    """
    :type inputs: Inputs
    :type scale: Scale
    """

    def __init__(self):
        super(DeploymentTemplate, self).__init__()
        self.inputs = Inputs()
        self.application_name = None
        self.deployment_name = None
        self.scale = Scale()
        self.external_routes = []
        self.internal_routes = []
        self.containers = {}
        self.fixtures = []
        self.volumes = {}
        self.termination_policy = None
        self.min_ready_seconds = None
        self.strategy = Strategy()

        fields = {
            "inputs": Inputs.fqcn(),
            "application_name": None,
            "deployment_name": None,
            "scale": Scale.fqcn(),
            "external_routes": ExternalRoute.fqcn(),
            "internal_routes": InternalRoute.fqcn(),
            "containers": None, # needs special handling later
            "fixtures": None,
            "volumes": None, # needs special handling later
            "termination_policy": TerminationPolicy.fqcn(),
            "min_ready_seconds": None,
            "strategy": Strategy.fqcn()
        }

        self._object_parser_fields.update(fields)

    def parse(self, data, error_on_not_found=False):
        super(DeploymentTemplate, self).parse(data, error_on_not_found=error_on_not_found)

        # now handle containers
        for k,v in iteritems(self.containers):
            new_container = ContainerTemplate()
            if "template" in v:
                # found the service object not template
                new_container.parse(v["template"], error_on_not_found=error_on_not_found)
            else:
                new_container.parse(v, error_on_not_found=error_on_not_found)
            self.containers[k] = new_container

        # now handle volumes
        for k,v in iteritems(self.volumes):
            new_vol = VolumeDetails()
            new_vol.parse(v, error_on_not_found=error_on_not_found)
            self.volumes[k] = new_vol

    def to_dict(self):
        ret = super(DeploymentTemplate, self).to_dict()

        # handle special cases now
        ret["containers"] = {}
        for k,v in iteritems(self.containers):
            ret["containers"][k] = v.to_dict()

        ret["volumes"] = {}
        for k,v in iteritems(self.volumes):
            ret["volumes"][k] = v.to_dict()

        return ret

    def get_main_container(self):
        for name, template in iteritems(self.containers):
            if template.name is None:
                template.name = name
            return template
        raise ValueError("There needs to be at least one container defined in the specification")


class Scale(ObjectParser):

    def __init__(self):
        self.min = 1
        self.max = None

        self._object_parser_fields = {
            "min": None,
            "max": None
        }
        super(Scale, self).__init__()


class ExternalRoute(ObjectParser):

    def __init__(self):
        self.name = None
        self.dns_name = None
        self.dns_prefix = None
        self.dns_domain = None
        self.target_port = None
        self.ip_white_list = []
        self.visibility = None

        self._object_parser_fields = {
            "name": None,
            "dns_name": DnsDomain.fqcn(),
            "dns_prefix": DnsLabel.fqcn(),
            "dns_domain": DnsDomain.fqcn(),
            "target_port": None,
            "ip_white_list": None,
            "visibility": None
        }
        super(ExternalRoute, self).__init__()


class DnsLabel(ObjectParser):
    """
    This Subclass of ObjectParser parses dns label and matches them with
    regex to ensure that it is a valid dns label
    """

    def __init__(self):
        self.dns_label = None
        # don't need _object_parser_fields as it has no keys that it needs to parse
        super(DnsLabel, self).__init__()

    def parse(self, data, error_on_not_found=False):
        """
        regex is [a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?
        """
        matched = re.match(r"^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$", data.lower())
        if not matched:
            raise ValueError("{} is not a valid Dns Label".format(data))
        self.dns_label = data.lower()

    def __call__(self, *args, **kwargs):
        return self.dns_label

    def to_dict(self):
        return self.dns_label


class DnsDomain(ObjectParser):

    def __init__(self):
        self.dns_domain = None
        super(DnsDomain, self).__init__()
    
    def parse(self, data, error_on_not_found=False):
        labels = data.lower().split(".")
        for label in labels:
            if len(label) == 0:
                continue
            matched = re.match(r"^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$", label)
            if not matched:
                raise ValueError("Dns Domain {} is not valid as the label {} did not match requirements".format(label, data.lower()))
        
        self.dns_domain = data.lower()
    
    def __call__(self, *args, **kwargs):
        return self.dns_domain

    def to_dict(self):
        return self.dns_domain


class InternalRoute(ObjectParser):

    def __init__(self):
        self.name = None
        self.ports = []

        self._object_parser_fields = {
            "name": None,
            "ports": Port.fqcn()
        }
        super(InternalRoute, self).__init__()


class Port(ObjectParser):

    def __init__(self):
        self.port = None
        self.target_port = None

        self._object_parser_fields = {
            "port": None,
            "target_port": None
        }
        super(Port, self).__init__()


class Strategy(ObjectParser):

    def __init__(self):
        self.type = "recreate"
        self.rolling_update = RollingUpdateStrategy()

        self._object_parser_fields = {
            "type": None,
            "rolling_update": RollingUpdateStrategy.fqcn()
        }
        super(Strategy, self).__init__()


class RollingUpdateStrategy(ObjectParser):

    def __init__(self):
        self.max_surge = 1
        self.max_unavailable = 1

        self._object_parser_fields = {
            "max_surge": OD(converter_tuple=(int, str)),
            "max_unavailable": OD(converter_tuple=(int, str))
        }
        super(RollingUpdateStrategy, self).__init__()
