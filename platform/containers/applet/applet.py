#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from ax.util.az_patch import az_patch
az_patch()

import argparse
import logging
import os

from ax.version import __version__
from ax.platform.applet.applet_main import Applet


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Applet')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    args = parser.parse_args()
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    logging.getLogger("ax.kubernetes.kubelet").setLevel(logging.INFO)
    logging.getLogger("botocore").setLevel(logging.WARNING)
    logging.getLogger("boto3").setLevel(logging.WARNING)

    Applet().run()
