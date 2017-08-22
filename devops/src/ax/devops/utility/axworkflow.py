#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import copy
import uuid

from ax.devops.utility.axjson import substitute


def substitute_parameters(service_request):
    """Recursively substitute environment variables in a service request.

    :param service_request:
    :return:
    """
    if not service_request:
        return
    elif 'template' in service_request: # dynamic fixtures or steps
        if service_request['template'].get('steps') is None:
            service_request['template'] = substitute(service_request['template'], **service_request.get('parameters', {}))
        else:
            for f_or_s in ["steps", "fixtures"]:
                s = service_request['template'].get(f_or_s)
                if s is None:
                    continue
                for i in range(len(s)):
                    for key in s[i]:
                        substitute_parameters(s[i][key])
        if service_request['template'].get('volumes'):
            volumes = service_request['template'].get('volumes')
            if volumes and isinstance(volumes, dict):
                service_request['template']['volumes'] = substitute(service_request['template']['volumes'], **service_request.get('parameters', {}))
    else:
        for tag in ['requirements', 'name', 'class', 'category']:
            if tag in service_request: # static fixtures
                service_request[tag] = substitute(service_request[tag], **service_request.get('parameters', {}))
