#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#

import platform

from ctypes import *
from ctypes.util import find_library

# See: https://bugs.python.org/issue23606 -Jesse
# ctypes.util.find_library("c") no longer makes sense
_libpath = find_library('c') or 'libc.so.6'
_libc = CDLL(_libpath)

def _get_monotonic_time_mac():
    class BASE(Structure):
        _fields_ = [("numer", c_uint32),
                    ("denom", c_uint32)]

    _libc.mach_absolute_time.restype = c_uint64
    current = _libc.mach_absolute_time()

    base = BASE(0, 0)
    _libc.mach_timebase_info(byref(base))
    return current / 1000000000 * base.numer / base.denom


def _get_monotonic_time_linux():
    class TS(Structure):
        _fields_ = [("tv_sec", c_ulong),
                    ("tv_nsec", c_ulong)]
    # As defined by libc in time.h
    CLOCK_MONOTONIC = 1

    ts = TS(0, 0)
    _libc.clock_gettime(CLOCK_MONOTONIC, byref(ts))
    return int(ts.tv_sec)


def get_monotonic_time():
    plat = platform.platform(terse=True)
    if "Darwin" in plat:
        # Mac
        return _get_monotonic_time_mac()
    elif "Linux" in plat:
        # Linux
        return _get_monotonic_time_linux()
    assert 0, "Unknown platform"


if __name__ == "__main__":
    """
    Simple test.
    """
    import time
    for i in range(1, 11):
        print "Sleeping %s seconds" % i,
        before = get_monotonic_time()
        time.sleep(i)
        after = get_monotonic_time()
        print "%s %s" % (before, after)
        assert abs(after - before - i) < 1
