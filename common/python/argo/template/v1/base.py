#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from argo.parser.parser import ObjectParser


class BaseTemplate(ObjectParser):

    def __init__(self):
        self.name = None
        self.type = None
        self.description = None
        self.labels = {}

        self._object_parser_fields = {
            "name": None,
            "type": None,
            "description": None,
            "labels": None
        }
        super(BaseTemplate, self).__init__()
