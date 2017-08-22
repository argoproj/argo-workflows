#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse
import logging

from ax.platform.ax_cluster_config_upgrade import ClusterConfigUpgrade

if __name__ == "__main__":
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    parser = argparse.ArgumentParser(description="Minion upgrade")
    parser.add_argument("--profile", help="AWS profile")
    parser.add_argument("--region", help="AWS region")
    parser.add_argument("--cluster-name-id", help="Name-ID of the cluster")

    args = parser.parse_args()

    cc = ClusterConfigUpgrade(profile=args.profile,
                              region=args.region,
                              cluster_name_id=args.cluster_name_id)
    cc.update_cluster_config()
