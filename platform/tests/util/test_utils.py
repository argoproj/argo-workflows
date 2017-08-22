#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from ax.cloud.aws.util import default_aws_retry
from botocore.exceptions import ClientError, EndpointConnectionError, ConnectionClosedError, BotoCoreError


def test_default_aws_retry():
    # AssertionError should not retry
    e = AssertionError()
    assert not default_aws_retry(e)

    # NotImplementedError should not retry
    e = NotImplementedError()
    assert not default_aws_retry(e)

    # KeyError should not retry
    e = KeyError()
    assert not default_aws_retry(e)

    # IndexError should not retry
    e = IndexError()
    assert not default_aws_retry(e)

    # EndpointConnectionError should retry
    e = EndpointConnectionError(endpoint_url=None, error=None)
    assert default_aws_retry(e)

    # ConnectionClosedError should retry
    e = ConnectionClosedError(endpoint_url=None)
    assert default_aws_retry(e)

    # Generic BotoCoreError should not retry
    e = BotoCoreError()
    assert not default_aws_retry(e)

    # Generic ClientError should not retry
    err = {
        "Error": {
            "Code": "xxx",
            "Message": "xxx"
        }
    }
    e = ClientError(error_response=err, operation_name="xxx")
    assert not default_aws_retry(e)

    # ClientError with RequestLimitExceeded should retry
    err = {
        "Error": {
            "Code": "RequestLimitExceeded",
            "Message": "xxx"
        }
    }
    e = ClientError(error_response=err, operation_name="xxx")
    assert default_aws_retry(e)
