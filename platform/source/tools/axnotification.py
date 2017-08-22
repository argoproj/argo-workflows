#!/usr/bin/env python3

from gevent import monkey
monkey.patch_all()

from ax.util.az_patch import az_patch
az_patch()

from ax.platform.axnotification.app import myapp, views
from ax.platform.axnotification.app import db

from gevent import pywsgi
import argparse
from ax.version import __version__

try:
    parser = argparse.ArgumentParser(description='AXNotification')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    args = parser.parse_args()

    db.create_all()
    views.init_db()
    server = pywsgi.WSGIServer(('', 9889), myapp)
    server.serve_forever()
except KeyboardInterrupt:
    pass
