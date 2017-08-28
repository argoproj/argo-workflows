#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
import argparse
import logging
import requests
import urllib3
from retrying import retry


urllib3.disable_warnings()
requests.packages.urllib3.disable_warnings()

logger = logging.getLogger("axbootstrap_misc")

logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S")
logging.getLogger("ax").setLevel(logging.DEBUG)
logger.setLevel(logging.INFO)

"""
This helper stays as a temporary collection of miscellaneous
functions we need to ensure before bringing up platform.

This is used during upgrade time
"""


if __name__ == "__main__":
    # subcommands might be useful here but we don't want to over complicating things
    ax_parser = argparse.ArgumentParser(description="AX Bootstrap")
    ax_parser.add_argument("--customer-id", default=None, help="applatix customer id")
    ax_parser.add_argument("--aws-profile", default=None, help="AWS profile name")
    ax_parser.add_argument("--aws-region", default=None, help="Cluster aws region")
    ax_parser.add_argument("--cluster-name-id", help="Cluster name id")
    ax_parser.add_argument("--kubeconfig", default=None, help="Kubernetes config file path")
    ax_parser.add_argument("--ensure-aws-iam", default=False, action="store_true",
                           help="ensure aws iam settings")
    ax_parser.add_argument("--delete-aws-iam", default=False, action="store_true",
                           help="delete aws iam settings")
    ax_parser.add_argument("--ensure-aws-s3", default=False, action="store_true",
                           help="ensure aws iam settings")

    args = ax_parser.parse_args()
    if args.ensure_aws_iam or args.delete_aws_iam:
        from ax.platform.cluster_instance_profile import AXClusterInstanceProfile
        assert args.cluster_name_id, "Missing cluster name id to ensure aws iam"
        assert args.aws_region, "Missing AWS region to ensure aws iam"
        if args.ensure_aws_iam:
            AXClusterInstanceProfile(args.cluster_name_id, args.aws_region, aws_profile=args.aws_profile).update()
        elif args.delete_aws_iam:
            AXClusterInstanceProfile(args.cluster_name_id, args.aws_region, aws_profile=args.aws_profile).delete()

    if args.ensure_aws_s3:
        from ax.platform.cluster_buckets import AXClusterBuckets
        name_id = args.cluster_name_id
        aws_profile = args.aws_profile
        aws_region = args.aws_region
        assert name_id and aws_profile and aws_region, \
            "Missing parameters to ensure s3. name_id: {}, aws_profile: {}, aws_region: {}".format(name_id,
                                                                                                   aws_profile,
                                                                                                   aws_region)

        AXClusterBuckets(name_id, aws_profile, aws_region).update()

