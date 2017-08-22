#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
"""
Web service to provide exec console access and live logs to running containers
"""
from ax.util.az_patch import az_patch
az_patch()

from ax.platform.console.main import main
if __name__ == "__main__":
    main()
