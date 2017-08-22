#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import requests
from retrying import retry

from ax.platform_client.env import AXEnv
from ax.util.retry_exception import AXRetry, ax_retry


class AWSMetaData(object):
    """
    Get AWS environment information from local user data or meta data.
    It can only run from inside AWS instance.

    Most application shouldn't use interfaces here.
    There are usually higher level interfaces to use in other places.
    """
    def __init__(self):
        assert AXEnv().is_in_pod() or AXEnv().on_kube_host()
        self._meta_url = "http://169.254.169.254/latest/meta-data/"
        self._user_url = "http://169.254.169.254/latest/user-data/"

    def get_security_groups(self):
        return requests.get(self._meta_url + "security-groups").text.strip()

    def get_region(self):
        url = self._meta_url + "placement/availability-zone"
        retry = AXRetry(retry_exception=(Exception,))
        r = ax_retry(requests.get, retry, url, timeout=10)
        return r.text[:-1]

    def get_zone(self):
        url = self._meta_url + "placement/availability-zone"
        retry = AXRetry(retry_exception=(Exception,))
        r = ax_retry(requests.get, retry, url, timeout=10)
        return r.text

    def get_public_ip(self):
        return requests.get(self._meta_url + "public-ipv4", timeout=3).text.strip()

    def get_instance_id(self):
        url = self._meta_url + "instance-id"
        retry = AXRetry(retries=3, delay=2, retry_exception=(Exception,))
        r = ax_retry(requests.get, retry, url, timeout=3)
        return r.text.strip()

    def get_instance_type(self):
        url = self._meta_url + "instance-type"
        retry = AXRetry(retries=3, delay=2, retry_exception=(Exception,))
        r = ax_retry(requests.get, retry, url, timeout=3)
        return r.text.strip()

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=3)
    def get_private_ip(self):
        return requests.get(self._meta_url + "local-ipv4", timeout=5).text.strip()

    def get_user_data(self, attr=None, plain_text=False):
        """
        Get AWS EC2 user data.
        :param attr: string: name of attribute if only one attribute is needed.
        :param plain_text: Is user data plain text or json.
        :return value of user data
        """
        retry = AXRetry(retry_exception=(Exception,))
        r = ax_retry(requests.get, retry, self._user_url, timeout=10)
        if r:
            if plain_text:
                return r.text
            else:
                if attr:
                    return r.json().get(attr, None)
                else:
                    return r.json()
