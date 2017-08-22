#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Test utilities for applet
"""

import random
import string


def generate_random_string(ascii_lower=False, ascii_upper=False, digits=False, prefix=None, suffix=None, rand_len=0):
    pool = "" + \
           string.ascii_lowercase if ascii_lower else "" + \
           string.ascii_uppercase if ascii_upper else "" + \
           string.digits if digits else ""
    randmsg = "".join(random.choice(pool) for _ in range(rand_len))

    if prefix:
        randmsg = prefix + randmsg

    if suffix:
        randmsg = randmsg + suffix

    return randmsg