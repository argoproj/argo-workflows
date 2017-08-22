#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import json
import unittest

from ax.platform.testutils.kubectl_wrapper import KubectlWrapper

class AXMonTests(unittest.TestCase):
    """
    Assumes that your environment has the following:
    1. A valid version of kubectl (one that matches the server)
    2. kubectl is in your run path
    3. minikube based VM already running and set to your default context
    """
    @classmethod
    def setUpClass(self):
        print("Creating axmon as a service")
        assert(KubectlWrapper().run_command("../containers/axmon/axmon-svc.yml", "create") is True)

        print("Wait for axmon service to start")
        assert(KubectlWrapper().wait_for_service("axmon") is True)

        print("Wait for axmon deployment to start")
        assert(KubectlWrapper().wait_for_pod_status("Running", selector="app=axmon-deployment") is True)

    @classmethod
    def tearDownClass(self):
        print("Deleting axmon")
        assert(KubectlWrapper().run_command("../containers/axmon/axmon-svc.yml", "delete") is True)

    def test_ping(self):
        """
        AXMon bring-up and ping test without portaldb
        """
        op = KubectlWrapper().run_one_time_command("tutum/curl", "curl", "curl", "-ss", "axmon:8901/v1/axmon/ping")
        self.assertNotEqual(op, None)
        d = json.loads(op)
        self.assertEqual(d["status"], "OK")
