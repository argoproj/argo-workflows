# Copyright 2015-2016 Applatix, Inc. All rights reserved.

from gevent import monkey
monkey.patch_all()

from ax.util.az_patch import az_patch
az_patch()

import argparse
import logging
import signal
import sys

from gevent import pywsgi

from ax.cloud import Cloud
from ax.devops.artifact import rest
from ax.version import __version__
from ax.util.ax_signal import traceback_multithread
from .artifactmanager import ArtifactManager

logger = logging.getLogger(__name__)

ARTIFACTMANAGER_DEFAULT_PORT = 9892


def signal_handler(signal_num, *args):
    logger.info("Artifact manager killed with signal %s", signal_num)
    sys.exit(0)


def signal_debugger(signal_num, frame):
    logger.info("Artifact manager debugged with signal %s", signal_num)
    result = traceback_multithread(signal_num, frame)
    logger.info(result)


def main():
    parser = argparse.ArgumentParser(description='artifactmanager')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--port', type=int, default=ARTIFACTMANAGER_DEFAULT_PORT, help="Run server on the specified port")
    args = parser.parse_args()

    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S",
                        stream=sys.stdout)
    logging.getLogger("ax").setLevel(logging.DEBUG)

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGUSR1, signal_debugger)

    try:
        Cloud().set_target_cloud(Cloud().own_cloud())
        rest.artifact_manager = ArtifactManager()
        rest.artifact_manager.init()
        rest.artifact_manager.start_background_process()  # start retention thread
        server = pywsgi.WSGIServer(('', args.port), rest.app)
        logger.info("Artifact manager %s serving on port %s", __version__, args.port)
        server.serve_forever()
    except SystemExit:
        raise
    except Exception as err:
        logger.exception("Unhandled exception: %s", err)
        if rest.artifact_manager:
            rest.artifact_manager.stop_background_process()  # stop retention thread
        sys.exit(1)
