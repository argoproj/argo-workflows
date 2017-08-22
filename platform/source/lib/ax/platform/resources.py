#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import json
import logging
import importlib

from datetime import datetime
from abc import ABCMeta, abstractmethod
from future.utils import with_metaclass

from ax.kubernetes import swagger_client

"""
This module handles the interfaces and implementation of additional resources that
are created as part of a deployment or task or application.
The responsibilities of this module include marshalling and unmarshalling these resources
from the kubernetes object and supporting a unified interface for creating, modifying and
deleting these resources.
"""

logger = logging.getLogger(__name__)


class AXResource(with_metaclass(ABCMeta, object)):
    """
    This class is an interface that should be implemented by all classes
    that need to create additional resources.
    """
    __metaclass__ = ABCMeta

    @staticmethod
    @abstractmethod
    def create_object_from_info(info):
        pass

    @abstractmethod
    def get_resource_name(self):
        """
        This function returns the name of the resource as a string
        Returns: string
        """
        pass

    @abstractmethod
    def get_resource_info(self):
        """
        This function should return a dictionary that needs to be stored
        in the kubernetes object
        Returns: dict
        """
        pass

    def get_resource_type(self):
        """
        Return the type of object as a string
        Returns: string
        """
        return "{}.{}".format(type(self).__module__,type(self).__name__)

    @abstractmethod
    def delete(self):
        """
        Delete the object in kubernetes
        """
        pass

    @abstractmethod
    def status(self):
        """
        This method gets the status of the object in kubernetes and returns
        a dictionary of values
        Returns: dict
        """
        pass

    @staticmethod
    def get_ax_meta(obj):
        if obj.metadata is None or obj.metadata.annotations is None:
            return None
        meta = obj.metadata.annotations.get("ax_metadata", None)
        if meta is not None:
            return json.loads(meta)
        return None

    @staticmethod
    def set_ax_meta(obj, ax_meta):
        assert obj.metadata is not None and obj.metadata.annotations is not None, "Need to pass an object that has annotations"
        obj.metadata.annotations["ax_metadata"] = json.dumps(ax_meta)


class AXResources(object):
    """
    This class stores information about each AXResource object in the main
    kubernetes object and has methods for marshalling and unmarshalling
    this information.
    """

    current_version = "1_0"

    def __init__(self, existing=None):
        """
        Pass an existing kubernetes object that may have the marshalled list
        of AXResource. If existing is None, then we are creating a new list
        that will be marshalled into a kubernetes object
        Args:
            existing:
        """
        self._resources = {}
        if existing:
            self._resources = AXResources._read_from_object(existing)

    def insert(self, obj):
        """
        Insert an object in the resources map, If the object name
        already exists then existing object will be updated
        Args:
            obj: AXResource object must be passed
        """
        name = obj.get_resource_name()
        restype = obj.get_resource_type()
        key = "{}/{}".format(restype, name)
        val = obj.get_resource_info()
        self._resources[key] = {
            "ax_time":  str(datetime.utcnow()),
            "type": restype,
            "resource": val
        }

    def remove(self, name):
        res = self._resources.pop(name)

    def __iter__(self):
        """
        Custom iterator for AXResources
        """
        for _, val in self._resources.iteritems():
            yield val["resource"]
        return

    def finalize(self, obj):
        """
        Write all the resources to kubernetes object
        Args:
            obj:
        """
        AXResources._write_to_object(obj, self._resources)

    def get_all_types(self):
        return self._resources.keys()

    def delete_all(self):
        objs = self._create_resources()
        for obj in objs:
            logger.debug("Calling delete on AXResource object {}".format(obj))
            obj.delete()

    def _create_resources(self):
        objs = []
        for _, res in self._resources.iteritems():
            try:
                full_type = res["type"]
                info = res["resource"]
                (mod_str, _, cls_str) = full_type.rpartition(".")
                cls = getattr(importlib.import_module(mod_str), cls_str)
                objs.append(cls.create_object_from_info(info))
            except Exception as e:
                logger.debug("Error {} while recreating resources".format(e))
        return objs

    @staticmethod
    def _read_from_object(obj):
        if obj.metadata is None or obj.metadata.annotations is None:
            return
        resources = obj.metadata.annotations.get("ax_resources", None)
        if resources is None:
            return

        resources_d = json.loads(resources)
        version = resources_d.get("version", "old")
        handler_name = "_read_from_object_v_{}".format(version)
        handler_func = getattr(AXResources, handler_name)
        assert handler_func, "No handler found for reading ax_resources of version {}".format(version)
        return handler_func(resources_d)

    @staticmethod
    def _read_from_object_v_1_0(res):
        return res["resources"]

    @staticmethod
    def _read_from_object_v_old(res):
        # TODO: This needs more thought on how to read ax_resources from old tasks
        return res

    @staticmethod
    def _write_to_object(obj, res_map):
        if obj.metadata is None:
            obj.metadata = swagger_client.V1ObjectMeta()
        if obj.metadata.annotations is None:
            obj.metadata.annotations = {}

        final = {
            "version": AXResources.current_version,
            "resources": res_map
        }
        obj.metadata.annotations["ax_resources"] = json.dumps(final)

