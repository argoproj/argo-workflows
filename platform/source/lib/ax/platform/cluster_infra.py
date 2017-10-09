#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
##

import json

from ax.kubernetes.client import KubernetesApiClient


def get_host_ip(kube_config=None):
    """
    Get's the IP address of the host in the cluster.
    """
    k8s = KubernetesApiClient(config_file=kube_config)
    resp = k8s.api.list_node()
    assert len(resp.items) == 1, "Need 1 node in the cluster"
    for n in resp.items:
        for addr in n.status.addresses:
            addr_dict = addr.to_dict()
            if addr_dict['type'] == 'InternalIP':
                return addr_dict['address']

    return None
