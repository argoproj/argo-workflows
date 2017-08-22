from gevent import monkey
monkey.patch_all()

from ax.util.az_patch import az_patch
az_patch()

import argparse
import logging
import os
import subprocess
import signal
import sys

from gevent import pywsgi

import ax.devops.fixture.rest as rest
from ax.util.ax_signal import traceback_multithread
from ax.version import __version__
from .manager import FixtureManager
from . import FIXTUREMANAGER_DEFAULT_PORT

logger = logging.getLogger(__name__)

def signal_handler(signalnum, *args):
    logger.info("fixturemanager killed with signal %s", signalnum)
    sys.exit(0)

def signal_debugger(signal_num, frame):
    logger.info("fixturemanager debugged with signal %s", signal_num)
    result = traceback_multithread(signal_num, frame)
    logger.info(result)

def start_mongodb():
    """Fork mongodb process and return the process id"""
    logger.info("Starting mongodb")
    if not os.path.isdir('/data/db'):
        os.makedirs('/data/db')
    proc = subprocess.Popen(["/usr/bin/mongod"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    logger.info("Started mongodb as pid %s", proc.pid)
    return proc

def main():
    parser = argparse.ArgumentParser(description='fixturemanager')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--port', type=int, default=FIXTUREMANAGER_DEFAULT_PORT, help="Run server on the specified port")
    parser.add_argument('--redis', default='redis', help="Redis host")
    parser.add_argument('--mongodb', default=None, help="MongoDB host")
    parser.add_argument('--axops', default='axops-internal', help="Axops host")
    args = parser.parse_args()

    logging.basicConfig(format="%(asctime)s %(levelname)5s %(threadName)s %(name)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S",
                        stream=sys.stdout)
    logging.getLogger("ax").setLevel(logging.DEBUG)
    logging.getLogger("transitions").setLevel(logging.INFO)

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGUSR1, signal_debugger)
    try:
        mongo_proc = None
        if os.path.isfile('/.dockerenv') and args.mongodb is None:
            mongo_proc = start_mongodb()
        rest.fixmgr = FixtureManager(mongodb_host=args.mongodb, redis_host=args.redis, axops_host=args.axops)
        rest.fixmgr.initdb()
        rest.fixmgr.check_consistency()
        rest.fixmgr.notify_template_updates()
        rest.fixmgr.reqproc.start_processor()
        rest.fixmgr.volumemgr.start_workers()
        rest.fixmgr.start_workers()
        rest.fixmgr.reqproc.trigger_processor()
        rest.app.logger.setLevel(logging.DEBUG)
        server = pywsgi.WSGIServer(('', args.port), rest.app)
        logger.info("fixturemanager %s serving on port %s", __version__, args.port)
        if mongo_proc:
            server.start()
            exit_status = mongo_proc.wait()
            logger.error("Mongodb process exited with %s", exit_status)
            sys.exit(1)
        else:
            server.serve_forever()
    except SystemExit:
        raise
    except Exception as err:
        logger.exception("Unhandled exception: %s", err)
        sys.exit(1)
    finally:
        if mongo_proc and mongo_proc.returncode is None:
            logger.info("Terminating mongodb")
            mongo_proc.terminate()
        if rest.fixmgr:
            rest.fixmgr.reqproc.stop_processor()
            rest.fixmgr.volumemgr.stop_workers()
            rest.fixmgr.stop_workers()
