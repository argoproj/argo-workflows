from gevent import monkey, pywsgi
# monkey patching needs be performed as early as possible
monkey.patch_all()

import argparse
import logging
import sys

from ax.version import __version__

logger = logging.getLogger(__name__)

def main():
    """Entry point to axconsole web service"""
    parser = argparse.ArgumentParser(description='AXConsole')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--port', type=int, help="Run server on the specified port")
    args = parser.parse_args()
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S",
                        stream=sys.stdout)
    logging.getLogger("ax").setLevel(logging.DEBUG)

    from ax.platform.console import AXCONSOLE_DEFAULT_PORT, rest
    port = args.port or AXCONSOLE_DEFAULT_PORT
    from geventwebsocket.handler import WebSocketHandler

    try:
        server = pywsgi.WSGIServer(('', port), rest.app, handler_class=WebSocketHandler)
        logger.info("AXconsole %s serving on port %s", __version__, port)
        server.serve_forever()
    except KeyboardInterrupt:
        pass
