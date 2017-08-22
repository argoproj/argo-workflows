#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

from ax.util.az_patch import az_patch
az_patch()

import argparse
import logging

from ax.version import __version__
from ax.platform.stats import AXStats


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='AXStats')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    args = parser.parse_args()
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s %(message)s")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    logger = logging.getLogger("ax.stats")
    logger.debug("AXStats %s server starting", __version__)
    AXStats().watch()
