#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Main entry point for AXmon.
"""
from gevent import monkey
monkey.patch_all()

from ax.util.az_patch import az_patch
az_patch()

import argparse
import logging
import signal

from ax.cloud import Cloud
from ax.platform.axmon_main import AXMon, __version__, AXMON_DEFAULT_PORT
from ax.platform.rest import axmon_rest_start


def debug(sig, frame):
    logger = logging.getLogger("ax")
    import gc
    import traceback
    from greenlet import greenlet

    for ob in gc.get_objects():
        if not isinstance(ob, greenlet):
            continue
        if not ob:
            continue
        logger.debug(''.join(traceback.format_stack(ob.gr_frame)))


if __name__ == "__main__":
    """
    Main entry point for AXmon.
    """
    parser = argparse.ArgumentParser(description='AXMon')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--port', type=int, default=AXMON_DEFAULT_PORT, help="Run server on the specified port")
    args = parser.parse_args()

    # Basic logging.
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    logging.getLogger("botocore").setLevel(logging.WARNING)
    logging.getLogger("boto3").setLevel(logging.WARNING)

    Cloud().set_target_cloud(Cloud().own_cloud())
    signal.signal(signal.SIGUSR1, debug)
    axmon_rest_start(port=args.port)
    AXMon().run()
