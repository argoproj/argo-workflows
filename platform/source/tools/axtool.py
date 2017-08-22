#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Back end client tools for AX platform.

Axtool.py is tools used by axmon in back end.
  It can import some internal AX modules and shouldn't be distributed to customer.
"""

import argparse
import json
import logging
import re
import os
import sys

from ax.util.const import COLOR_NORM, COLOR_RED
from ax.version import __version__
from ax.cloud import Cloud

logger = logging.getLogger('ax.axtool')
logging.getLogger('ax.platform.ax_monitor').setLevel(logging.INFO)

def _add_arguments(parser, positional=[], optional=[]):
    """Adds some common positional and optional arguments to the parser"""
    if set(positional) & set(optional):
        raise ValueError("Argument(s) {} cannot be both positional and optional"
                         .format(list(set(positional) & set(optional))))

    param_defs = {
        'cluster-name' : {'help' : "Name of cluster", 'default': os.getenv('AX_CLUSTER_NAME', None)},
        'cluster-config' : {'help': "AX cluster config file"},
        'image-namespace' : {'help' : "Image namespace",
                             'default' : os.getenv("AX_NAMESPACE")},
        'image-version' : {'help' : "Image version",
                           'default' : os.getenv("AX_VERSION")},
        'cluster-install-version' : {'help' : "Version for cluster install tools (kube-up)",
                           'default' : os.getenv("CLUSTER_INSTALL_VERSION")},
        'object-name' : {'help' : "Kubernetes object name"},
        'dry-run': {'help': "Dry run", 'action': 'store_true', 'default': False},
        'aws-profile': {'help': "AWS profile name", 'default': os.getenv('AWS_DEFAULT_PROFILE', None)},
        'volume-only': {'help': "Create or delete volumes (PVC) only", 'action': 'store_true', 'default': False},
        'kube-version': {'help': "Kubernetes version", 'default': os.getenv('AX_KUBE_VERSION')},
        'target-cloud': {'help': "Target cloud, aws or gcp", 'default': os.getenv('AX_TARGET_CLOUD', 'aws')},
        'debug': {'help': "Don't delete object if error occurs during creation", 'action': 'store_true', 'default': False}
    }
    for param in positional:
        # argparse's dash-to-underscore replacement does not work for positional arguments.
        # http://stackoverflow.com/questions/12834785/having-options-in-argparse-with-a-dash
        parser.add_argument(param.replace('-', '_'),
                            metavar=param,
                            **(param_defs[param]))
    for param in optional:
        parser.add_argument('--' + param, **(param_defs[param]))


class AxTool(object):

    def __init__(self):
        parser = argparse.ArgumentParser(description='AX tools',
                                         formatter_class=argparse.ArgumentDefaultsHelpFormatter)
        subparsers = parser.add_subparsers(dest="command")
        parser.add_argument('--version', action='version', version="%(prog)s {}".format(__version__))

        kube_parser = subparsers.add_parser('kubernetes', help="Kubernetes commands")
        kube_subparsers = kube_parser.add_subparsers(dest='subcommand')
        kube_create_parser = kube_subparsers.add_parser('create', help="Create a new kubernetes object.")
        _add_arguments(kube_create_parser, positional=['object-name'],
                       optional=['cluster-name', 'image-namespace', 'image-version', 'kube-version', 'aws-profile'])
        kube_delete_parser = kube_subparsers.add_parser('delete', help="Delete a kubernetes object.")
        _add_arguments(kube_delete_parser, positional=['object-name'],
                       optional=['cluster-name', 'image-namespace', 'image-version', 'kube-version', 'aws-profile'])

        platform_parser = subparsers.add_parser('platform', help="Platform commands")
        platform_subparsers = platform_parser.add_subparsers(dest='subcommand')

        # Install & cleanup
        platform_create_parser = platform_subparsers.add_parser('install', help="Platform install. Create resources in cluster.")
        _add_arguments(platform_create_parser, optional=['cluster-name', 'volume-only', 'image-namespace', 'image-version', 'kube-version', 'cluster-install-version', 'aws-profile', 'target-cloud', 'debug'])
        platform_delete_parser = platform_subparsers.add_parser('cleanup', help="Platform cleanup. Delete resources from cluster")
        _add_arguments(platform_delete_parser, optional=['cluster-name', 'volume-only', 'aws-profile', 'target-cloud'])

        # Create & delete
        platform_create_parser = platform_subparsers.add_parser('create', help="Platform creation. Create resources in cluster.")
        _add_arguments(platform_create_parser, optional=['cluster-name', 'volume-only', 'image-namespace', 'image-version', 'kube-version', 'cluster-install-version', 'aws-profile', 'target-cloud', 'debug'])
        platform_delete_parser = platform_subparsers.add_parser('delete', help="Platform deleteion. Delete resources from cluster")
        _add_arguments(platform_delete_parser, optional=['cluster-name', 'volume-only', 'aws-profile', 'target-cloud'])

        # Start & stop
        platform_start_parser = platform_subparsers.add_parser('start', help="Platform start. Create services")
        _add_arguments(platform_start_parser, optional=['cluster-name', 'image-namespace', 'image-version', 'kube-version', 'cluster-install-version', 'aws-profile', 'target-cloud', 'debug'])
        platform_stop_parser = platform_subparsers.add_parser('stop', help="Platform stop. Stop services.")
        _add_arguments(platform_stop_parser, optional=['cluster-name', 'aws-profile', 'target-cloud'])

        # Upgrade, reset, & renaissance
        platform_upgrade_parser = platform_subparsers.add_parser('upgrade', help="Platform upgrade. Restart all services")
        _add_arguments(platform_upgrade_parser, optional=['cluster-name', 'image-namespace', 'image-version', 'kube-version', 'aws-profile', 'target-cloud', 'debug'])
        platform_reset_parser = platform_subparsers.add_parser('reset', help="Platform factory reset. Restart all services and remove all data.")
        _add_arguments(platform_reset_parser, optional=['cluster-name', 'image-namespace', 'image-version', 'kube-version', 'aws-profile', 'target-cloud', 'debug'])
        platform_reset_parser = platform_subparsers.add_parser('renaissance', help="Recreate every applatix component")
        _add_arguments(platform_reset_parser, optional=['cluster-name', 'image-namespace', 'image-version', 'kube-version', 'aws-profile', 'target-cloud', 'debug'])

        cluster_parser = subparsers.add_parser('cluster', help="Cluster commands")
        cluster_subparsers = cluster_parser.add_subparsers(dest='subcommand')

        # Cluster show and download
        parser_cluster_show = cluster_subparsers.add_parser('show', help="Show cluster information.")
        _add_arguments(parser_cluster_show, positional=['cluster-name'], optional=['aws-profile'])
        parser_cluster_download_config = cluster_subparsers.add_parser('download-config', help="Download config.")
        _add_arguments(parser_cluster_download_config, positional=['cluster-name'], optional=['aws-profile', 'target-cloud'])

        cron_parser = subparsers.add_parser('cron', help="Cron commands")
        cron_parser.add_argument('--run-once', action='store_true', help="Run cron commands once and exit.")

        args = parser.parse_args()
        print("axtool", __version__, args)

        if not args.command:
            parser.print_help()
            return
        if hasattr(args, 'subcommand') and not args.subcommand:
            locals()['{}_parser'.format(args.command)].print_help()
            return

        try:
            getattr(self, args.command)(args)
        except NotImplementedError as e:
            parser.error(e)

    def kubernetes(self, args):
        from ax.platform.platform import AXPlatform
        from ax.meta import AXClusterId
        from ax.platform_client.env import AXEnv

        assert AXEnv().is_in_pod() or args.cluster_name, "Must specify cluster name from outside cluster"
        name_id = AXClusterId(args.cluster_name, args.aws_profile).get_cluster_name_id()
        plat = AXPlatform(cluster_name_id=name_id, aws_profile=args.aws_profile)
        if args.subcommand == 'create':
            plat.start_one(args.object_name)

        elif args.subcommand == 'delete':
            plat.stop_one(args.object_name)

    def platform(self, args):
        from ax.platform.platform import AXPlatform
        from ax.meta import AXClusterId
        from ax.platform_client.env import AXEnv

        Cloud().set_target_cloud(args.target_cloud)

        assert AXEnv().is_in_pod() or args.cluster_name, "Must specify cluster name from outside cluster"
        name_id = AXClusterId(args.cluster_name, args.aws_profile).get_cluster_name_id()
        portal_url = os.getenv("PORTAL_URL", None)
        if args.subcommand == 'start':
            AXPlatform(cluster_name_id=name_id, aws_profile=args.aws_profile, debug=args.debug, portal_url=portal_url).start()
        elif args.subcommand == 'stop':
            AXPlatform(cluster_name_id=name_id, aws_profile=args.aws_profile).stop()

        else:
            logger.error("%sInvalid command '%s'%s", COLOR_RED, COLOR_NORM)
            sys.exit(1)

    def cluster(self, args):
        from ax.platform.ax_cluster_info import AXClusterInfo
        from ax.meta import AXClusterId
        from ax.platform_client.env import AXEnv

        Cloud().set_target_cloud(args.target_cloud)

        assert AXEnv().is_in_pod() or args.cluster_name, "Must specify cluster name from outside cluster"

        if args.subcommand in ['start', 'create']:
            logger.error("=" * 80)
            logger.error("axtool cluster start/create has be moved to axinstaller")
            logger.error("=" * 80)
            sys.exit(1)
        elif args.subcommand in ['stop', 'delete']:
            logger.error("=" * 80)
            logger.error("axtool cluster stop/delete has be moved to axinstaller")
            logger.error("=" * 80)
            sys.exit(1)
        elif args.subcommand == 'show':
            import subprocess
            name_id = AXClusterId(args.cluster_name, args.aws_profile).get_cluster_name_id()
            AXClusterInfo(name_id, aws_profile=args.aws_profile).download_kube_key()
            conf_file = AXClusterInfo(name_id, aws_profile=args.aws_profile).download_kube_config()
            logger.info("Kubeconfig")
            with open(conf_file, "r") as f:
                conf = f.read()
            logger.info("%s", conf)
            subprocess.call(["kubectl", "--kubeconfig", conf_file, "cluster-info"])
            subprocess.call(["kubectl", "--kubeconfig", conf_file, "get", "no"])
            subprocess.call(["kubectl", "--kubeconfig", conf_file, "--namespace", "axsys", "get", "po"])
        elif args.subcommand == 'download-config':
            name_id = AXClusterId(args.cluster_name, args.aws_profile).get_cluster_name_id()
            if Cloud().target_cloud_aws():
                AXClusterInfo(name_id, aws_profile=args.aws_profile).download_kube_key()
            AXClusterInfo(name_id, aws_profile=args.aws_profile).download_kube_config()

    def cron(self, args):
        from ax.platform.cron import AXCron
        c = AXCron()
        if args.run_once:
            c.run_once()
        else:
            c.start()


if __name__ == "__main__":
    logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                        datefmt="%Y-%m-%dT%H:%M:%S")
    logging.getLogger("ax").setLevel(logging.DEBUG)
    AxTool()
