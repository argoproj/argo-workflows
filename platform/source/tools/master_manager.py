#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
import logging
import argparse
from ax.platform.ax_master_manager import AXMasterManager
from ax.version import __version__

logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S")
logging.getLogger("ax").setLevel(logging.DEBUG)

def run():
    parser = argparse.ArgumentParser(description="Start a new K8S master")
    parser.add_argument("cluster_name_id", help="Name of the cluster")
    parser.add_argument("command", help="Command, server or upgrade")
    parser.add_argument("--region", help="Region name")
    parser.add_argument("--profile", help="Profile name")
    parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))
    usr_args = parser.parse_args()

    m = AXMasterManager(usr_args.cluster_name_id, profile=usr_args.profile, region=usr_args.region)
    if usr_args.command == "server":
        m.run()
    elif usr_args.command == "upgrade":
        m.upgrade()

if __name__ == "__main__":
    run()
