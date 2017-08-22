#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module to handle callbacks
"""

import logging

from collections import deque
try:
    from queue import Queue
except ImportError:
    from Queue import Queue
try:
    from threading import _Semaphore as _Semaphore
except:
    from threading import Semaphore as _Semaphore

from threading import Thread
from future.utils import with_metaclass

from ax.util.singleton import Singleton

logger = logging.getLogger(__name__)


class ReturnWrapperCond(_Semaphore):
    def __init__(self, value=1):
        super(ReturnWrapperCond, self).__init__(value=value)
        self.exception = None
        self.ret = None


class Callback(Thread):

    def __init__(self):
        # FIFO queue Replace with collections.deque later on
        self._q = Queue()
        self._cbs = deque()
        super(Callback, self).__init__()
        self.daemon = True

    def post_event(self, *args, **kwargs):
        self._q.put((args, kwargs))

    def add_cb(self, func):
        # deque has atomic append
        self._cbs.append(func)
        logger.debug("Callback added")

    def run(self):
        """
        The event processing loop
        """
        logger.debug("Callback event processing loop has started")
        while True:
            args, kwargs = self._q.get()
            if args is None:
                break
            self._process_event(args, kwargs)

        logger.debug("Callback event processing loop has stopped")

    def stop(self):
        self._q.put((None, None))

    def _process_event(self, args, kwargs):
        for cb in self._cbs:
            try:
                cb(*args, **kwargs)
            except Exception as e:
                logger.exception("Callback exception: %s", e)


class ContainerEventCallbacks(with_metaclass(Singleton, Callback)):

    def __init__(self):
        super(ContainerEventCallbacks, self).__init__()
        self.start()
