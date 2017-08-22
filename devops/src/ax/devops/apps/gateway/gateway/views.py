#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import random

from django.shortcuts import redirect

from rest_framework.decorators import api_view
from rest_framework.exceptions import NotFound
from rest_framework.response import Response


@api_view(['GET'])
def hello_world(request):
    """A hello world API for user to test if gateway is up.

    :param request:
    :return:
    """
    messages = [
        'Hello!',
        'Greetings!',
        'Nice to see you!',
        'Welcome!',
    ]
    return Response({'message': random.choice(messages)})


@api_view(['GET', 'POST', 'PUT', 'DELETE', 'HEAD', 'OPTIONS', 'PATCH'])
def resource_not_found(request):
    """An API for all requests not matching any URLs.

    :param request:
    :return:
    """
    if not request.path.endswith('/'):
        return redirect(request.path + '/')
    else:
        raise NotFound('Requested resource ({}) not found'.format(request.path))
