#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import datetime
import logging
import os
import threading
import traceback

from abc import ABCMeta, abstractmethod
from threading import Lock, Condition

from ax.util.singleton import Singleton
from ax.platform.exceptions import AXPlatformException
from future.utils import with_metaclass

logger = logging.getLogger(__name__)


class Trace(object):
    def __init__(self):
        self._stack = traceback.extract_stack()
        self._formatted = None

    def __str__(self):
        if self._formatted:
            return self._formatted
        else:
            self._formatted = ""
            for x in self._stack or []:
                self._formatted += "-> {}:{} {} ".format(x[0].rpartition("/")[2], x[1], x[2])
            return self._formatted


class OperationException(AXPlatformException):
    status_code = 409


class Operation(with_metaclass(ABCMeta, object)):

    """
    Stores semantic information about the operation being performed.
    For now, we use the type of the object to make decisions about
    concurrency of operations.
    """
    def __init__(self, token=None):
        self._token = token if token is not None else "static"
        self._manager = OperationsManager()
        self._thread = None
        self._stack = None
        self._lock_time = None
        self._conflict_count = 0

    def __enter__(self):
        while True:
            try:
                logger.debug("Trying to register {}".format(self))
                self._manager.register(self)
                logger.debug("Registered {}".format(self))
                return self
            except OperationException as oe:
                logger.debug(oe)
                self._manager.wait()
                logger.debug("Waking up {}".format(self))

        assert False, "Operation should not reach here"

    def register_thread_and_stack(self):
        self._thread = threading.current_thread().name
        self._stack = Trace()
        self._lock_time = datetime.datetime.now()

    def lock_details(self):
        return "Thread {} Stack {} Locked at {}".format(self._thread, self._stack, self._lock_time)

    def increment_conflict(self):
        self._conflict_count += 1
        currtime = datetime.datetime.now()
        try:
            delta = currtime - self._lock_time
            if delta.total_seconds() > 300.0:
                logger.debug("Operation {} is locked for more {} seconds and has conflicted {} times".format(
                    self, delta.total_seconds(), self._conflict_count))
                logger.debug("Lock details {}".format(self.lock_details()))

            if delta.total_seconds() > 600.0:
                logger.debug("Operation {} is locked for more {} seconds and has conflicted {} times".format(
                    self, delta.total_seconds(), self._conflict_count))
                logger.debug("Lock details {}".format(self.lock_details()))
                logger.warn("Going to terminate process as it is possible an IO op is stuck forever")
                os._exit(1)

        except Exception as e:
            logger.debug("Unexpected exception in compting time delta for {}".format(self))

    def __exit__(self, exc_type, exc_value, traceback):
        self._manager.deregister(self)
        self._manager.release()

    def type(self):
        return type(self)

    def set_token(self, token):
        self._token = token

    @property
    def token(self):
        return self._token

    def check_type(self, other_type):
        return True

    def check_object(self, operation):
        if self.type() == operation.type() and operation.token == self._token:
            return False
        return True

    @staticmethod
    @abstractmethod
    def prettyname():
        return "Operation"

    def __str__(self):
        return "{}({})".format(self.prettyname(), self.token)


class TaskOperation(Operation):

    def __init__(self, task_obj):
        token = task_obj.jobname
        super(TaskOperation, self).__init__(token)

    @staticmethod
    def prettyname():
        return "TaskOperation"


class OperationsManager(with_metaclass(Singleton, object)):
    """
    This class stores a list of current operations
    objects and has the logic for allowing or preventing
    concurrent operations to be performed. It is up to the
    caller to decide when to enclose some code in an operation.

    Implementation: Each supported type can be stored in a map
    so that operations can be added and retrieved easily. In
    addition checks be defined in each operation as to how they
    interact with other operations. These checks will be called
    for each category of operation if the map for that operation
    contains a non-zero count of objects.
    """

    def __init__(self):
        self._db = {}
        self._lock = Lock()
        self._cond = Condition()

    def register(self, operation):
        otype = operation.type()
        with self._lock:
            self.operation_allowed(operation)
            if otype not in self._db:
                self._db[otype] = {}
            self._db[otype][operation] = 1
            operation.register_thread_and_stack()

    def deregister(self, operation):
        otype = operation.type()
        with self._lock:
            self._db[otype].pop(operation)

    def operation_allowed(self, operation):
        for types in self._db:
            for existing_operation in self._db[types]:
                if not operation.check_type(types):
                    existing_operation.increment_conflict()
                    raise OperationException("Operation {} conflicts with other operations of type {}".format(operation.prettyname(), types.prettyname()))
                if not existing_operation.check_object(operation):
                    existing_operation.increment_conflict()
                    raise OperationException("Operation {} conflicts with operation {}".format(operation, existing_operation))

        return True

    def wait(self):
        with self._cond:
            self._cond.wait(timeout=60)

    def release(self):
        with self._cond:
            self._cond.notifyAll()

    def __str__(self):
        retstr = ""
        with self._lock:
            for types in self._db:
                l = len(self._db[types])
                if l > 0:
                    retstr += "{} {}\n".format(types.prettyname(), l)

        return retstr

    def lockstats(self):
        logger.debug("OperationsManager LockStats")
        with self._lock:
            logger.debug("{:50} | {:8} | {:30} | {}".format("Name", "Conflict", "LockTime", "Stack"))
            for types in self._db:
                for op in self._db[types]:
                    logger.debug("{:50} | {:8} | {:30} | {} {}".format(op, op._conflict_count, str(op._lock_time), op._thread, op._stack))
