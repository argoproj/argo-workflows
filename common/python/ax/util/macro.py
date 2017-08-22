#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module to replace a macro in config string or dict object.
Very simple now. Should be changed to handle default values.
"""

import json

def macro_replace(input, macros):
    """
    :param input: Input value, must be either string or dict.
                  Input string must in format of $(HOSTNAME) or ${HOSTNAME}
    :param macros: Dict for macro replacement, e.g. {"HOSTNAME": "127.0.0.1"}
    :return: Replaced config.
    """
    convert_back = False
    if isinstance(input, dict):
        out = json.dumps(input)
        convert_back = True
    elif isinstance(input, basestring):
        out = input
    else:
        assert 0, "Input must be string or dict %s" % input

    for m in macros:
        if macros[m] is not None:
            out = out.replace("$(%s)" % m, macros[m])
            out = out.replace("${%s}" % m, macros[m])
            out = out.replace("#{%s}" % m, macros[m])


    if convert_back:
        out = json.loads(out)
    return out
