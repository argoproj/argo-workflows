#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

from ax.axdb.axdb import AXDB
from ax.axdb.axsys import host_table
from ax.axdb.axsys import host_usage_table
from ax.axdb.axsys import container_table
from ax.axdb.axsys import container_usage_table
from ax.axdb.axsys import artifacts_table
from ax.axdb.axsys import config_table

import sys


if __name__ == "__main__":
    if len(sys.argv) == 1:
        print "You need to pass in the axdb url, e.g. http://localhost:8080/v1"
        exit(1)
    db = AXDB(sys.argv[1])
#    db.create_table(host_table)
#    db.create_table(container_table)
#    db.create_table(host_usage_table)
#    db.create_table(container_usage_table)
#    db.create_table(artifacts_table)
#    db.create_table(config_table)
