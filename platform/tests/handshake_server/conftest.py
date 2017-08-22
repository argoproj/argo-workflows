#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import os
import pytest
import subprocess

TEST_SOCK = "/tmp/test.sock"


@pytest.fixture(scope="module")
def handshake():
    PWD = os.path.dirname(__file__)
    handshake = os.path.join(PWD, "../../../common/python/ax/platform/applet/handshake.py")
    p = subprocess.Popen(["python", handshake, "&"])

    yield p

    p.kill()


