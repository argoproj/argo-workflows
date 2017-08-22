#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Sleep module used for function calls that needs to be triggered
with a deterministic interval. For example:

while True:
    with AXSleep(60):
        foo()

foo() will be called once exactly every 60 seconds, except when
foo() takes more than 60 seconds to return

"""

import time


class AXSleep(object):
    def __init__(self, interval):
        self._start = 0
        self._end = 0
        self._interval = interval

    def __enter__(self):
        self._start = time.time()

    def __exit__(self, exc_type, exc_val, exc_tb):
        self._end = time.time()
        sleep_time = self._interval - (self._end - self._start)
        if sleep_time > 0:
            time.sleep(sleep_time)
