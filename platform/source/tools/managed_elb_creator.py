#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from gevent import monkey
monkey.patch_all()

import argparse
import requests
import sys

from retrying import retry

from ax.version import __version__

from ax.cloud.aws.server_cert import ServerCert
from ax.util.hash import generate_hash


@retry(wait_exponential_multiplier=1000, stop_max_attempt_number=10)
def create_elb(name, arn, private):
    print ("Trying to create a {} ELB {}".format("Private" if private else "Public", name))
    url = "http://axmon.axsys:8901/v1/axmon/managed_elb"
    dep = 'ingress-controller-int-deployment' if private else 'ingress-controller-deployment'
    data = {
        'name': name,
        'application': 'axsys',
        'deployment': dep,
        'deployment_selector': {
            'app': dep,
            'role': 'axcritical',
            'tier': 'platform'
        },
        'type': 'internal' if private else 'external',
        'ports': [
            {"listen_port": 80, "container_port": 80, "protocol": "http"},
            {"listen_port": 443, "container_port": 80, "protocol": "https", "certificate": arn}
        ]
    }

    # timeout is long as this is longer operation (several minutes)
    response = requests.post(url=url, json=data, timeout=15*60)
    print("Response is {}".format(response.json()))
    response.raise_for_status()
    return response.json()['result']


@retry(wait_exponential_multiplier=1000, stop_max_attempt_number=10)
def delete_elb(name):
    print("Deleting ELB {}".format(name))
    url = "http://axmon.axsys:8901/v1/axmon/managed_elb/{}".format(name)
    response = requests.delete(url=url, timeout=15*60)
    print("Response is {}".format(response.json()))
    if response.status_code == 404:
        return
    response.raise_for_status()


@retry(wait_exponential_multiplier=1000, stop_max_attempt_number=10)
def list_elbs():
    print("Trying to list all ELBs")
    url = "http://axmon.axsys:8901/v1/axmon/managed_elb"
    response = requests.get(url=url, timeout=2*60)
    print("Response is {}".format(response.json()))
    response.raise_for_status()
    return response.json()['result']


@retry(wait_exponential_multiplier=1000, stop_max_attempt_number=10)
def get_name_id():
    print("Trying to get cluster name id")
    url = "http://axmon.axsys:8901/v1/axmon/portal"
    response = requests.get(url=url, timeout=30)
    response.raise_for_status()
    print("Response is {}".format(response.json()))
    return response.json()['cluster_name_id']


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Managed ELB creator')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--private', action='store_true')              # default to true
    parser.add_argument('--force', action='store_true')                 # default to false
    parser.add_argument('--delete', action='store_true')
    args = parser.parse_args()

    # generate server certificate
    name_id = get_name_id()
    if not name_id:
        raise ValueError("Cannot get the name and id of cluster. Please report this error to Applatix.")

    suffix = "ing-pri" if args.private else "ing-pub"
    unique_name = generate_hash("{}-{}".format(name_id, suffix))[:32]
    print("Unique name for ELB using cluster name and id: {}".format(unique_name))
    cert = ServerCert(unique_name)
    if not args.delete:
        print("Generating cert for ELB")
        arn = cert.generate_certificate()

    if args.delete:
        delete_elb(unique_name)
        print("ELB {} deleted successfully".format(unique_name))
        print("Deleting cert for ELB")
        cert.delete_certificate()
        sys.exit(0)

    if not args.force and unique_name in list_elbs():
        print("Not creating ELB {} as it already exists. Use the --force flag to recreate it")
        sys.exit(0)

    elb_name = create_elb(unique_name, arn, args.private)
    print("ELB {} created successfully".format(elb_name))

    sys.exit(0)
