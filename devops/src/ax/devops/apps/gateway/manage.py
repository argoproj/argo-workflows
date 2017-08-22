#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

from ax.util.az_patch import az_patch
az_patch()

import os
import sys


if __name__ == "__main__":
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "gateway.settings")

    from django.core.management import execute_from_command_line

    execute_from_command_line(sys.argv)
