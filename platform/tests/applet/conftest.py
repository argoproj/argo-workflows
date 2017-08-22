#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import pytest
import subprocess
from ax.platform.applet.appdb import ApplicationRecord


@pytest.fixture(scope="module")
def app_record():
    ar = ApplicationRecord(db="/tmp/example.db", table_create=True)
    yield ar
    subprocess.call(["rm", "/tmp/example.db"])

