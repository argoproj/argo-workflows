#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from .resource import ResourceValueConverter

class RetryWrapper(object):
    def __init__(self, obj, decorator=lambda x: x):
        self.obj = obj
        self.decorator = decorator

    def __getattr__(self, attr):
        o_attr = self.obj.__getattribute__(attr)
        if callable(o_attr):
            def wrapper(*args, **kwargs):

                @self.decorator
                def retry_wrapper(*args, **kwargs):
                    return o_attr(*args, **kwargs)

                return retry_wrapper(*args, **kwargs)

            return wrapper
        else:
            return o_attr
