#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Constants used by applet packet
"""

APPLET_SOCK = "/var/run/applet.sock"
APPLET_DB = "/ax/applet/applet.db"
APPLET_TABLE_NAME = "app_records"
APPLET_SYNC_PERIOD = 10

CUR_RECORD_VERSION = "1.0"
CUR_HB_VERSION = "1.0"


class HeartBeatType(object):
    ARTIFACT_LOAD_START = "LOADING_ARTIFACTS"
    ARTIFACT_LOAD_FAILED = "ARTIFACT_LOAD_FAILED"
    BIRTH_CRY = "BIRTH_CRY"
    HEART_BEAT = "HEART_BEAT"
    TOMB_STONE = "TOMB_STONE"
