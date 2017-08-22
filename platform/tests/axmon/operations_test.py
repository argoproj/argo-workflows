#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
from gevent import monkey
monkey.patch_all()

import logging
import unittest
import time
from random import uniform, randint

from multiprocessing.dummy import Pool

from ax.platform.operations import Operation, OperationsManager

logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")
logging.getLogger("ax").setLevel(logging.DEBUG)
logger = logging.getLogger(__name__)


class Op(object):
    """
    This is the shared object
    """
    def __init__(self):
        self.value = 42

    def incr(self):
        self.value += 1

    def decr(self):
        self.value -= 1

    def get(self):
        return self.value


class GuardedOperation(Operation):
    """
    This implements a guarded operation on object of type Op
    """
    def __init__(self, token, sleep_time, value_ref):
        super(GuardedOperation, self).__init__(token=token)
        self.sleep_time = sleep_time
        self.value_ref = value_ref

    def perform(self):
        with self:
            # lock before perform_no_lock (so this is locked)
            self._perform()

    def _perform(self):
        self.value_ref.incr()
        time.sleep(self.sleep_time)
        v = self.value_ref.get()
        assert v == 43, "Incorrect value got {}, expect 43".format(v)
        time.sleep(self.sleep_time)
        self.value_ref.decr()
        time.sleep(self.sleep_time)
        v = self.value_ref.get()
        assert v == 42, "Incorrect value got {}, expect 42".format(v)

    @staticmethod
    def prettyname():
        return "GuardedOperation"

single_op = Op()


def guarded_op(sleep_time):
    g = GuardedOperation("test1", sleep_time, single_op)
    g.perform()
    return 42


NUM_OPS = 10
OPS = [Op() for _ in range(0, NUM_OPS, 1)]


def guarded_ops(sleep_time):
    opnum = randint(0, NUM_OPS-1)
    g = GuardedOperation("op{}".format(opnum), sleep_time, OPS[opnum])
    g.perform()
    return 42


class OperationsTests(unittest.TestCase):

    def test_1(self):
        """
        Run single guarded operation on two threads
        """
        p = Pool(2)
        p.map(guarded_op, [0.5, 0.4])

    def test_2(self):
        """
        Run single guarded operation on two threads
        """
        count = 100
        p = Pool(count)
        p.map(guarded_op, [uniform(0, 0.5) for _ in range(0, count, 1)])

    def test_3(self):
        """
        Run multiple guarded operations on multiple threads
        """
        count = 100
        p = Pool(count)
        p.map(guarded_ops, [uniform(0, 0.5) for _ in range(0, count, 1)])

