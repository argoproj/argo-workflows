# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


"""
Module for pausing an Argo cluster
"""


import logging
import os
import subprocess
import sys
import yaml
from pprint import pformat

from ax.cloud import Cloud
from ax.meta import AXCustomerId
from ax.platform.ax_cluster_info import AXClusterInfo
from ax.platform.bootstrap import AXBootstrap
from ax.platform.cluster_config import AXClusterConfig
from ax.platform.component_config import SoftwareInfo
from ax.platform.platform import AXPlatform
from ax.util.const import COLOR_GREEN, COLOR_YELLOW, COLOR_NORM
from ax.util.network import get_public_ip

from .common import ClusterOperationBase, check_cluster_staging, ensure_manifest_temp_dir,\
    TEMP_PLATFORM_MANIFEST_ROOT, TEMP_PLATFORM_CONFIG_PATH, is_portal_env
from .options import ClusterUpgradeConfig

logger = logging.getLogger(__name__)


class ClusterUpgrader(ClusterOperationBase):
    def __init__(self, cfg):
        assert isinstance(cfg, ClusterUpgradeConfig)
        self._cfg = cfg
        super(ClusterUpgrader, self).__init__(
            cluster_name=self._cfg.cluster_name,
            cluster_id=self._cfg.cluster_id,
            cloud_profile=self._cfg.cloud_profile,
            dry_run=self._cfg.dry_run
        )

        # This will raise exception if name/id mapping cannot be found
        self._name_id = self._idobj.get_cluster_name_id()
        self._cluster_info = AXClusterInfo(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile
        )
        self._cluster_config = AXClusterConfig(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile
        )
        self._bootstrap_obj = AXBootstrap(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            region=self._cluster_config.get_region()
        )
        self._current_software_info = SoftwareInfo(
            info_dict=yaml.load(
                self._cluster_info.download_cluster_software_info()
            )
        )
        self._cidr = str(get_public_ip()) + "/32"
        self._upgrade_kube_needed = True
        self._upgrade_service_needed = True

    def pre_run(self):
        if not is_portal_env() and self._cluster_info.is_cluster_supported_by_portal():
            raise RuntimeError("Cluster is currently supported by portal. Please login to portal to perform cluster management operations.")

        if not check_cluster_staging(cluster_info_obj=self._cluster_info, stage="stage2"):
            raise RuntimeError("Cluster is not successfully installed: Stage2 information missing! Operation aborted.")

        self._runtime_validation()

        self._csm.do_upgrade()
        self._persist_cluster_state_if_needed()

        if self._cfg.target_software_info.kube_installer_version == self._current_software_info.kube_installer_version \
            and self._cfg.target_software_info.kube_version == self._current_software_info.kube_version \
                and not self._cfg.force_upgrade:
            self._upgrade_kube_needed = False

        if self._cfg.target_software_info.image_namespace == self._current_software_info.image_namespace \
            and self._cfg.target_software_info.image_version == self._current_software_info.image_version \
            and self._cfg.target_software_info.image_version != "latest" \
            and not self._upgrade_kube_needed \
                and not self._cfg.force_upgrade:
            self._upgrade_service_needed = False

        if not self._upgrade_kube_needed and not self._upgrade_service_needed:
            logger.info("%sCluster's software versions is not changed, not performing upgrade.%s", COLOR_GREEN, COLOR_NORM)
            logger.info("%sIf you want to force upgrade cluster, please specify --force-upgrade flag.%s", COLOR_YELLOW, COLOR_NORM)
            sys.exit(0)

    def run(self):
        upgrade_info = "    Software Image: {}:{}  ->  {}:{}\n".format(
            self._current_software_info.image_namespace, self._current_software_info.image_version,
            self._cfg.target_software_info.image_namespace, self._cfg.target_software_info.image_version
        )
        upgrade_info += "    Kubernetes: {}  ->  {}\n".format(
            self._current_software_info.kube_version, self._cfg.target_software_info.kube_version
        )
        upgrade_info += "    Kubernetes Installer: {}  ->  {}".format(
            self._current_software_info.kube_installer_version, self._cfg.target_software_info.kube_installer_version
        )
        logger.info("\n\n%sUpgrading cluster %s:\n\n%s%s\n", COLOR_GREEN, self._name_id, upgrade_info, COLOR_NORM)

        if self._cfg.dry_run:
            logger.info("DRY RUN: upgrading cluster %s", self._name_id)
            return

        # Main upgrade cluster routine
        try:
            self._ensure_credentials()

            self._ensure_upgrader_access()

            ensure_manifest_temp_dir()

            if self._upgrade_service_needed:
                self._shutdown_platform()

            if self._upgrade_kube_needed:
                self._upgrade_kube()

            if self._upgrade_service_needed:
                self._start_platform()
                self._cluster_info.upload_platform_manifests_and_config(
                    platform_manifest_root=self._cfg.manifest_root,
                    platform_config=self._cfg.bootstrap_config
                )
            logger.info("\n\n%sSuccessfully upgraded cluster %s:\n\n%s%s\n", COLOR_GREEN, self._name_id, upgrade_info, COLOR_NORM)
        except Exception as e:
            logger.exception(e)
            raise RuntimeError(e)
        finally:
            self._disallow_upgrader_access_if_needed()

    def post_run(self):
        self._csm.done_upgrade()
        self._persist_cluster_state_if_needed()

    def _runtime_validation(self):
        all_errs = []
        # Abort operation if cluster is not successfully installed
        if not check_cluster_staging(cluster_info_obj=self._cluster_info, stage="stage2"):
            all_errs.append("Cannot upgrade cluster that is not successfully installed: Stage2 information missing!")

        cluster_status_raw = self._cluster_info.download_cluster_status_before_pause()
        if cluster_status_raw:
            all_errs.append("Upgrading a paused cluster is not currently supported. Please restart it first")

        # Abort operation if registry information changed
        if self._cfg.target_software_info.registry != self._current_software_info.registry \
            or self._cfg.target_software_info.registry_secrets != self._current_software_info.registry_secrets:
            all_errs.append("Changing registry information during upgrade is not supported currently!")

        # Abort operation if ami information changed
        if self._cfg.target_software_info.ami_name != self._current_software_info.ami_name \
            or (self._cfg.target_software_info.ami_id and self._cfg.target_software_info.ami_id != self._current_software_info.ami_id):
            all_errs.append("Upgrading AMI information is not currently supported.")

        if all_errs:
            raise RuntimeError("Upgrade aborted. Error(s): {}".format(all_errs))

    def _ensure_credentials(self):
        self._cluster_info.download_kube_config()
        self._cluster_info.download_kube_key()

    def _shutdown_platform(self):
        """
        This step shuts down platform based on the config and manifest provided
        :return:
        """
        logger.info("Shutting Argo platform ...")
        self._cluster_info.download_platform_manifests_and_config(
            target_platform_manifest_root=TEMP_PLATFORM_MANIFEST_ROOT,
            target_platform_config_path=TEMP_PLATFORM_CONFIG_PATH
        )
        platform = AXPlatform(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            manifest_root=TEMP_PLATFORM_MANIFEST_ROOT,
            config_file=TEMP_PLATFORM_CONFIG_PATH
        )
        platform.stop()
        platform.stop_monitor()

    def _upgrade_kube(self):
        """
        This function calls our script to upgrade Kubernetes and cluster nodes
        :return:
        """
        env = {
            "CLUSTER_NAME_ID": self._name_id,
            "AX_CUSTOMER_ID": AXCustomerId().get_customer_id(),
            "OLD_KUBE_VERSION": self._current_software_info.kube_version,
            "NEW_KUBE_VERSION": self._cfg.target_software_info.kube_version,
            "NEW_CLUSTER_INSTALL_VERSION": self._cfg.target_software_info.kube_installer_version,
            "ARGO_AWS_REGION": self._cluster_config.get_region(),
            "AX_TARGET_CLOUD": Cloud().target_cloud()
        }
        
        if self._cfg.cloud_profile:
            env["ARGO_AWS_PROFILE"] = self._cfg.cloud_profile

        logger.info("Upgrading Kubernetes with environments %s", pformat(env))
        env.update(os.environ)
        subprocess.check_call(["upgrade-kubernetes"], env=env)

    def _start_platform(self):
        """
        This step brings up Argo platform services
        :return:
        """
        logger.info("Bringing up Argo platform ...")

        platform = AXPlatform(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            manifest_root=self._cfg.manifest_root,
            config_file=self._cfg.bootstrap_config,
            software_info=self._cfg.target_software_info
        )
        platform.start()
        platform.stop_monitor()

    def _ensure_upgrader_access(self):
        if self._cidr not in self._cluster_config.get_trusted_cidr():
            logger.info("Upgrading cluster from a not trusted IP (%s). Temporarily allowing access.", self._cidr)
            self._bootstrap_obj.modify_node_security_groups(
                old_cidr=[],
                new_cidr=[self._cidr],
                action_name="allow-cluster-manager"
            )

    def _disallow_upgrader_access_if_needed(self):
        if self._cidr not in self._cluster_config.get_trusted_cidr():
            logger.info("Upgrading cluster from a not trusted IP (%s). Disallowing access.", self._cidr)
            self._bootstrap_obj.modify_node_security_groups(
                old_cidr=[self._cidr],
                new_cidr=[],
                action_name="disallow-cluster-manager"
            )
