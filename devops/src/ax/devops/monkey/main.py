from gevent import monkey
monkey.patch_all()

import argparse
import logging
import signal
import sys
from logging.handlers import RotatingFileHandler

from ax.version import __version__
from ax.devops.monkey.chaosmonkey import ChaosMonkey

from . import LOG_FILE_NAME

logger = logging.getLogger(__name__)


def signal_handler(signalnum, *args):
    logger.info("Chaos monkey killed with signal %s", signalnum)
    sys.exit(0)


def main():
    parser = argparse.ArgumentParser(description="Chaos Monkey")
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--cluster-name', help="Kube cluster name")
    parser.add_argument('--config-file', default=None, help="Config file, default:config.yaml")
    args = parser.parse_args()

    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S",
                        stream=sys.stdout)
    rotate_handler = RotatingFileHandler(LOG_FILE_NAME, maxBytes=100000)
    rotate_handler.setFormatter(logging.Formatter(fmt="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s",
                                                  datefmt="%Y-%m-%dT%H:%M:%S"))
    rotate_handler.setLevel(logging.DEBUG)
    logging.getLogger("ax").addHandler(rotate_handler)
    logging.getLogger("ax").setLevel(logging.INFO)

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)

    try:
        chaosmonkey = ChaosMonkey(cluster_name=args.cluster_name, config_file=args.config_file)
        chaosmonkey.run()
    except SystemExit:
        raise
    except Exception as err:
        logger.exception("Unhandled exception: %s", err)
        sys.exit(1)
