#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
from rest_framework import exceptions as rest_exceptions
from rest_framework.response import Response

from ax import exceptions as ax_exceptions

exception_mapping = {
    rest_exceptions.APIException: ax_exceptions.AXApiInternalError,
    rest_exceptions.AuthenticationFailed: ax_exceptions.AXApiAuthFailed,
    rest_exceptions.MethodNotAllowed: ax_exceptions.AXApiResourceNotFound,  # Method not allowed is casted to resource not found
    rest_exceptions.NotAuthenticated: ax_exceptions.AXApiAuthFailed,
    rest_exceptions.NotFound: ax_exceptions.AXApiResourceNotFound,
    rest_exceptions.ParseError: ax_exceptions.AXApiInvalidParam,
    rest_exceptions.PermissionDenied: ax_exceptions.AXApiForbiddenReq,
    rest_exceptions.ValidationError: ax_exceptions.AXApiInvalidParam
}


def ax_exception_handler(e, context):
    """Customize return error code.

    :param e:
    :param context:
    :return:
    """
    if not isinstance(e, ax_exceptions.AXException):
        exception_class = exception_mapping.get(e.__class__, ax_exceptions.AXApiInternalError)
        try:
            e = exception_class(e.detail)
        except AttributeError:
            e = exception_class(str(e))
    return Response(e.json(), status=e.status_code)
