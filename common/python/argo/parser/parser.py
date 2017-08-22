#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import copy
import importlib

from future.utils import iteritems


class ObjectDefinition(object):

    def __init__(self, class_string=None, field_name=None, required=False, converter_tuple=(None, None)):
        self.cls = None
        if class_string:
            (mod_str, _, cls_str) = class_string.rpartition(".")
            self.cls = getattr(importlib.import_module(mod_str), cls_str)
        self.field_name = field_name
        self.required = required
        self.converter_tuple = converter_tuple

    @staticmethod
    def str_to_object_definition(string):
        assert isinstance(string, str), "{} must be of type string".format(string)
        return ObjectDefinition(class_string=string)

    @staticmethod
    def tuple_to_object_definition(t):
        assert isinstance(t, tuple), "{} must be of type tuple".format(t)
        if len(t) == 1:
            return ObjectDefinition(class_string=t[0])
        elif len(t) == 2:
            return ObjectDefinition(class_string=t[0], field_name=t[1])
        elif len(t) >= 3:
            return ObjectDefinition(class_string=t[0], field_name=t[1], required=t[2])


class ObjectParser(object):
    """
    This base class implements a method
    that parses dictionaries
    """
    def __init__(self):
        fields = getattr(self, "_object_parser_fields", None)
        self._field_map = fields or {}

    def set_fields(self, fields):
        self._field_map.update(fields)

    @staticmethod
    def convert_to(od, obj):
        if od.converter_tuple[0]:
            return od.converter_tuple[0](obj)
        else:
            return obj

    @staticmethod
    def convert_from(od, obj):
        if od.converter_tuple[1]:
            return od.converter_tuple[1](obj)
        else:
            return obj

    def parse(self, data, error_on_not_found=False):
        """
        This function parses a dictionary and builds the object. The object
        built is of type ObjectParser. Typical use is to create a class that 
        is the parent of ObjectParser and then set the _object_parser_fields
        attribute in the constructor before calling super().__init__()
        
        :param data: The data to be converted to object (dict)
        :param error_on_not_found: Raise ValueError exception if key does not
                                   have any data
        :return: nothing
        """
        if data is None:
            return
        for k,v in iteritems(self._field_map):
            data_k = data.get(k, None)
            od = ObjectParser.get_cls_and_field_name(v)
            if data_k is not None:
                # This is specifically checked for None to ensure that empty strings
                # are added to returned object
                if od.field_name is None:
                    field_name = k
                else:
                    field_name = od.field_name

                if od.cls is not None:
                    if isinstance(data_k, list):
                        obj = []
                        for each_k in data_k:
                            each_obj = od.cls()
                            each_obj.parse(each_k)
                            obj.append(each_obj)
                    else:
                        obj = od.cls()
                        obj.parse(data_k)

                    setattr(self, field_name, self.convert_to(od, obj))
                else:
                    # simple types are just copied
                    setattr(self, field_name, self.convert_to(od, copy.deepcopy(data_k)))
            else:
                if od.required or error_on_not_found:
                    raise ValueError("No data found for {} while generating {} using data {}".format(k, self.fqcn(), data))

    @staticmethod
    def get_cls_and_field_name(v):
        if v is None:
            return ObjectDefinition()
        if isinstance(v, str):
            return ObjectDefinition.str_to_object_definition(v)
        elif isinstance(v, tuple):
            return ObjectDefinition.tuple_to_object_definition(v)
        elif isinstance(v, ObjectDefinition):
            return v
        else:
            raise ValueError("Unexpected type {} here".format(type(v)))

    def to_dict(self):
        """
        This is the reverse of parse function. It returns a dictionary of k,v pair
        :return: dict
        """
        ret = {}
        for k,v in iteritems(self._field_map):
            od = ObjectParser.get_cls_and_field_name(v)
            if od.field_name is None:
                field_name = k
            else:
                field_name = od.field_name
            if od.cls is None:
                val = getattr(self, field_name, None)
                # only add keys in dict if value is not none
                if val is not None:
                    ret[k] = self.convert_from(od, copy.deepcopy(val))
            else:
                obj = getattr(self, field_name, None)

                # only add keys in dict if value is not none
                if not obj:
                    continue

                obj = self.convert_from(od, obj)
                if isinstance(obj, ObjectParser):
                    ret[k] = obj.to_dict()
                elif isinstance(obj, list):
                    ret[k] = [x.to_dict() for x in obj]
                else:
                    assert False, "Expect an object or list of objects in {}".format(k)
        return ret

    @classmethod
    def fqcn(cls):
        """
        stands for fully qualified class name :)
        """
        return cls.__module__ + "." + cls.__name__

