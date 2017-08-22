#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging
import os
from twisted.python.filepath import FilePath
from twisted.internet.protocol import Protocol, Factory
from twisted.internet import reactor


DEFAULT_HANDSHAKE_SOCK = "/var/run/applatix.sock"
DEFAULT_HANDSHAKE_MSG_SIZE = 512

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


class DefaultHandshakeProtocol(Protocol):
    """ A simple echo protocol """

    def dataReceived(self, data):
        logger.info("Received data: %s. of type %s", data, type(data))
        rsp = self.generate_response_from_data(str(data.decode("utf-8")))
        self.transport.write(rsp.encode("utf-8"))
        logger.info("Replied data: %s in response to %s", rsp, data)
        self.transport.loseConnection()

    def generate_response_from_data(self, data):
        return data


class AXHandshakeServer(object):
    """
    A Twisted server that can handle multiple
    simultaneous handshake connections
    """
    def __init__(self, sock_addr=DEFAULT_HANDSHAKE_SOCK, proto=DefaultHandshakeProtocol):
        self._sock_addr = sock_addr
        logger.debug("Reactor: %s", reactor)
        # Make sure the socket does not exist
        try:
            os.remove(self._sock_addr)
        except OSError:
            if os.path.exists(self._sock_addr):
                raise

        self._address = FilePath(self._sock_addr)
        self._server_factory = Factory()
        self._server_factory.protocol = proto

    def start_server(self):
        reactor.listenUNIX(self._address.path, self._server_factory)
        logger.info("Serving %s", self._address.path)
        reactor.run()


if __name__ == '__main__':
    hs = AXHandshakeServer(sock_addr="/tmp/test.sock")
    hs.start_server()

