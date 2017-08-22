#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Random code for random output
"""

import random
import string


def random_string(length):
    """
    Generate a random string of length
    Args:
        length: length of string requested

    Returns: string
    """
    return ''.join(random.choice(string.lowercase) for _ in range(length))
