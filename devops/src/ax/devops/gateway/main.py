from gevent import monkey
monkey.patch_all()

from ax.util.az_patch import az_patch
az_patch()

import argparse
import logging
import signal
import sys
import threading

from gevent import pywsgi

import ax.devops.gateway.rest as rest
from ax.version import __version__
from ax.devops.gateway.gateway import Gateway
from ax.util.ax_signal import traceback_multithread
from . import GATEWAY_DEFAULT_PORT

logger = logging.getLogger(__name__)


def signal_handler(signalnum, *args):
    logger.info("Gateway killed with signal %s", signalnum)
    sys.exit(0)


def signal_debugger(signal_num, frame):
    logger.info("Gateway debugged with signal %s", signal_num)
    result = traceback_multithread(signal_num, frame)
    logger.info(result)


def main():
    parser = argparse.ArgumentParser(description='gateway')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--port', type=int, default=GATEWAY_DEFAULT_PORT, help="Run server on the specified port")
    args = parser.parse_args()

    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S",
                        stream=sys.stdout)
    logging.getLogger("ax").setLevel(logging.DEBUG)

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGUSR1, signal_debugger)

    try:
        rest.gateway = Gateway()
        # Start repo manager process
        logger.info("Starting Gateway repo_manager thread...")
        repo_manager_thread = threading.Thread(target=rest.gateway.repo_manager.run,
                                               name="repo_manager",
                                               daemon=True)
        repo_manager_thread.start()
        # Start event trigger process
        logger.info("Starting Gateway event_trigger thread...")
        event_trigger_thread = threading.Thread(target=rest.gateway.event_trigger.run,
                                                name="event_trigger",
                                                daemon=True)
        event_trigger_thread.start()

        # Start flask server
        logger.info("Starting Flask server...")
        server = pywsgi.WSGIServer(('', args.port), rest.app)
        logger.info("Gateway %s serving on port %s", __version__, args.port)
        server.serve_forever()
    except SystemExit:
        raise
    except Exception as err:
        logger.exception("Unhandled exception: %s", err)
        sys.exit(1)
