# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


"""
Module for pausing an Argo cluster
"""


import logging
import yaml
import sys
import time

from ax.cloud.aws import EC2InstanceState
from ax.kubernetes.client import KubernetesApiClient
from ax.platform.ax_cluster_info import AXClusterInfo
from ax.platform.ax_asg import AXUserASGManager
from ax.platform.ax_master_manager import AXMasterManager
from ax.platform.bootstrap import AXBootstrap
from ax.platform.cluster_config import AXClusterConfig
from ax.platform.platform import AXPlatform
from ax.util.const import COLOR_GREEN, COLOR_NORM
from ax.util.network import get_public_ip

from .common import ClusterOperationBase, check_cluster_staging, ensure_manifest_temp_dir,\
    TEMP_PLATFORM_MANIFEST_ROOT, TEMP_PLATFORM_CONFIG_PATH
from .options import ClusterPauseConfig

logger = logging.getLogger(__name__)


class ClusterPauser(ClusterOperationBase):
    def __init__(self, cfg):
        assert isinstance(cfg, ClusterPauseConfig)
        self._cfg = cfg
        super(ClusterPauser, self).__init__(
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
        self._master_manager = AXMasterManager(
            cluster_name_id=self._name_id,
            region=self._cluster_config.get_region(),
            profile=self._cfg.cloud_profile
        )
        self._bootstrap_obj = AXBootstrap(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            region=self._cluster_config.get_region()
        )
        self._cidr = str(get_public_ip()) + "/32"

    def pre_run(self):
        if self._cluster_info.is_cluster_supported_by_portal():
            raise RuntimeError("Cluster is currently supported by portal. Please login to portal to perform cluster management operations.")
        if self._csm.is_paused():
            logger.info("Cluster is already paused.")
            sys.exit(0)

        # This is for backward compatibility
        if not check_cluster_staging(cluster_info_obj=self._cluster_info, stage="stage2"):
            raise RuntimeError("Cluster is not successfully installed: Stage2 information missing! Operation aborted.")
        self._csm.do_pause()
        self._persist_cluster_state_if_needed()

    def run(self):
        if self._cfg.dry_run:
            logger.info("DRY RUN: pausing cluster %s", self._name_id)
            return

        # Check if cluster's master is paused already. Since terminating master is the very last thing
        # of pausing cluster, if master is already stopped, cluster has already been successfully paused
        stopped_master = self._master_manager.discover_master(state=[EC2InstanceState.Stopped])
        if stopped_master:
            logger.info("\n\n%sMaster %s already stopped. Cluster %s already paused%s\n", COLOR_GREEN, stopped_master,
                        self._name_id, COLOR_NORM)
            return
        else:
            logger.info("\n\n%sPausing cluster %s%s\n", COLOR_GREEN, self._name_id, COLOR_NORM)

        # Main pause cluster routine
        try:
            self._ensure_pauser_access()
            ensure_manifest_temp_dir()
            self._shutdown_platform()
            self._scale_down_auto_scaling_groups()
            self._wait_for_deregistering_minions()
            logger.info("Stopping master ...")
            self._master_manager.stop_master()
            logger.info("\n\n%sSuccessfully paused cluster %s%s\n", COLOR_GREEN, self._name_id, COLOR_NORM)
        except Exception as e:
            logger.exception(e)
            raise RuntimeError(e)
        finally:
            self._disallow_pauser_access_if_needed()

    def post_run(self):
        self._csm.done_pause()
        self._persist_cluster_state_if_needed()

    def _wait_for_deregistering_minions(self):
        """
        This step waits for all minions to be de-registered from Kubernetes master,
        e.g. `kubectl get nodes` returns no minions besides master
        :return:
        """
        # Wait for kubernetes master de-register all minions
        logger.info("Waiting for Kubernetes master to de-register all existing minions")
        self._cluster_info.download_kube_config()
        kube_config = self._cluster_info.get_kube_config_file_path()
        kubectl = KubernetesApiClient(config_file=kube_config)
        while True:
            try:
                nodes = kubectl.api.list_node()
                node_names = []

                # list nodes should only show master now
                if len(nodes.items) > 1:
                    for n in nodes.items:
                        node_names.append(n.metadata.name)
                    logger.info("Remaining Kubernetes minions: %s", node_names)
                else:
                    # I don't see it necessary to check if the remaining node is master or not
                    logger.info("%sAll minions de-registered from master%s", COLOR_GREEN, COLOR_NORM)
                    break
            except Exception as e:
                logger.warning("Caught exception when listing nodes: %s", e)
            time.sleep(15)

    def _scale_down_auto_scaling_groups(self):
        """
        This step:
            - Persist autoscaling group states to S3,
            - Scale down all autoscaling groups to zero,
            - Wait for all minion to be terminated
        :return:
        """
        logger.info("Discovering autoscaling groups")
        asg_mgr = AXUserASGManager(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            region=self._cluster_config.get_region()
        )
        all_asgs = asg_mgr.get_all_asgs()

        # Generate cluster status before pause. This is used to recover same amount of nodes
        # when we want to restart cluster
        cluster_status = {
            "asg_status": {}
        }
        for asg in all_asgs:
            cluster_status["asg_status"][asg["AutoScalingGroupName"]] = {
                "min_size": asg["MinSize"],
                "max_size": asg["MaxSize"],
                "desired_capacity": asg["DesiredCapacity"]
            }
        self._cluster_info.upload_cluster_status_before_pause(
            status=yaml.dump(cluster_status)
        )

        # Scale down asg
        logger.info("Scaling down autoscaling groups ...")
        for asg in all_asgs:
            asg_name = asg["AutoScalingGroupName"]
            asg_mgr.set_asg_spec(name=asg_name, minsize=0, maxsize=0)

        # Waiting for nodes to be terminated
        logger.info("Waiting for all auto scaling groups to scale down ...")
        asg_mgr.wait_for_desired_asg_state()
        logger.info("%sAll cluster nodes are terminated%s", COLOR_GREEN, COLOR_NORM)

    def _shutdown_platform(self):
        """
        This step shuts down platform based on the config and manifest provided
        :return:
        """
        logger.info("Shutting platform for pausing the cluster ...")
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

    def _ensure_pauser_access(self):
        if self._cidr not in self._cluster_config.get_trusted_cidr():
            logger.info("Pausing cluster from a not trusted IP (%s). Temporarily allowing access.", self._cidr)
            self._bootstrap_obj.modify_node_security_groups(
                old_cidr=[],
                new_cidr=[self._cidr],
                action_name="allow-cluster-manager"
            )

    def _disallow_pauser_access_if_needed(self):
        if self._cidr not in self._cluster_config.get_trusted_cidr():
            logger.info("Pausing cluster from a not trusted IP (%s). Disallowing access.", self._cidr)
            self._bootstrap_obj.modify_node_security_groups(
                old_cidr=[self._cidr],
                new_cidr=[],
                action_name="disallow-cluster-manager"
            )
