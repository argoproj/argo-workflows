#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

# Need boto3 import to get exceptions.
import boto3
from botocore.exceptions import ClientError, EndpointConnectionError, ConnectionClosedError, BotoCoreError

from .consts import AWSPartitions

def default_aws_retry(exception):
    # Boto has 3 types of errors (see botocore/exceptions.py)
    # ConnectionError, BotoCoreError, ClientError
    #
    # ConnectionError happens when network has glitches or there is  an unexpected endpoint that it cannot connect to;
    # BotoCoreError happens where there is local preparation failures before actually making the API call
    # ClientError happens where HTTP code > 300, but according to current experience, we never see any retryable
    # error codes from ClientError except RequestLimitExceeded, so we are not blindly retrying here until
    # we find something worth it.
    if isinstance(exception, AssertionError) or \
            isinstance(exception, NotImplementedError) or \
            isinstance(exception, KeyError) or \
            isinstance(exception, IndexError):
        # Some generic not-retryable errors
        return False
    elif isinstance(exception, EndpointConnectionError) or \
            isinstance(exception, ConnectionClosedError):
        # We retry anything connection related
        return True
    elif isinstance(exception, BotoCoreError):
        # Other generic BotoCoreError are not related to server errors
        # or generic network errors, so we should not retry
        return False
    elif isinstance(exception, ClientError):
        # For ClientError, only retry when we hit request limit
        if "RequestLimitExceeded" in str(exception):
            return True
        else:
            return False
    else:
        # Retry any other unknown errors
        return True


def tag_dict_to_aws_filter(tags):
    """
    Convert key-value tag dictionary to boto3's filter list
    :param tags: dictionary. key=tag-key, value=list_of_tag_values
    :return:
    """
    if not tags:
        return []
    assert isinstance(tags, dict), "tags must be a dictionary"
    filters = []
    for k in tags.keys():
        assert isinstance(tags[k], list), "tag value must be a list"
        filters.append(
            {
                "Name": "tag-key",
                "Values": [k]
            }
        )
        filters.append(
            {
                "Name": "tag-value",
                "Values": tags[k]
            }
        )
    return filters


def get_aws_partition_from_region(region_name):
    """
    Given region, return the partition name. Partition is used to form Amazon Resource Number (ARNs)
    See http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
    :param region_name:
    :return:
    """
    region_string = region_name.lower()
    if region_string.startswith("cn-"):
        return AWSPartitions.PARTITION_CHINA
    elif region_string.startswith("us-gov"):
        return AWSPartitions.PARTITION_US_GOV
    else:
        return AWSPartitions.PARTITION_AWS
