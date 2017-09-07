# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


"""
Module for pausing an Argo cluster
"""


import logging
import json
import subprocess
import yaml

from ax.cloud import Cloud
from ax.cloud.aws import AWS_DEFAULT_PROFILE
from ax.cloud.aws.ebs import delete_tagged_ebs
from ax.cloud.aws.elb import ManagedElb
from ax.cloud.aws.server_cert import delete_server_certificate
from ax.platform.ax_cluster_info import AXClusterInfo
from ax.platform.ax_kube_up_down import AXKubeUpDown
from ax.platform.bootstrap import AXBootstrap
from ax.platform.cluster_config import AXClusterConfig
from ax.platform.cluster_buckets import AXClusterBuckets
from ax.platform.consts import COMMON_CLOUD_RESOURCE_TAG_KEY
from ax.util.const import COLOR_GREEN, COLOR_NORM, COLOR_YELLOW
from ax.util.network import get_public_ip

from .common import ClusterOperationBase, check_cluster_staging
from .options import ClusterUninstallConfig

logger = logging.getLogger(__name__)


class ClusterUninstaller(ClusterOperationBase):
    def __init__(self, cfg):
        assert isinstance(cfg, ClusterUninstallConfig)
        self._cfg = cfg
        super(ClusterUninstaller, self).__init__(
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

        # Initialize node count to 1 as master is not in an auto scaling group
        self._total_nodes = 1
        self._cidr = str(get_public_ip()) + "/32"

    def pre_run(self):
        # Abort operation if cluster is not successfully installed
        if not check_cluster_staging(cluster_info_obj=self._cluster_info, stage="stage2") and not self._cfg.force_uninstall:
            raise RuntimeError("Cluster is not successfully installed or has already been half deleted. If you really want to uninstall the cluster, please add '--force-uninstall' flag to finish uninstalling cluster. e.g. 'argocluster uninstall --force-uninstall --cluster-name xxx'")
        if not self._csm.is_running() and not self._cfg.force_uninstall:
            raise RuntimeError("Cluster is not in Running state. If you really want to uninstall the cluster, please add '--force-uninstall' flag to finish uninstalling cluster. e.g. 'argocluster uninstall --force-uninstall --cluster-name xxx'")
        self._csm.do_uninstall()
        self._ensure_critical_information()
        self._persist_cluster_state_if_needed()

    def post_run(self):
        return

    def run(self):
        if self._cfg.dry_run:
            logger.info("DRY RUN: Uninstalling cluster %s", self._name_id)
            return

        logger.info("%s\n\nUninstalling cluster %s%s\n", COLOR_GREEN, self._name_id, COLOR_NORM)

        # Main uninstall cluster routine
        try:
            self._check_cluster_before_uninstall()

            # We only need to keep stage0 information, which is an indication of we still need to
            # clean up the Kubernetes cluster
            self._cluster_info.delete_staging_info("stage2")
            self._cluster_info.delete_staging_info("stage1")
            self._clean_up_kubernetes_cluster()

            # As _clean_up_argo_specific_cloud_infrastructure() will clean everything inside bucket
            # that is related to this cluster, stage0 information is not explicitly deleted here
            self._clean_up_argo_specific_cloud_infrastructure()

            logger.info("\n\n%sSuccessfully uninstalled cluster %s%s\n", COLOR_GREEN, self._name_id, COLOR_NORM)
        except Exception as e:
            logger.exception(e)
            raise RuntimeError(e)

    def _ensure_critical_information(self):
        """
        If not force uninstall, we don't require user to provide a cloud regions / placement and therefore
        these 2 fields in self._cfg are None. We need to load them from cluster config
        :return:
        """
        load_from_cluster_config = True
        if self._cfg.force_uninstall:
            if self._cfg.cloud_region and self._cfg.cloud_placement:
                load_from_cluster_config = False
            elif not check_cluster_staging(cluster_info_obj=self._cluster_info, stage="stage0"):
                # Fail uninstall when cluster_config does not exist and region/placement
                # information are not provided
                raise RuntimeError(
                    """

        Cluster Stage 0 information is missing. Cluster is either not installed or it's management records in S3 are broken.
        If you believe there is still resource leftover, please provide cluster's region/placement information using
        "--cloud-placement" and "--cloud-region"
        
                    """
                )

        if load_from_cluster_config:
            self._cfg.cloud_region = self._cluster_config.get_region()
            self._cfg.cloud_placement = self._cluster_config.get_zone()

    def _clean_up_argo_specific_cloud_infrastructure(self):
        """
        This step cleans up components in cloud provider that are specifically needed by
        Argo cluster, including:
            - Buckets (everything under this cluster's directory)
            - Server certificates
        :return:
        """
        logger.info("Cluster uninstall step: Clean Up Argo-specific Infrastructure")
        AXClusterBuckets(self._name_id, self._cfg.cloud_profile, self._cfg.cloud_region).delete()

        # Delete server certificates: This code is deleting the default server certificates created
        # by public and private elb. Since server certs cannot be tagged, we need to delete them this way.
        certname = ManagedElb.get_elb_name(self._name_id, "ing-pub")
        delete_server_certificate(self._cfg.cloud_profile, certname)
        certname = ManagedElb.get_elb_name(self._name_id, "ing-pri")
        delete_server_certificate(self._cfg.cloud_profile, certname)

    def _clean_up_kubernetes_cluster(self):
        """
        This step cleans up Kubernetes if needed. It only touches components in cloud provider that
        Kubernetes needs, including:
            - Load Balancers
            - Instances
            - Auto scaling groups
            - launch configurations
            - Volumes
            - Security groups
            - Elastic IPs
            - VPCs (If this VPC is not shared)
        :return:
        """
        if not check_cluster_staging(cluster_info_obj=self._cluster_info, stage="stage0") and not self._cfg.force_uninstall:
            logger.info("Skip clean up Kubernetes cluster")
            return

        logger.info("Cluster uninstall step: Clean Up Kubernetes Cluster")

        if self._cfg.force_uninstall:
            msg = "{}\n\nIt is possible that cluster S3 bucket is accidentally deleted,\n".format(COLOR_YELLOW)
            msg += "or S3 bucket information has been altered unintentionally. In this\n"
            msg += "case, we still try to delete cluster since this is force uninstall.\n"
            msg += "NOTE: cluster deletion might NOT be successful and still requires\n"
            msg += "user to clean up left-over resources manually.{}\n".format(COLOR_NORM)
            logger.warning(msg)

        env = {
            "KUBERNETES_PROVIDER": self._cfg.cloud_provider,
            "KUBE_AWS_ZONE": self._cfg.cloud_placement,
            "KUBE_AWS_INSTANCE_PREFIX": self._name_id
        }

        if self._cfg.cloud_profile:
            env["AWS_DEFAULT_PROFILE"] = self._cfg.cloud_profile
        else:
            env["AWS_DEFAULT_PROFILE"] = AWS_DEFAULT_PROFILE

        logger.info("\n\n%sCalling kube-down ...%s\n", COLOR_GREEN, COLOR_NORM)
        AXKubeUpDown(cluster_name_id=self._name_id, env=env, aws_profile=self._cfg.cloud_profile).down()

        # TODO (#111): revise volume teardown in GCP
        if Cloud().target_cloud_aws():
            delete_tagged_ebs(
                aws_profile=self._cfg.cloud_profile,
                tag_key=COMMON_CLOUD_RESOURCE_TAG_KEY,
                tag_value=self._name_id,
                region=self._cfg.cloud_region
            )

    def _check_cluster_before_uninstall(self):
        """
        This step does sanity check before uninstalling the cluster.
        :return:
        """
        if not self._cfg.force_uninstall:
            logger.info("Cluster uninstall step: Sanity Checking")
            self._cluster_info.download_kube_config()
            self._ensure_uninstaller_access()
            self._check_cluster_fixture(kube_config_path=self._cluster_info.get_kube_config_file_path())
        else:
            msg = "{}\n\nForce uninstall: Skip checking cluster. Note that uninstall might fail if there is\n".format(COLOR_YELLOW)
            msg += "still managed fixture hooked up with cluster. In case cluster uninstall failed due to AWS\n"
            msg += "resource dependency, please manually clean up those resources and retry uninstall.\n{}".format(COLOR_NORM)
            logger.warning(msg)

    @staticmethod
    def _check_cluster_fixture(kube_config_path):
        """
        This step checks if the cluster has any fixture hooked up.
            - If there are fixtures hooked up, we abort uninstall, as we don't know how to tear down managed
               fixtures when we clean up cloud resources
            - If we don't know whether there is fixture or not, we print out a warning for now and continue
        :param kube_config_path: path to kube_config
        :return:
        """
        with open(kube_config_path, "r") as f:
            config_data = f.read()
        kube_config = yaml.load(config_data)
        username = None
        password = None

        # All kubeconfig we generate has only 1 cluster
        server = kube_config["clusters"][0]["cluster"]["server"]

        for user in kube_config.get("users", []):
            u = user["user"]
            if u.get("username", ""):
                username = u.get("username")
                password = u.get("password")
                break
        if not (username and password):
            logger.warning(
                "%sFailed to check managed fixture because Kubernetes credentials cannot be found to access cluster%s",
                COLOR_YELLOW, COLOR_NORM)
            return

        cmd = [
            "curl",
            "--insecure",
            "--silent",
            "-u",
            "{}:{}".format(username, password),
            "--max-time",
            "15",
            "{server}/api/v1/proxy/namespaces/axsys/services/fixturemanager/v1/fixture/instances?deleted=false".format(
                server=server
            )
        ]

        try:
            ret = subprocess.check_output(cmd)
        except subprocess.CalledProcessError as cpe:
            msg = "{}\n\nFailed to check cluster fixture state due to {}. Cluster might\n".format(COLOR_YELLOW, cpe)
            msg += "not be healthy. We will proceed to uninstall cluster with best effort. Note if there are\n"
            msg += "fixtures that are not cleaned up, uninstall can fail. You can manually\n"
            msg += "clean them up and uninstall again.\n{}".format(COLOR_NORM)
            logger.warning(msg)
            return

        if ret:
            try:
                fixture = json.loads(ret).get("data", [])
                if fixture:
                    logger.error("Remaining fixtures:\n%s", fixture)
                    raise RuntimeError("Please cleanup all fixtures before doing uninstall. Or use '--force-uninstall' option to skip this check")
                else:
                    logger.info("Cluster has no fixture hooked up, proceed to uninstall.")
            except ValueError as ve:
                # In case cluster is not healthy, command output will not be able to loaded
                # as json. Currently treat it same as "Cannot get fixture data" case
                logger.warning("Cannot parse fixture info: %s. Assume cluster has no fixture, proceed to uninstall. Fixture info: %s", ve, ret)
        else:
            logger.warning(
                "Cannot get fixture data. Assume that cluster has no fixture hooked up, proceed to uninstall.")

    def _ensure_uninstaller_access(self):
        if self._cidr not in self._cluster_config.get_trusted_cidr():
            logger.info("Pausing cluster from a not trusted IP (%s). Temporarily allowing access.", self._cidr)
            bootstrap = AXBootstrap(
                cluster_name_id=self._name_id,
                aws_profile=self._cfg.cloud_profile,
                region=self._cfg.cloud_region
            )
            bootstrap.modify_node_security_groups(
                old_cidr=[],
                new_cidr=[self._cidr],
                action_name="allow-cluster-manager"
            )
