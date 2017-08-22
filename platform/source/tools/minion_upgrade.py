#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse
import logging

from ax.platform.ax_minion_upgrade import MinionUpgrade

if __name__ == "__main__":
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    parser = argparse.ArgumentParser(description="Minion upgrade")
    parser.add_argument("--profile", help="AWS profile")
    parser.add_argument("--region", help="AWS region")
    parser.add_argument("--new-kube-version", help="New kuberetes version")
    parser.add_argument("--new-cluster-install-version", help="New cluster install version")
    parser.add_argument("--new-kube-server-hash", help="New kuberetes server hash")
    parser.add_argument("--new-kube-salt-hash", help="New kuberetes salt hash")
    parser.add_argument("--ax-vol-disk-type", help="Type of the AX volume disk (gp2/io1/etc.)", default="gp2")
    parser.add_argument("--retain-spot-price", action="store_true", help="Whether to retain the spot-instance bid price", default=False)
    parser.add_argument("--cluster-name-id", help="NameId of the cluster")

    args = parser.parse_args()

    m = MinionUpgrade(new_kube_version=args.new_kube_version,
                      new_cluster_install_version=args.new_cluster_install_version,
                      new_kube_server_hash=args.new_kube_server_hash,
                      new_kube_salt_hash=args.new_kube_salt_hash,
                      profile=args.profile,
                      region=args.region,
                      ax_vol_disk_type=args.ax_vol_disk_type,
                      cluster_name_id=args.cluster_name_id)
    m.update_all_launch_configs(args.retain_spot_price)
