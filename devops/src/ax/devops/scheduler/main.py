from gevent import monkey
monkey.patch_all()

import argparse
import logging
import signal
import sys
import ax.devops.scheduler.rest as rest
from gevent import pywsgi

from ax.version import __version__
from . import JOBSCHEDULER_DEFAULT_PORT
from ax.devops.scheduler.jobscheduler import JobScheduler

logger = logging.getLogger(__name__)


def signal_handler(signalnum, *args):
    logger.info("Job scheduler killed with signal %s", signalnum)
    sys.exit(0)


def main():
    parser = argparse.ArgumentParser(description='jobscheduler')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--port', type=int, default=JOBSCHEDULER_DEFAULT_PORT, help="Run server on the specified port")
    parser.add_argument('--axops', default='axops-internal', help="Axops host")
    args = parser.parse_args()

    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S",
                        stream=sys.stdout)
    logging.getLogger("ax").setLevel(logging.DEBUG)

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)

    try:
        rest.jobScheduler = JobScheduler(args.axops)
        rest.jobScheduler.init()
        server = pywsgi.WSGIServer(('', args.port), rest.app)
        logger.info("Job scheduler %s serving on port %s", __version__, args.port)
        server.serve_forever()
    except SystemExit:
        raise
    except Exception as err:
        logger.exception("Unhandled exception: %s", err)
        sys.exit(1)
