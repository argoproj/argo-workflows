#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
import unittest
import logging

from threading import Thread
from ax.platform.routes import ServiceOperation, ServiceEndpoint

logger = logging.getLogger("ax")

class Foo(Thread):

    def __init__(self, s, msg):
        self.s = s
        self.msg = msg
        super(Foo, self).__init__()

    def run(self):
        print "{}".format(self.msg)
        with ServiceOperation(self.s):
            print "{} enter".format(self.msg)
            time.sleep(5)
        print "{} done".format(self.msg)



class ServicesEndpointTests(unittest.TestCase):
    """
    Test for Service endpoints
    Requires kubectl proxy
    """
    @classmethod
    def setUpClass(self):
        pass

    @classmethod
    def tearDownClass(self):
        pass

    def test_1(self):
        """
        Test ServiceOperation overlap tests
        """
        s1 = ServiceEndpoint("a", "b")
        s2 = ServiceEndpoint("c", "d")

        f1 = Foo(s1, "s1")
        f2 = Foo(s2, "s2")
        f3 = Foo(s1, "s3")
        f4 = Foo(s1, "s4")

        f1.start()
        f2.start()
        f3.start()
        f4.start()

        f1.join()
        f2.join()
        f3.join()
        f4.join()

