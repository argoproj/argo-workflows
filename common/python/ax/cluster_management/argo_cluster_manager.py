# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import argparse
import logging
import os
import sys
import uuid

from ax.cloud import Cloud
from ax.cloud.aws import SecurityToken
from ax.util.const import COLOR_NORM, COLOR_RED
from .app import ClusterInstaller, ClusterPauser, ClusterResumer, ClusterUninstaller, ClusterUpgrader, \
    CommonClusterOperations
from .app.options import add_install_flags, add_platform_only_flags, ClusterInstallConfig, add_pause_flags, ClusterPauseConfig, \
    add_restart_flags, ClusterRestartConfig, add_uninstall_flags, ClusterUninstallConfig, \
    add_upgrade_flags, ClusterUpgradeConfig, add_misc_flags, ClusterMiscOperationConfig

logger = logging.getLogger(__name__)


class ArgoClusterManager(object):
    def __init__(self):
        self._parser = None

    def add_flags(self):
        self._parser = argparse.ArgumentParser(description="Argo cluster management",
                                               formatter_class=argparse.ArgumentDefaultsHelpFormatter)
        main_subparser = self._parser.add_subparsers(dest="command")

        # Add install cluster flags
        install_parser = main_subparser.add_parser("install", help="Install Argo cluster")
        add_install_flags(install_parser)

        # Add pause cluster flags
        pause_parser = main_subparser.add_parser("pause", help="Pause Argo cluster")
        add_pause_flags(pause_parser)

        # Add restart cluster flags
        restart_parser = main_subparser.add_parser("resume", help="Resume Argo cluster")
        add_restart_flags(restart_parser)

        # Add uninstall cluster flags
        uninstall_parser = main_subparser.add_parser("uninstall", help="Uninstall Argo cluster")
        add_uninstall_flags(uninstall_parser)

        # Add upgrade cluster flags
        upgrade_parser = main_subparser.add_parser("upgrade", help="Upgrade Argo cluster")
        add_upgrade_flags(upgrade_parser)

        # Add download credential flags
        download_cred_parser = main_subparser.add_parser("download-cluster-credentials", help="Download Argo cluster credentials")
        add_misc_flags(download_cred_parser)

        # Install on existing cluster
        platform_only_installer = main_subparser.add_parser("install-platform-only", help="Install platform only")
        add_platform_only_flags(platform_only_installer)

    def parse_args_and_run(self):
        assert isinstance(self._parser, argparse.ArgumentParser), "Please call add_flags() to initialize parser"
        args = self._parser.parse_args()
        if not args.command:
            self._parser.print_help()
            return

        try:
            cmd = args.command.replace("-", "_")
            getattr(self, cmd)(args)
        except NotImplementedError as e:
            self._parser.error(e)
        except Exception as e:
            logger.exception(e)
            print("\n{} !!! Operation failed due to runtime error: {} {}\n".format(COLOR_RED, e, COLOR_NORM))

    def install(self, args):
        install_config = ClusterInstallConfig(cfg=args)
        install_config.default_or_wizard()
        err = install_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(install_config.cloud_profile)
        ClusterInstaller(install_config).start()

    def pause(self, args):
        pause_config = ClusterPauseConfig(cfg=args)
        pause_config.default_or_wizard()
        err = pause_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(pause_config.cloud_profile)
        ClusterPauser(pause_config).start()

    def resume(self, args):
        resume_config = ClusterRestartConfig(cfg=args)
        resume_config.default_or_wizard()
        err = resume_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(resume_config.cloud_profile)
        ClusterResumer(resume_config).start()

    def uninstall(self, args):
        uninstall_config = ClusterUninstallConfig(cfg=args)
        uninstall_config.default_or_wizard()
        err = uninstall_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(uninstall_config.cloud_profile)
        ClusterUninstaller(uninstall_config).start()

    def download_cluster_credentials(self, args):
        config = ClusterMiscOperationConfig(cfg=args)
        config.default_or_wizard()
        err = config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(config.cloud_profile)
        if config.dry_run:
            logger.info("DRY RUN: downloading credentials for cluster %s.", config.cluster_name)
            return
        ops = CommonClusterOperations(
            input_name=config.cluster_name,
            cloud_profile=config.cloud_profile
        )
        ops.cluster_info.download_kube_config()
        ops.cluster_info.download_kube_key()

    def upgrade(self, args):
        upgrade_config = ClusterUpgradeConfig(cfg=args)
        upgrade_config.default_or_wizard()
        err = upgrade_config.validate()
        self._continue_or_die(err)
        self._ensure_customer_id(upgrade_config.cloud_profile)
        ClusterUpgrader(upgrade_config).start()

    def install_platform_only(self, args):
        # TODO(shri): Add more meat!
        logger.info("Installing platform only using args: %s", args)
        return

    @staticmethod
    def _ensure_customer_id(cloud_profile):
        if os.getenv("AX_CUSTOMER_ID", None):
            logger.info("Using customer ID %s", os.getenv("AX_CUSTOMER_ID"))
            return

        # TODO (#111): set customer id to GCP
        if Cloud().target_cloud_aws():
            account_info = SecurityToken(aws_profile=cloud_profile).get_caller_identity()
            customer_id = str(uuid.uuid5(uuid.NAMESPACE_OID, account_info["Account"]))
            logger.info("Using AWS account ID hash (%s) for customer id", customer_id)
            os.environ["AX_CUSTOMER_ID"] = customer_id

    @staticmethod
    def _continue_or_die(err):
        if err:
            print("\n{}====== Errors:\n".format(COLOR_RED))
            for e in err:
                print(e)
            print("\n!!! Operation failed due to invalid inputs{}\n".format(COLOR_NORM))
            sys.exit(1)
