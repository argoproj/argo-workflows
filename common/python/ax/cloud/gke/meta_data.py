#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import requests
from retrying import retry

class GCEMetaData(object):
    def __init__(self):
        self._meta_url = "http://169.254.169.254/0.1/meta-data/"
        self._network = None
        self._instance_id = None

    def get_security_groups(self):
        raise NotImplementedError("GCP")

    def get_region(self):
        raise NotImplementedError("GCP")

    def get_zone(self):
        raise NotImplementedError("GCP")

    def get_public_ip(self):
        return self._get_network()["networkInterface"][0]["accessConfiguration"][0]["externalIp"]

    def get_private_ip(self):
        return self._get_network()["networkInterface"][0]["ip"]

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_instance_id(self):
        if self._instance_id is None:
            self._instance_id = requests.get(self._meta_url + "instance-id").text.strip()
        return self._instance_id

    def get_instance_type(self):
        raise NotImplementedError("GCP")

    def get_user_data(self):
        raise NotImplementedError("GCP")

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def _get_network(self):
        # Sample data:
        # {
        #     "networkInterface": [
        #         {
        #             "accessConfiguration":
        #                 [
        #                     {
        #                         "externalIp": "35.185.238.217",
        #                         "type":"ONE_TO_ONE_NAT"
        #                     }
        #                 ],
        #             "ip": "10.138.0.5",
        #             "mac":"42:01:0a:8a:00:05",
        #             "network":"projects/956587539888/networks/default"
        #         }
        #     ]
        # }
        if self._network is None:
            self._network = requests.get(self._meta_url + "network").json()
            count = len(self._network["networkInterface"])
            assert count == 1, "Interface count {} {}".format(count, self._network)
        return self._network
