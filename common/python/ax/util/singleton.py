#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#
import abc

from future.utils import with_metaclass


class Singleton(type):
    _instances = {}
    def __call__(cls, *args, **kwargs):
        if cls not in cls._instances:
            cls._instances[cls] = super(Singleton, cls).__call__(*args, **kwargs)
        return cls._instances[cls]


class SingleABC(Singleton, abc.ABCMeta):
    # helper class for solving Singleton and abc.ABCMeta
    # metaclass conflict
    pass

"""
if __name__ == "__main__":
    class SingletonTest(with_metaclass(Singleton, object)):
        def __init__(self):
            self._test = 0

        def get(self):
            return self._test

        def set(self, new):
            self._test = new

    x = SingletonTest()
    y = SingletonTest()
    val = 0xdeadbeef
    x.set(val)
    assert x.get() == val
    assert y.get() == val

    class SingleABCTest(with_metaclass(SingleABC, object)):

        def __init__(self):
            self._val = 1

        @abc.abstractmethod
        def get_val(self):
            pass

    try:
        a = SingleABCTest()
    except TypeError as t:
        print "TypeError expected as SingleABCTest is abstract class and cannot be instantiated"

    class A(SingleABCTest):
        def get_val(self):
            print self._val

    a = A()
    b = A()

    assert id(a) == id(b)
"""