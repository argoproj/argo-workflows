#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

import argparse

from ax.version import __version__
from ax.aws.network import AWSNetwork


if __name__ == "__main__":
    import logging
    logging.basicConfig()
    logging.getLogger("ax").setLevel(logging.DEBUG)
    parser = argparse.ArgumentParser(description='search_subnet')
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    parser.add_argument('--aws-profile', help="AWS profile name.")
    parser.add_argument('vpc_id', help="AWS VPC ID")
    parser.add_argument('subnet_size', type=int, help="Subnet size, example 24")
    args = parser.parse_args()

    print(AWSNetwork(args.vpc_id, args.aws_profile).find_subnet(args.subnet_size))
