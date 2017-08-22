# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Core AX Exceptions

All exception defined in this file should have counterpart definition to the errors defined in the 
"Core AX errors" section in the common thrift file (and vice versa):
 
  prod/common/error/error.thrift

Thrift is used only to provide a consistent definition of errors across our APIs and languages.
However, the generated python code by thrift is not used for our python exception handling.
This is because the generated AXError class is not a true python Exception which can be
raised and excepted. Furthermore thrift does not have the ability for inheritance or exception
hierarchies, which we wish to use in our python code. Thus, the error definitions are hand
duplicated in this file, at least for the AX core exceptions.
"""
import json
import logging

logger = logging.getLogger(__name__)


class AXException(Exception):
    code = "ERR_AX_INTERNAL"
    status_code = 500

    def __init__(self, message, detail=None):
        """Base class for AX Exceptions
        
        :param message: exception message
        :param detail: JSON serializable detail information
        :raises: TypeError if detail is not JSON serializable 
        """
        super(AXException, self).__init__(message)
        if detail is not None and type(detail) != dict:
            detail = str(detail)
        self.detail = detail
        json.dumps(self.json())  # Verify the object is json serializable

    def json(self):
        """Returns dictionary representation of this error, suitable as a REST API return value"""
        return {'code': self.code,
                'message': self.args[0],
                'detail': self.detail if self.detail else ""}


def deserialize(error_json):
    """Given an Argo API error json response, return a new instance of its corresponding exception"""
    if deserialize.mapped_codes is None:
        # Build up a mapping from error code, to exception class. do this only once
        # Attempt import ax.platform.exceptions and ax.devops.exceptions so that
        # AXExeception.__subclasses__() can detect all subclasses
        try:
            import ax.platform.exceptions
        except ImportError:
            pass
        try:
            import ax.devops.exceptions
        except ImportError:
            pass
        mapped_codes = {}
        for ax_exc_class in AXException.__subclasses__():
            mapped_codes[ax_exc_class.code] = ax_exc_class
        deserialize.mapped_codes = mapped_codes

    if error_json['code'] in deserialize.mapped_codes:
        error_class = deserialize.mapped_codes[error_json['code']]
    else:
        logger.warning("Error code %s does not match any exception. Defaulting to AXException", error_json['code'])
        deserialize.mapped_codes[error_json['code']] = AXException
        error_class = AXException
    error_instance = error_class(error_json['message'], detail=error_json.get('detail'))
    error_instance.code = error_json['code']
    return error_instance


# Static variable to keep mapping of error codes to AXException classes
deserialize.mapped_codes = None


class AXTimeoutException(AXException):
    code = "ERR_AX_TIMEOUT"
    status_code = 408


class AXIllegalArgumentException(AXException):
    code = "ERR_AX_ILLEGAL_ARGUMENT"
    status_code = 422


class AXIllegalOperationException(AXException):
    code = "ERR_AX_ILLEGAL_OPERATION"
    status_code = 400


class AXServiceTemporarilyUnavailableException(AXException):
    code = "ERR_AX_SERVICE_TEMPORARILY_UNAVAILABLE"
    status_code = 503


# REST API Exceptions

class AXApiResourceNotFound(AXException):
    code = "ERR_API_RESOURCE_NOT_FOUND"
    status_code = 404


class AXApiInvalidParam(AXException):
    code = "ERR_API_INVALID_PARAM"
    status_code = 400


class AXApiAuthFailed(AXException):
    code = 'ERR_API_AUTH_FAILED'
    status_code = 401


class AXApiForbiddenReq(AXException):
    code = 'ERR_API_FORBIDDEN_REQ'
    status_code = 403


class AXApiInternalError(AXException):
    code = 'ERR_API_INTERNAL_ERROR'
    status_code = 500


class AXKubeApiException(AXException):
    code = "ERR_AX_KUBE_API"
    status_code = 422


class AXUnauthorizedException(AXException):
    code = "ERR_AX_UNAUTHORIZED"
    status_code = 401


class AXNotFoundException(AXException):
    code = "ERR_AX_NOTFOUND"
    status_code = 404


class AXConflictException(AXException):
    code = "ERR_AX_CONFLICT"
    status_code = 409


# For Axops parse specific purposes
class AXWorkflowAlreadyFailed(AXIllegalArgumentException):
    code = "ERR_AX_WORKFLOW_ALREADY_FAILED"


class AXWorkflowAlreadySucceed(AXIllegalArgumentException):
    code = "ERR_AX_WORKFLOW_ALREADY_SUCCEED"


class AXWorkflowDoesNotExist(AXIllegalArgumentException):
    code = "ERR_AX_WORKFLOW_NOT_EXIST"
