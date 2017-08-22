#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import unittest

from ax.util.ax_network_address import find_fitting_subnet


class AXNetworkAddressTest(unittest.TestCase):
    @staticmethod
    def test_network_address():
        parent_network = "128.169.0.0/16"
        tests = [
            {
                "other_networks": ["128.169.1.0/17"],
                "subnet_prefixlen": 28,
                "result": "128.169.128.0/28"
            },
            {
                "other_networks": ["128.169.1.0/17", "128.169.130.3/32"],
                "subnet_prefixlen": 28,
                "result": "128.169.130.16/28"
            },
            {
                "other_networks": ["128.169.1.0/17", "128.169.128.0/18"],
                "subnet_prefixlen": 28,
                "result": "128.169.192.0/28"
            },
            {
                "other_networks": ["128.169.1.0/17", "128.169.128.0/18", "128.169.192.1/32", "128.169.192.17/32"],
                "subnet_prefixlen": 28,
                "result": "128.169.192.32/28"
            },
            # Fail with outside CIDR.
            {
                "other_networks": ["128.169.1.0/17", "10.0.0.0/24"],
                "subnet_prefixlen": 28,
                "result": ""
            },
            # Overlapping others should fail
            {
                "other_networks": ["128.169.1.0/17", "128.169.3.3/32"],
                "subnet_prefixlen": 28,
                "result": ""
            },
        ]
        for test in tests:
            other_subnets = test["other_networks"]
            result = find_fitting_subnet(parent_network, other_subnets, test["subnet_prefixlen"])

            assert result == test["result"], "test={}, result={}".format(test, result)

        for i in range(16, 21):
            n = []
            while True:
                a = find_fitting_subnet(parent_network, n, i)
                if a:
                    n.append(a)
                else:
                    break
            assert len(n) == 2 ** (i - 16), "len(n)={} i={} n={}".format(len(n), i, n)

        n = []
        for i in range(17, 33):
            a = find_fitting_subnet(parent_network, n, i)
            if a:
                n.append(a)
            else:
                break
        assert len(n) == 33 - 17, "len(n)={} n={}".format(len(n), n)

        n = []
        for i in range(32, 16, -1):
            a = find_fitting_subnet(parent_network, n, i)
            if a:
                n.append(a)
            else:
                break
        assert len(n) == 33 - 17, "len(n)={} n={}".format(len(n), n)
