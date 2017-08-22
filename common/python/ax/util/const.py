#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Definitions for common constants.
"""

# Sizes.
KB = 1000
MB = 1000 * KB
GB = 1000 * MB
TB = 1000 * GB
PB = 1000 * TB
EB = 1000 * PB

KiB = 1024
MiB = 1024 * KiB
GiB = 1024 * MiB
TiB = 1024 * GiB
PiB = 1024 * TiB
EiB = 1024 * PiB

# Kubernetes resources can have suffix lower case 'm', means 0.001
mB = 0.001

MS_PER_SECOND = 1000
US_PER_SECOND = 1000 * MS_PER_SECOND
NS_PER_SECOND = 1000 * US_PER_SECOND

# Time
SECONDS_PER_MINUTE = 60
SECONDS_PER_HOUR = 60 * SECONDS_PER_MINUTE
SECONDS_PER_DAY = 24 * SECONDS_PER_HOUR
SECONDS_PER_WEEK = 7 * SECONDS_PER_DAY

HOURS_PER_DAY = 24

# Color
# Harry: why do I use 9x rather than 3x? coz I LIKE BRIGHT COLORS :)
COLOR_RED = "\033[0;91m"
COLOR_GREEN = "\033[0;92m"
COLOR_YELLOW = "\033[0;93m"
COLOR_MAGENTA = "\033[0;95m"
COLOR_CYAN = "\033[0;96m"
COLOR_NORM = "\033[0m"

