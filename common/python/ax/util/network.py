#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module to get local network information.
"""

import json
import logging
import os
import requests
import socket
try:
    from urllib.parse import urlparse
except ImportError:
    from urlparse import urlparse

from ax.util.cloud import cloud_provider, CLOUD_AWS, CLOUD_VBOX

logger = logging.getLogger(__name__)

def get_own_ip():
    """
    Get local default IP address. This is POD IP if it's inside.

    :return: IP address in string or None if failed.
    """
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        s.connect((socket.gethostbyname("google.com"), 0))
        ip = s.getsockname()[0]
    except Exception:
        ip = None
    s.close()
    return ip


def get_own_name(fullname=False):
    """
    Get local hostname. This is pod hostname if it's inside.

    :return: short hostname by default, or full name if fullname is True
    """
    if fullname:
        return socket.gethostname()
    else:
        return socket.gethostname().split(".")[0]


def get_public_ip():
    """
    Get public IP
    Currently only applicable to EC2 instances.
    Get IP from meta data.
    :return:
    """
    if cloud_provider() == CLOUD_AWS:
        from ax.aws.meta_data import AWSMetaData
        try:
            return AWSMetaData().get_public_ip()
        except:
            pass

    return get_public_ip_through_external_server()


def get_public_ip_through_external_server():
    ip_servers = [["https://httpbin.org/ip", "origin"],
                  ["http://httpbin.org/ip", "origin"],
                  ["http://ipinfo.io", "ip"],
                  ["https://api.ipify.org?format=json", "ip"],
                  ["http://api.ipify.org?format=json", "ip"],
                  ["https://yourapihere.com/", "origin"]
                  ]
    ex = None
    for server in ip_servers:
        try:
            server_url = server[0]
            resp = requests.get(url=server_url, timeout=15)
            conf = json.loads(resp.text)
            ip = conf[server[1]]
            if ip is not None:
                return ip
            else:
                raise Exception("bad reply")
        except Exception as e:
            logger.exception("server %s has problem", server_url)
            ex = e
    # cannot find the ip after try all servers, raise the last exception
    raise ex


def cluster_service_url_translate(in_container_url, cluster_id):
    try:
        parsed = urlparse(in_container_url)
        service_name = parsed.hostname
        user_pass = ""
        if parsed.username is not None:
            user_pass += parsed.username
        if parsed.password is not None:
            user_pass += ":" + parsed.password
        if parsed.username is not None or parsed.password is not None:
            user_pass += "@"
        replaced = parsed._replace(netloc="{}{}:{}".format(user_pass, service_name, parsed.port))
        return replaced.geturl()
    except Exception:
        logger.exception("cannot translate %s %s %s", in_container_url, cluster_id, dns_server)
        return None
