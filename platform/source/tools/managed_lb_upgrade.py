#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from gevent import monkey
monkey.patch_all()

import boto3
import argparse
import requests
import sys

from retrying import retry

from ax.version import __version__
from ax.platform.cloudprovider.aws.route53 import Route53, Route53HostedZone
from ax.platform.routes import ServiceEndpoint
from ax.cloud.aws.elb import visibility_to_elb_name, visibility_to_elb_addr, ExternalRouteVisibility
from ax.kubernetes.client import KubernetesApiClient, KubernetesApiClientWrapper

"""
This script migrates entries from managed domains from old kubernetes created elb 
to the ELB created as part of managed LB.
"""

# helper function to get the name of the old elb
def get_k8s_elb(client):
    ing_service = ServiceEndpoint("ingress-controller-svc", namespace="axsys", client=client)
    if not ing_service.exists():
        return None
    addrs = ing_service.get_addrs()
    assert len(addrs) == 1, "Need 1 address for ingress-controller-svc but found {}".format(addrs)
    return addrs[0]


# helper function to delete the old elb and old service object
def delete_k8s_svc(client):
    ing_service = ServiceEndpoint("ingress-controller-svc", namespace="axsys", client=client)
    ing_service.delete()


# helper function to get new elb addr and new elb name
# this function will raise an exception if new elb is not found.
def get_new_elb_info():
    name = visibility_to_elb_name(ExternalRouteVisibility.VISIBILITY_WORLD)
    addr = visibility_to_elb_addr(ExternalRouteVisibility.VISIBILITY_WORLD)
    return name, addr


# helper function to get managed domains
def get_managed_domains():
    # Code to parse the tools json for domains.

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=10)
    def get_tools_from_axops():
        response = requests.get(url="http://axops-internal.axsys:8085/v1/tools")
        response.raise_for_status()
        return response.json()

    # TODO: Use generated lib for axops
    tools = get_tools_from_axops()
    for tool in tools.get("data", []):
        if tool["category"] == "domain_management" and tool["type"] == "route53":
            return [x["name"] for x in tool.get("domains", [])]

    return None


def change_elb(subdomain, old_elb, new_elb_name, new_elb_addr, client):
    print("Looking for records in {} that point to {}".format(subdomain, old_elb))
    zone = Route53HostedZone(client, subdomain)
    for record in zone.list_records():
        if 'AliasTarget' in record and old_elb in record['AliasTarget']['DNSName']:
            print ("UPDATING RECORD: {}".format(record))
            zone.create_alias_record(record['Name'].partition(".")[0], new_elb_addr, elb_name=new_elb_name)
            print("Record {} updated to point to new elb".format(record['Name']))


def exit_all_containers():
    pass

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Upgrade ELB')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    args = parser.parse_args()

    client = KubernetesApiClient()
    old_elb_addr = get_k8s_elb(client)
    if not old_elb_addr:
        print("Did not find an the old ELB from ingress controller. Aborting")
        sys.exit(0)

    name, addr = get_new_elb_info()
    print("Old ELB {}, New Elb {} {}".format(old_elb_addr, name, addr))

    subdomains = get_managed_domains()
    r53client = Route53(boto3.client("route53"))
    for x in subdomains or []:
        change_elb(x, old_elb_addr, name, addr, r53client)

    print("Deleting old ingress controller elb and service object")
    delete_k8s_svc(client)
