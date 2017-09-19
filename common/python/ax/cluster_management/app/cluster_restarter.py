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
from ax.platform.component_config import SoftwareInfo
from ax.platform.platform import AXPlatform
from ax.util.const import COLOR_GREEN, COLOR_NORM
from ax.util.network import get_public_ip

from .common import ClusterOperationBase, check_cluster_staging, ensure_manifest_temp_dir,\
    TEMP_PLATFORM_CONFIG_PATH, TEMP_PLATFORM_MANIFEST_ROOT, is_portal_env
from .options import ClusterRestartConfig

logger = logging.getLogger(__name__)


WAIT_FOR_RUNNING_MASTER_RETRY = 30
WAIT_FOR_MINION_REG_RETRY = 120


class ClusterResumer(ClusterOperationBase):
    def __init__(self, cfg):
        assert isinstance(cfg, ClusterRestartConfig)
        self._cfg = cfg
        super(ClusterResumer, self).__init__(
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

        # Initialize node count to 1 as master is not in an auto scaling group
        self._total_nodes = 1
        self._cidr = str(get_public_ip()) + "/32"
        self._software_info = SoftwareInfo(
            info_dict=yaml.load(
                self._cluster_info.download_cluster_software_info()
            )
        )

    def pre_run(self):
        if not is_portal_env() and self._cluster_info.is_cluster_supported_by_portal():
            raise RuntimeError("Cluster is currently supported by portal. Please login to portal to perform cluster management operations.")

        if self._csm.is_running():
            logger.info("Cluster is already running.")
            sys.exit(0)
        if not check_cluster_staging(cluster_info_obj=self._cluster_info, stage="stage2"):
            raise RuntimeError("Cluster is not successfully installed: Stage2 information missing! Operation aborted.")
        self._csm.do_resume()
        self._persist_cluster_state_if_needed()

    def post_run(self):
        self._csm.done_resume()
        self._persist_cluster_state_if_needed()

    def run(self):
        if self._cfg.dry_run:
            logger.info("DRY RUN: Resuming cluster %s with software info %s",
                        self._name_id, self._software_info.to_dict())
            return

        logger.info("%s\n\nResuming cluster %s%s\n", COLOR_GREEN, self._name_id, COLOR_NORM)
        # Main resume cluster routine
        try:
            self._master_manager.restart_master()
            self._recover_auto_scaling_groups()
            self._wait_for_master()
            self._ensure_restarter_access()
            self._wait_for_minions()
            ensure_manifest_temp_dir()
            self._start_platform()
            logger.info("\n\n%sSuccessfully resumed cluster %s%s\n", COLOR_GREEN, self._name_id, COLOR_NORM)
        except Exception as e:
            logger.exception(e)
            raise RuntimeError(e)
        finally:
            self._disallow_restarter_access_if_needed()

    def _start_platform(self):
        """
        This step brings up Argo platform services
        :return:
        """
        logger.info("Bringing up Argo platform ...")

        self._cluster_info.download_platform_manifests_and_config(
            target_platform_manifest_root=TEMP_PLATFORM_MANIFEST_ROOT,
            target_platform_config_path=TEMP_PLATFORM_CONFIG_PATH
        )

        platform = AXPlatform(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            manifest_root=TEMP_PLATFORM_MANIFEST_ROOT,
            config_file=TEMP_PLATFORM_CONFIG_PATH,
            software_info=self._software_info
        )
        platform.start()
        platform.stop_monitor()

    def _wait_for_master(self):
        """
        This step waits for master to be up and running
        :return:
        """
        count = 0
        running_master = None
        while count < WAIT_FOR_RUNNING_MASTER_RETRY:
            logger.info("Waiting for master to be up and running. Trail %s / %s", count, WAIT_FOR_RUNNING_MASTER_RETRY)
            running_master = self._master_manager.discover_master(state=[EC2InstanceState.Running])
            if not running_master:
                time.sleep(5)
            else:
                logger.info("%sMaster %s is running%s", COLOR_GREEN, running_master, COLOR_NORM)
                break
            count += 1
        if count == WAIT_FOR_RUNNING_MASTER_RETRY:
            raise RuntimeError(
                "Timeout waiting for master {} to come up. Please manually check cluster status".format(running_master)
            )

    def _wait_for_minions(self):
        """
        This step waits for all minions to come up and registered in Kubernetes master
        :return:
        """
        # Get kubernetes access token
        self._cluster_info.download_kube_config()
        kube_config = self._cluster_info.get_kube_config_file_path()

        # Wait for nodes to be ready.
        # Because we made sure during pause that kubernetes master already knows that all minions are gone,
        # we don't need to worry about cached minions here
        logger.info("Wait 120 seconds before Kubernetes master comes up ...")
        time.sleep(120)
        kubectl = KubernetesApiClient(config_file=kube_config)
        logger.info("Waiting for all Kubelets to be ready ...")

        trail = 0
        while True:
            try:
                all_kubelets_ready = True
                nodes = kubectl.api.list_node()
                logger.info("%s / %s nodes registered", len(nodes.items), self._total_nodes)
                if len(nodes.items) < self._total_nodes:
                    all_kubelets_ready = False
                else:
                    for n in nodes.items:
                        kubelet_check = {
                            "KubeletHasSufficientDisk",
                            "KubeletHasSufficientMemory",
                            "KubeletHasNoDiskPressure",
                            "KubeletReady",
                            "RouteCreated"
                        }
                        for cond in n.status.conditions:
                            if cond.reason in kubelet_check:
                                kubelet_check.remove(cond.reason)
                        if kubelet_check:
                            logger.info("Node %s not ready yet. Remaining Kubelet checkmarks: %s", n.metadata.name,
                                        kubelet_check)
                            all_kubelets_ready = False
                            break
                        else:
                            logger.info("Node %s is ready.", n.metadata.name)
                if all_kubelets_ready:
                    logger.info("All Kubelets are ready")
                    break
            except Exception as e:
                if "Max retries exceeded" in str(e):
                    # If master API server is still not ready at this moment, we don't count as a trail
                    trail -= 1
                    logger.info("Kubernetes API server not ready yet")
                else:
                    logger.exception("Caught exception when listing nodes: %s", e)
            trail += 1
            if trail > WAIT_FOR_MINION_REG_RETRY:
                raise RuntimeError("Timeout waiting for minions to come up. Please manually check cluster status")
            time.sleep(10)

    def _recover_auto_scaling_groups(self):
        """
        This steps does the following:
            - fetch the previously restored auto scaling group config. If this config cannot be found,
              we can assume that all autoscaling groups have correct configurations. This could happen
              when previous restart failed in the middle but passed this stage already, or the cluster is
              not even paused
            - Wait for all instances to be in service
        :return:
        """
        # Get previously persisted asg status
        logger.info("Fetching last cluster status ...")
        cluster_status_raw = self._cluster_info.download_cluster_status_before_pause()

        asg_mgr = AXUserASGManager(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            region=self._cluster_config.get_region()
        )

        if cluster_status_raw:
            logger.info("Found last cluster status, restoring cluster ...")
            cluster_status = yaml.load(cluster_status_raw)
            all_asg_statuses = cluster_status["asg_status"]

            # Restore minions
            for asg_name in all_asg_statuses.keys():
                asg_status = all_asg_statuses[asg_name]
                min_size = asg_status["min_size"]
                max_size = asg_status["max_size"]
                desired = asg_status["desired_capacity"]
                self._total_nodes += desired
                logger.info("Recovering autoscaling group %s. Min: %s, Max: %s, Desired: %s", asg_name, min_size,
                            max_size, desired)
                asg_mgr.set_asg_spec(name=asg_name, minsize=min_size, maxsize=max_size, desired=desired)

            logger.info("Waiting for all auto scaling groups to scale up ...")
            asg_mgr.wait_for_desired_asg_state()
            logger.info("%sAll cluster instances are in service%s", COLOR_GREEN, COLOR_NORM)

            # Delete previously stored cluster status
            self._cluster_info.delete_cluster_status_before_pause()
        else:
            all_asgs = asg_mgr.get_all_asgs()
            for asg in all_asgs:
                self._total_nodes += asg["DesiredCapacity"]

            logger.info("Cannot find last cluster status, cluster already resumed with %s nodes", self._total_nodes)

    def _ensure_restarter_access(self):
        if self._cidr not in self._cluster_config.get_trusted_cidr():
            logger.info("Restarting cluster from a not trusted IP (%s). Temporarily allowing access.", self._cidr)
            self._bootstrap_obj.modify_node_security_groups(
                old_cidr=[],
                new_cidr=[self._cidr],
                action_name="allow-cluster-manager"
            )

    def _disallow_restarter_access_if_needed(self):
        if self._cidr not in self._cluster_config.get_trusted_cidr():
            logger.info("Restarting cluster from a not trusted IP (%s). Disallowing access.", self._cidr)
            self._bootstrap_obj.modify_node_security_groups(
                old_cidr=[self._cidr],
                new_cidr=[],
                action_name="disallow-cluster-manager"
            )
