# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
from future.utils import with_metaclass
from jsonschema import validate, ValidationError

from ax.exceptions import AXIllegalArgumentException
from ax.util.singleton import Singleton


class Annotations(with_metaclass(Singleton, object)):

    ax_ea_docker_enable = {
        "$schema": "http://json-schema.org/schema#",
        "title": "Validation schema for enabling docker",
        "type": "object",
        "properties": {
            "graph-storage-name": {
                "type": "string",
                "minLength": 5,
                "maxLength": 233
            },
            "graph-storage-size": {
                "type": "string",
                "pattern": "^[0-9]+Gi$"
            },
            "cpu_cores": {
                "type": "number",
                "minimum": 0,
                "exclusiveMinimum": True
            },
            "mem_mib": {
                "type": "integer",
                "minimum": 32
            }

        },
        "required": ["graph-storage-size", "mem_mib", "cpu_cores"]
    }

    def __init__(self):
        self._annotations = {}

        for ax_ea_annotation in self.__class__.__dict__:
            if ax_ea_annotation.startswith("ax_ea_"):
                self.register(ax_ea_annotation, self.__class__.__dict__[ax_ea_annotation])

    def register(self, annotation, schema):
        self._annotations[annotation] = schema

    def parse(self, annotation, data):
        try:
            schema = self._annotations.get(annotation, None)
            if not schema:
                raise AXIllegalArgumentException("annotation {} is not supported".format(annotation))
            validate(data, schema)
        except ValidationError as e:
            raise AXIllegalArgumentException(e.message, detail=e)
