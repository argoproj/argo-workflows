#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from future.utils import iteritems

from argo.parser import ObjectParser
from argo.parser import ObjectDefinition as OD


class Parameter(ObjectParser):

    def __init__(self):

        self.description = None
        self.default = None
        self.options = []
        self.regex = None

        self._object_parser_fields = {
            "description": None,
            "default": None,
            "options": None,
            "regex": None
        }
        super(Parameter, self).__init__()


class InputArtifact(ObjectParser):

    def __init__(self):
        self.description = None
        self.path = None
        self.from_loc = None

        self._object_parser_fields = {
            "description": None,
            "path": None,
            "from": OD(field_name="from_loc")
        }
        super(InputArtifact, self).__init__()


class Inputs(ObjectParser):

    def __init__(self):
        self.parameters = {}
        self.artifacts = {}
        self.volumes = {}
        self.fixtures = {}

        self._object_parser_fields = {
            "parameters": None,
            "artifacts": None,
            "volumes": None,
            "fixtures": None,
        }
        super(Inputs, self).__init__()

    def parse(self, data, error_on_not_found=False):
        super(Inputs, self).parse(data, error_on_not_found=error_on_not_found)

        # This is a special implementation since each subtype needs object creation
        for param_key, param_val in iteritems(self.parameters):
            new_param_val = Parameter()
            new_param_val.parse(param_val)
            self.parameters[param_key] = new_param_val

        for artifact_key, artifact_val in iteritems(self.artifacts):
            new_artificat_val = InputArtifact()
            new_artificat_val.parse(artifact_val)
            self.artifacts[artifact_key] = new_artificat_val

        for vol_key, vol_val in iteritems(self.volumes):
            new_vol = VolumeInput()
            new_vol.parse(vol_val)
            self.volumes[vol_key] = new_vol

    def to_dict(self):
        ret = super(Inputs, self).to_dict()
        ret["parameters"] = {}
        ret["artifacts"] = {}
        ret["volumes"] = {}
        for param_key, param_val in iteritems(self.parameters):
            ret["parameters"][param_key] = param_val.to_dict()
        for artifact_key, artifact_val in iteritems(self.artifacts):
            ret["artifacts"][artifact_key] = artifact_val.to_dict()
        for vol_key, vol_val in iteritems(self.volumes):
            ret["volumes"][vol_key] = vol_val.to_dict()
        return ret

    def count(self):
        return len(self.parameters) + len(self.artifacts) + len(self.volumes) + len(self.fixtures)


class Outputs(ObjectParser):

    def __init__(self):
        self.artifacts = {}
        self.reporting_callback = None

        self._object_parser_fields = {
            "artifacts": None, # needs special handling due to k,v pairs
            "reporting_callback": OutputReportingCallback.fqcn()
        }
        super(Outputs, self).__init__()

    def parse(self, data, error_on_not_found=False):
        super(Outputs, self).parse(data, error_on_not_found=error_on_not_found)

        for k,v in iteritems(self.artifacts):
            new_v = OutputArtifact()
            new_v.parse(v, error_on_not_found=error_on_not_found)
            self.artifacts[k] = new_v

    def to_dict(self):
        ret = super(Outputs, self).to_dict()
        ret["artifacts"] = {}
        for k,v in iteritems(self.artifacts):
            ret["artifacts"][k] = v.to_dict()
        return ret

    def count(self):
        return len(self.artifacts) + 1 if self.reporting_callback else 0


class OutputArtifact(ObjectParser):

    def __init__(self):
        self.path = None
        self.excludes = []
        self.archive_mode = None
        self.storage_method = None
        self.retention = None
        self.aliases = None

        self._object_parser_fields = {
            "path": None,
            "excludes": None,
            "archive_mode": None,
            "storage_method": None,
            "from": None,
            "retention": None,
            "aliases": None
        }
        super(OutputArtifact, self).__init__()


class OutputReportingCallback(ObjectParser):

    def __init__(self):
        self.run_once = None
        self.is_wfe = None
        self.cookie = None
        self.uuid = None
        
        self._object_parser_fields = {
            "run_once": None,
            "is_wfe": None,
            "cookie": OutputReportingCallbackCookie.fqcn(),
            "uuid": None
        }
        super(OutputReportingCallback, self).__init__()


class OutputReportingCallbackCookie(ObjectParser):

    def __init__(self):
        self.instance_salt = None,
        self.start_timestamp = None
        
        self._object_parser_fields = {
            "instance_salt": None,
            "start_timestamp": None
        }
        super(OutputReportingCallbackCookie, self).__init__()
        

class Volume(ObjectParser):

    def __init__(self):
        self.name = None
        self.storage_class = None
        self.size_gb = None

        self._oject_parser_fields = {
            "name": None,
            "storage_class": None,
            "size_gb": None
        }
        super(Volume, self).__init__()


class VolumeInput(ObjectParser):

    def __init__(self):
        self.description = None
        self.from_src = None
        self.mount_path = None
        self.details = {}

        self._object_parser_fields = {
            "description": OD(),
            "from": OD(field_name="from_src"),
            "mount_path": OD(),
            "details": OD(),
        }
        super(VolumeInput, self).__init__()


class TerminationPolicy(ObjectParser):

    def __init__(self):
        self.spending_cents = None
        self.time_seconds = None
        
        self._object_parser_fields = {
            "spending_cents": None,
            "time_seconds": None
        }
        super(TerminationPolicy, self).__init__()


class VolumeDetails(Volume):
    """
    This class has additional details passed between argo
    microservices that are not part of yaml specification
    """
    def __init__(self):
        super(VolumeDetails, self).__init__()
        self.axrn = None
        self.name = None
        self.details = {}

        fields = {
            "name": None,
            "axrn": None,
            "details": None
        }

        self.set_fields(fields)


class GraphStorageSpec(ObjectParser):
    def __init__(self):
        self.graph_storage_size_mib = 0.0
        self._object_parser_fields = {
            "graph-storage-size": OD(field_name="graph_storage_size_mib", required=True)
        }
        super(GraphStorageSpec, self).__init__()

    def parse(self, data, error_on_not_found=False):
        super(GraphStorageSpec, self).parse(data, error_on_not_found=error_on_not_found)
        # special parsing for graph storage size
        if not isinstance(self.graph_storage_size_mib, int):
            self.graph_storage_size_mib = int(self.graph_storage_size_mib[:-2]) * 1024

    def to_dict(self):
        ret = super(GraphStorageSpec, self).to_dict()
        ret["graph-storage-size"] = "{}Gi".format(self.graph_storage_size_mib / 1024)
        return ret


class DockerSpec(GraphStorageSpec):
    def __init__(self):
        super(DockerSpec, self).__init__()
        self.cpu_cores = 0.0
        self.mem_mib = 0

        fields = {
            "cpu_cores": OD(required=True),
            "mem_mib": OD(required=True)
        }
        self.set_fields(fields)


class GraphStorageVolumeSpec(GraphStorageSpec):
    def __init__(self):
        self.mount_path = None
        super(GraphStorageVolumeSpec, self).__init__()
        fields = {
            "mount-path": OD(field_name="mount_path", required=True)
        }
        self.set_fields(fields)


class ExecutorSpec(ObjectParser):
    def __init__(self):
        self.disable = False
        self._object_parser_fields = {
            "disable" : None
        }
        super(ExecutorSpec, self).__init__()
