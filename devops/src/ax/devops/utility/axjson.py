#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import json
import os
import re


def load(file, **kwargs):
    """Load an AXJSON string from file.

    :param file:
    :param kwargs: The args need to be update.
    :return: A dictionary.
    """
    extension = os.path.splitext(file)
    if extension not in {'.json', '.axjson'}:
        raise RuntimeError('Given extension ({}) not supported'.format(extension))
    with open(file) as f:
        data = json.load(f)
        if extension == '.json':
            return data
        else:
            json_str = json.dumps(data)
            return loads(json_str, **kwargs)


def loads(string, **kwargs):
    """Load an AXJSON string.

    :param string:
    :param kwargs: The args need to be update.
    :return: A dictionary.
    """
    string = substitutes(string, **kwargs)
    return json.loads(string)


def substitutes(string, **kwargs):
    """Substitute variables in a JSON string.

    :param string:
    :param kwargs:
    :return: A string.
    """
    return json.dumps(substitute(json.loads(string), **kwargs))


def substitute(obj, **kwargs):
    """Recursively substitute a object.

    :param obj:
    :param kwargs:
    :return:
    """
    if type(obj) == dict:
        new_obj = {}
        for k in obj:
            new_obj[k] = substitute(obj[k], **kwargs)
        return new_obj
    elif type(obj) in {list, tuple}:
        new_obj = []
        for i in range(len(obj)):
            new_obj.append(substitute(obj[i], **kwargs))
        return type(obj)(new_obj)
    elif type(obj) in {set, frozenset}:
        new_obj = set()
        for v in obj:
            new_obj.add(substitute(v, **kwargs))
        return type(obj)(new_obj)
    elif type(obj) == str:
        for _ in range(len(kwargs)):
            replaced = 0
            for k, v in kwargs.items():
                new_obj = re.sub('%%{}%%'.format(k), str(v), obj)
                if new_obj != obj:
                    obj = new_obj
                    replaced += 1
            if not replaced:
                break
        return obj
    else:
        return obj
