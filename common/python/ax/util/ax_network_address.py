#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Module for network address util.
"""

import logging

from netaddr import IPNetwork, IPAddress, iprange_to_cidrs

logger = logging.getLogger(__name__)


def find_fitting_subnet(parent_network, other_subnets, subnet_prefixlen):
    """
    Get the smallest subnet that is bigger or equals to subnet described by subnet_prefixlen,
    the subnet must be in the parent_network and cannot overlap with any of the other_subnets
    :param parent_network (str): the parent network
    :param other_subnets (list of str): list of other subnets that are already in the parent_network
    :param subnet_prefixlen (int): the subnet_size prefix, e.g. 24 for 255.255.255.0

    :return (str): the result IP subnet or ""
    """
    # Parse incoming strings to IPNetwork objects. Sort other subnet list.
    try:
        parent = IPNetwork(parent_network)
        others = sorted([IPNetwork(n) for n in other_subnets])
    except Exception:
        logger.exception("parent_network: %s, subnet_prefixlen: %s", parent_network, subnet_prefixlen)
        return ""

    # Check whether any subnet that is not in parent's range.
    for n in others:
        if n.first < parent.first or n.last > parent.last:
            logger.error("Subnets not in parent range %s %s", parent, other_subnets)
            return ""

    boundaries = [parent.first]
    for n in others:
        if n.first < boundaries[-1]:
            logger.error("Overlapping subnets others %s, offending %s", others, n)
            return ""
        boundaries += [n.first - 1, n.last + 1]
    boundaries += [parent.last]

    candidate_prefixlen = 0
    candidate_subnet = None
    for i in range(0, len(boundaries), 2):
        start = boundaries[i]
        end = boundaries[i + 1]
        if start <= end:
            # have to work around a bug
            # netaddr.iprange_to_cidrs('192.169.0.1', '192.168.255.255')
            splitted = iprange_to_cidrs(IPAddress(start), IPAddress(end))
            logger.debug("splitted %s", splitted)
            for n in splitted:
                # find the smallest network that fits
                if n.prefixlen <= subnet_prefixlen and \
                        (candidate_subnet is None or n.prefixlen > candidate_prefixlen):
                    logger.debug("candidate_subnet %s", n)
                    candidate_prefixlen = n.prefixlen
                    candidate_subnet = n

    return str(list(candidate_subnet.subnet(subnet_prefixlen, count=1))[0].cidr) if candidate_subnet is not None else ""

