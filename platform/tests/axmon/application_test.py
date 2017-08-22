#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
import unittest
import logging

from ax.platform.application import Application
from ax.exceptions import AXKubeApiException

logger = logging.getLogger("ax")


class ApplicationTests(unittest.TestCase):
    """
    Test for Application methods
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
        Tests simple application create and delete
        """
        a1 = Application("testapp")
        a1.create()
        a1.delete()

    def test_2(self):
        """
        Test assertion when application name exists
        """
        a1 = Application("testapp")
        a2 = Application("testapp")
        a1.create()
        with self.assertRaises(AXKubeApiException):
            a2.create()
        a1.delete()

    def test_3(self):
        """
        Test deletion of app that does not exists
        """
        a1 = Application("testappdoesnotexist")
        a1.delete()
