# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
This module has functions used for specific conversions
"""

import re

NOT_DNS_LABEL_CHAR = re.compile(r"[^a-z0-9-]+")
NOT_K8S_LABEL_CHAR = re.compile(r"[^a-zA-Z0-9-_\.]+")

def string_to_dns_label(string):
    """
    This function takes a string and returns a valid dns label

    From Kubernetes spec https://github.com/kubernetes/kubernetes/blob/master/pkg/api/types.go
    // DNS_LABEL:  This is a string, no more than 63 characters long, that conforms
    //             to the definition of a "label" in RFCs 1035 and 1123.  This is captured
    //             by the following regex:
    //             [a-z0-9]([-a-z0-9]*[a-z0-9])?

    :param string: string
    :return: string that matches dns label spec
    """
    return string_to_label(string.lower(), NOT_DNS_LABEL_CHAR)

def string_to_k8s_label(string):
    """
    This function takes a string and returns a valid k8s label

    From Kubernetes spec https://kubernetes.io/docs/concepts/overview/working-with-objects/labels

    :param string: string
    :return: string that matches k8s label spec
    """
    return string_to_label(string, NOT_K8S_LABEL_CHAR)

def string_to_label(string, invalid_chars):

    mod = invalid_chars.sub("", string)
    if len(mod) == 0:
        raise ValueError("Cannot convert {} into a dns label. Removing non dns characters results in empty string".format(string))

    if mod[0] == '-':
        mod = mod[1:]

    if len(mod) == 0:
        raise ValueError("Cannot convert {} into a dns label. Removing non dns characters results in empty string".format(string))

    mod = mod[:63]
    if mod[-1] == '-':
        mod = mod[:-1]

    return mod

