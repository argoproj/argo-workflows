#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module to manage cloud specific features.
"""

import os

CLOUD_AWS = "CLOUD_AWS"
CLOUD_VBOX = "CLOUD_VBOX"
CLOUD_OTHER = "CLOUD_OTHER"


def cloud_provider():
    """
    Guess my cloud provider from within instance.

    This requires hosting container to run as privileged.
    :return: CLOUD_* type
    """
    try:
        for d in os.listdir("/sys/class/block"):
            if d.startswith("xvd"):
                return CLOUD_AWS
    except:
        pass

    try:
        with open("/sys/bus/pci/devices/0000:00:04.0/vendor", "r") as f:
            if f.read().strip() == "0x80ee":
                return CLOUD_VBOX
    except:
        pass

    return CLOUD_OTHER
