#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

#!/usr/bin/env python

"""The main entry point for the minion-manager."""

import argparse
import logging
import os
import sys


from ax.cloud import Cloud
from ax.platform.minion_manager.cloud_broker import Broker

logger = logging.getLogger("minion_manager")
logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s " +
                    "%(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S",
                    stream=sys.stdout, level=logging.INFO)

def validate_usr_args(usr_args):
    """
    Validates the arguments provided by the user.
    """
    assert usr_args.cloud.lower() == Cloud.CLOUD_AWS, "Only AWS is currently supported."

def set_missing_params(usr_args):
    if not usr_args.region:
        usr_args.region = os.environ.get("MM_REGION", None)
        logger.info("Using config from env: %s", usr_args.region)

    if not usr_args.scaling_groups:
        groups = os.environ.get("MM_SCALING_GROUPS", None)
        usr_args.scaling_groups = groups and groups.split() or None
        logger.info("Using config from env: %s", usr_args.scaling_groups)

    if not usr_args.profile:
        usr_args.profile = None

    if not usr_args.monitor_minions:
        usr_args.monitor_minions = os.environ.get("MM_MONITOR_MINIONS", True)
        logger.info("Using config from env: %s", usr_args.monitor_minions)

def run():
    """
    Parses user provided arguments and validates them. Asserts if any of
    the provided arguments is incorrect.
    """
    parser = argparse.ArgumentParser(description="Manage the minions in a " +
                                     "K8S cluster")
    parser.add_argument('--version', action='version', version='%(prog)s')
    parser.add_argument('--scaling-groups', nargs="+",
                        help="Names of the scaling groups to manage")
    parser.add_argument("--region", help="Region in which the cluster exists")
    parser.add_argument("--cloud", default=Cloud.CLOUD_AWS,
                        help="Cloud provider (only AWS as of now)")
    parser.add_argument("--profile", help="Credentials profile to use")
    parser.add_argument("--monitor-minions", default=True,
                        action="store_true",
                        help="Check if nodes are 'Ready' and terminate if not")

    usr_args = parser.parse_args()
    set_missing_params(usr_args)
    validate_usr_args(usr_args)
    logger.info("Starting minion-manager for scaling groups: %s, in region " +
                "%s for cloud provider %s", usr_args.scaling_groups,
                usr_args.region, usr_args.cloud)

    if usr_args.cloud == Cloud.CLOUD_AWS:
        minion_manager = Broker.get_impl_object(
            usr_args.cloud, usr_args.scaling_groups, usr_args.region,
            aws_profile=usr_args.profile,
            monitor_minions=usr_args.monitor_minions)
        minion_manager.run()

    while True:
        import time
        logger.info("Running ...")
        time.sleep(10)

# A journey of a thousand miles ...
if __name__ == "__main__":
    logger.info("Starting ...")
    run()
