#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging
from ax.cluster_management import ArgoClusterManager


if __name__ == "__main__":
    logging.basicConfig(format="%(asctime)s %(levelname)s [ARGO] %(name)s %(lineno)d %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    logging.getLogger("ax").setLevel(logging.INFO)

    app = ArgoClusterManager()
    app.add_flags()
    app.parse_args_and_run()
