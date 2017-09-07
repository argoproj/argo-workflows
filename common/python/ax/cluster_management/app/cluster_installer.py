# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


"""
Module for installing an Argo cluster

WARNING: Order of each steps is extremely important as there is a lot of Chicken-N-Egg problem
         involved. Think carefully through the steps and what can go wrong when trying to modify
         this class
"""

import json
import logging
import os
import subprocess
import sys
import yaml
from pprint import pformat
from retrying import retry

from ax.cloud import Cloud
from ax.cloud.aws import AMI, EC2IPPermission
from ax.platform.bootstrap import AXBootstrap
from ax.platform.cluster_buckets import AXClusterBuckets
from ax.platform.cluster_config import AXClusterConfig, SpotInstanceOption, AXClusterSize
from ax.platform.ax_cluster_info import AXClusterInfo
from ax.platform.ax_kube_up_down import AXKubeUpDown
from ax.platform.kube_env_config import prepare_kube_install_config
from ax.platform.platform import AXPlatform
from ax.platform.resource import KubeMasterResourceConfig
from ax.util.const import COLOR_GREEN, COLOR_NORM
from .common import ClusterOperationBase, check_cluster_staging
from .options import ClusterInstallConfig


logger = logging.getLogger(__name__)


# Default cluster config templates
# TODO: remove "dev" in file name after we no longer support old axinstaller
# TODO: make cluster config template configurable through CLI once we are more comfortable to leave that to user
CLUSTER_CONFIG_ROOT = "/ax/config/cloud"
CLUSTER_CONFIG_TEMPLATES = {
        "mvc": "cloud_template_mvc.json",
        "small": "cloud_template_small.json",
        "medium": "cloud_template_medium.json",
        "large": "cloud_template_large.json",
        "xlarge": "cloud_template_xlarge.json",
}

DEFAULT_NODE_SPOT_PRICE = "0.1512"
CLUSTER_META_DATA_PATH = "/tmp/cluster_meta/metadata.yaml"
ARGO_CONFIG = "/root/.argo/{fname}"
ARGO_CONFIG_DEFAULT = ARGO_CONFIG.format(fname='default')


class ClusterInstaller(ClusterOperationBase):
    def __init__(self, cfg):
        assert isinstance(cfg, ClusterInstallConfig)
        self._cfg = cfg
        super(ClusterInstaller, self).__init__(
            cluster_name=self._cfg.cluster_name,
            cluster_id=self._cfg.cluster_id,
            cloud_profile=self._cfg.cloud_profile,
            generate_name_id=True,
            dry_run=self._cfg.dry_run
        )

        self._name_id = self._idobj.get_cluster_name_id()

        # Ensure cluster buckets before instantiating any class that uses cluster buckets
        # Note that AXClusterId object is an exception as we need to create cluster name_id
        # first, instantiating buckets, and finally upload cluster name id
        # TODO (#116) bucket initialization should not depend on cluster name id
        AXClusterBuckets(
            name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            aws_region=self._cfg.cloud_region
        ).update()

        self._cluster_config = AXClusterConfig(cluster_name_id=self._name_id, aws_profile=self._cfg.cloud_profile)
        self._cluster_info = AXClusterInfo(cluster_name_id=self._name_id, aws_profile=self._cfg.cloud_profile)

    def pre_run(self):
        if self._csm.is_running():
            logger.info("Cluster is already installed and running. Please ask your administrator")
            sys.exit(0)
        self._csm.do_install()
        self._persist_cluster_state_if_needed()

    def post_run(self):
        self._csm.done_install()
        self._persist_cluster_state_if_needed()

    def run(self):
        """
        Main install routine
        :return:
        """
        self._pre_install()
        self._ensure_kubernetes_cluster()
        if self._cfg.dry_run:
            logger.info("DRY RUN: not installing cluster")
            return
        cluster_dns, username, password = self._ensure_argo_microservices()

        # Dump Argo cluster profile
        if username and password:
            logger.info("Generating Argo cluster profile ...")
            argo_config_path = ARGO_CONFIG.format(fname=self._idobj.get_cluster_name_id())
            with open(argo_config_path, "w") as f:
                f.write(
                    """
insecure: true
password: {password}
url: https://{dns}
username: {username}
""".format(password=password, dns=cluster_dns, username=username)
            )
            if not os.path.exists(ARGO_CONFIG_DEFAULT):
                # if user has not yet configured default argo config, symlink a default config to the one just created
                os.symlink(os.path.basename(argo_config_path), ARGO_CONFIG_DEFAULT)

        summary = """
              Cluster Name:  {cluster_name}
                Cluster ID:  {cluster_id}
      Cluster Profile Name:  {name_id}
               Cluster DNS:  {dns}
          Initial Username:  {username}
          Initial Password:  {password}

Note if your username and password are empty, your cluster has already been successfully installed before.

In this case, your argo CLI profile is NOT configured, as we only generate initial username and password once,
please contact your administrator for more information to configure your argo CLI profile.
        """.format(
            cluster_name=self._idobj.get_cluster_name(),
            cluster_id =self._idobj.get_cluster_id(),
            name_id=self._name_id,
            dns=cluster_dns,
            username=username,
            password=password
        )
        logger.info("Cluster information:\n%s%s%s\n", COLOR_GREEN, summary, COLOR_NORM)

    def _pre_install(self):
        """
        Pre install ensures the following stuff:
            - Cluster name/id mapping is created and uploaded
            - A local copy of cluster config is generated
            - Upload stage0 information to S3

        Stage0 is an indication of the fact that at least some part of the cluster could have been created.
        This step is idempotent.
        :return:
        """
        if check_cluster_staging(self._cluster_info, "stage0"):
            logger.info("Skip pre install")
            return

        logger.info("Cluster installation step: Pre Install")

        # After buckets are ensured, we persist cluster name id information
        # There is no problem of re-uploading we rerun this step
        self._idobj.upload_cluster_name_id()

        # Generate raw config dict
        raw_cluster_config_dict = self._generate_raw_cluster_config_dict()

        # Set cluster config object with raw cluster config dict
        self._cluster_config.set_config(raw_cluster_config_dict)

        # Prepare configuration for kube installer. This call will write
        prepare_kube_install_config(
            name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            cluster_info=self._cluster_info,
            cluster_config=self._cluster_config
        )

        # Save config file to s3, which is also stage0 information
        self._cluster_config.save_config()
        logger.info("Cluster installation step: Pre Install successfully finished")

    def _ensure_kubernetes_cluster(self):
        """
        This step won't run if there is "--dry-run" specified.

        This step assumes pre-install is already finished. This step does the following:
            - Config kube-installer
            - Call kube-installer to create Kubernetes cluster
            - Persist cluster credentials (Kubeconfig file and ssh key) to S3
            - Upload finalized cluster config and cluster metadata to S3
            - Upload stage1 information to S3

        Stage1 is an indication of the fact that there is a kubernetes cluster ready, and we can
        create micro-services on it.

        This step is NOT necessarily idempotent:
        e.g., if you created master but install failed due to cloud provider rate limit, and as a result,
        you have not yet created minions, for some reasons you quit your cluster manager container, all
        your cluster credentials can be lost.

        So if this step fails, the safest way is to uninstall the half installed cluster and start another install
        :return:
        """
        if check_cluster_staging(self._cluster_info, "stage1"):
            logger.info("Skip ensure Kubernetes cluster")
            return

        logger.info("Cluster installation step: Ensure Kubernetes Cluster")

        # Reload config in case stage0 is skipped
        self._cluster_config.reload_config()
        logger.info("Creating cluster with config: \n\n%s\n", pformat(self._cluster_config.get_raw_config()))

        # if dry-run is specified, this step should be skipped
        if self._cfg.dry_run:
            return

        # Call kube-up
        logger.info("\n\n%sCalling kube-up ...%s\n", COLOR_GREEN, COLOR_NORM)
        AXKubeUpDown(
            cluster_name_id=self._name_id,
            env=self._cluster_config.get_kube_installer_config(),
            aws_profile=self._cfg.cloud_profile
        ).up()

        # kube-up will generate cluster metadata. We add information from cluster metadata into cluster config
        logger.info("Loading cluster meta into cluster config ...")
        with open(CLUSTER_META_DATA_PATH, "r") as f:
            data = f.read()
        cluster_meta = yaml.load(data)
        self._cluster_config.load_cluster_meta(cluster_meta)

        # Persist updated cluster config
        self._cluster_config.save_config()

        # Upload cluster metadata
        self._cluster_info.upload_cluster_metadata()

        # Finally persist stage1
        self._cluster_info.upload_staging_info(stage="stage1", msg="stage1")

        logger.info("Cluster installation step: Ensure Kubernetes Cluster successfully finished")

    def _ensure_argo_microservices(self):
        """
        This step won't run if there is "--dry-run" specified.

        This step assumes there is a running Kubernetes cluster. This step does the following:
            - ensure ASG count
            - ensure trusted CIDRs
            - install Argo software on to the cluster and make sure they are up and running (We don't monitor
               if the microservice is having a crash loop)
            - Remove manager CIDR if it is not part of user-specified trusted CIDRs
            - Upload stage2 information to S3

        Stage2 is an indication that the cluster has been successfully installed: Kubernetes is up and running, and
        all Argo software are up and running. It does not ensure that non of Argo software should be in crash loop
        This step is idempotent
        :return: cluster_dns_name, username, password
        """
        logger.info("Cluster installation step: Ensure Argo Micro-services")

        # Reload config in case stage0 and stage1 are skipped
        self._cluster_config.reload_config()

        trusted_cidrs = self._cluster_config.get_trusted_cidr()

        # Instantiate AXBootstrap object. There are a bunch of stand-alone tasks we need to
        # perform using that object.
        axbootstrap = AXBootstrap(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            region=self._cluster_config.get_region()
        )

        # We allow access from everywhere during installation phase, but will remove this access
        # if user does not specify 0.0.0.0/0 as their trusted CIDR
        axbootstrap.modify_node_security_groups(
            old_cidr=[],
            new_cidr=trusted_cidrs + [EC2IPPermission.AllIP],
            action_name="allow-creator"
        )

        if check_cluster_staging(self._cluster_info, "stage2"):
            # TODO: some duplicated logic here, might need to combine them.
            logger.info("Skip ensure Argo micro-services since cluster has already been successfully installed")
            platform = AXPlatform(cluster_name_id=self._name_id, aws_profile=self._cfg.cloud_profile)
            if EC2IPPermission.AllIP not in trusted_cidrs:
                axbootstrap.modify_node_security_groups(
                    old_cidr=[EC2IPPermission.AllIP],
                    new_cidr=[],
                    action_name="disallow-creator"
                )
            return platform.get_cluster_external_dns(), "", ""

        # Modify ASG
        axsys_node_count = int(self._cluster_config.get_asxys_node_count())
        axuser_min_count = int(self._cluster_config.get_min_node_count()) - axsys_node_count
        axuser_max_count = int(self._cluster_config.get_max_node_count()) - axsys_node_count
        axbootstrap.modify_asg(
            min=axuser_min_count,
            max=axuser_max_count
        )

        # Install Argo micro-services
        # Platform install
        platform = AXPlatform(
            cluster_name_id=self._name_id,
            aws_profile=self._cfg.cloud_profile,
            manifest_root=self._cfg.manifest_root,
            config_file=self._cfg.bootstrap_config
        )

        install_platform_failed = False
        install_platform_failure_message = ""
        try:
            platform.start()
            platform.stop_monitor()
        except Exception as e:
            logger.exception(e)
            install_platform_failed = True
            install_platform_failure_message = str(e) + "\nPlease manually check the cluster status and retry installation with same command if the error is transient."

        if install_platform_failed:
            raise RuntimeError(install_platform_failure_message)

        # In case platform is successfully installed,
        # connect to axops to get initial username and password
        username, password = self._get_initial_cluster_credentials()

        # Remove access from 0.0.0.0/0 if this is not what user specifies
        if EC2IPPermission.AllIP not in trusted_cidrs:
            axbootstrap.modify_node_security_groups(
                old_cidr=[EC2IPPermission.AllIP],
                new_cidr=[],
                action_name="disallow-creator"
            )

        # Persist manifests to S3
        self._cluster_info.upload_platform_manifests_and_config(
            platform_manifest_root=self._cfg.manifest_root,
            platform_config=self._cfg.bootstrap_config
        )

        # Finally persist stage2 information
        self._cluster_info.upload_staging_info(stage="stage2", msg="stage2")
        logger.info("Cluster installation step: Ensure Argo Micro-services successfully finished")
        return platform.cluster_dns_name, username, password

    @retry(wait_fixed=5, stop_max_attempt_number=10)
    def _get_initial_cluster_credentials(self):
        """
        This functions connects to axops pod to get cluster's initial credentials
        :return: (username, password)
        """
        # TODO: a less hacky way of getting initial credentials?
        ns_conf = "--namespace axsys --kubeconfig {config}".format(config=self._cluster_info.get_kube_config_file_path())
        cmd = "kubectl " + ns_conf + " exec $(kubectl " + ns_conf + " get pods | grep axops | awk '{print $1}') /axops/bin/axpassword -c axops"
        ret = subprocess.check_output(cmd, shell=True)
        username = None
        password = None
        for line in ret.split("\n"):
            if line.startswith("Username"):
                # Username line has format "Username: xxxxxxx"
                username = line[len("Username: "):]
            if line.startswith("Password"):
                # Password line has format "Password: xxxxxx"
                password = line[len("Password: "):]
        assert username and password, "Failed to get username and password from axops pod: {}".format(ret)
        return username, password

    def _generate_raw_cluster_config_dict(self):
        """
        This is a standalone method to generate cluster config dictionary based on install config. We might want to
        move it to ax.platform.cluster_config package for sanity
        :return:
        """
        config_file_name = CLUSTER_CONFIG_TEMPLATES[self._cfg.cluster_size]
        config_file_full_path = os.path.join(*[CLUSTER_CONFIG_ROOT, self._cfg.cluster_type, config_file_name])
        with open(config_file_full_path, "r") as f:
            config = json.load(f)

        if Cloud().target_cloud_aws():
            return self._generate_raw_cluster_config_dict_aws(config)
        elif Cloud().target_cloud_gcp():
            return self._generate_raw_cluster_config_dict_gcp(config)
        else:
            # Should never come here as aws/gcp is ensured at CLI validation level
            return config

    def _generate_raw_cluster_config_dict_aws(self, config):
        """
        Generate AWS specific cluster config.
        :param config:
        :return:
        """
        # TODO: once we support installing with config file, we only overwrite when item is specifically set through CLI
        config["cloud"]["configure"]["region"] = self._cfg.cloud_region
        config["cloud"]["configure"]["placement"] = self._cfg.cloud_placement
        config["cloud"]["trusted_cidr"] = self._cfg.trusted_cidrs
        config["cloud"]["vpc_id"] = self._cfg.vpc_id

        # If we install into existing VPC, i.e. vpc_id is not None, or we are going to fetch it
        # from cluster metadata after cluster is created.
        config["cloud"]["vpc_cidr_base"] = self._cfg.vpc_cidr_base if not self._cfg.vpc_id else None
        config["cloud"]["subnet_size"] = self._cfg.subnet_mask_size
        config["cloud"]["configure"]["sandbox_enabled"] = self._cfg.enable_sandbox

        # TODO (#119): might want to remove this filed as this was used for hacks before. Setting it to "dev" for now
        config["cloud"]["configure"]["cluster_user"] = "dev"

        # TODO (#117): Switch all spot related options by literals rather than true/false and some other hacks
        # also need to revise the need of specifying a spot price during installation
        if self._cfg.spot_instances_option in [SpotInstanceOption.PARTIAL_SPOT, SpotInstanceOption.ALL_SPOT]:
            spot_instances_enabled = "true"
        else:
            spot_instances_enabled = "false"
        config["cloud"]["configure"]["spot_instances_enabled"] = spot_instances_enabled
        config["cloud"]["configure"]["spot_instances_option"] = self._cfg.spot_instances_option
        config["cloud"]["node_spot_price"] = DEFAULT_NODE_SPOT_PRICE

        # Configure master
        axsys_node_type = config["cloud"]["configure"]["axsys_node_type"]
        axsys_node_max = config["cloud"]["configure"]["axsys_node_count"]
        axuser_node_type = config["cloud"]["configure"]["axuser_node_type"]
        axuser_node_max = config["cloud"]["configure"]["max_node_count"] - axsys_node_max
        cluster_type = config["cloud"]["configure"]["cluster_type"]
        master_config = KubeMasterResourceConfig(
            usr_node_type=axuser_node_type,
            usr_node_max=axuser_node_max,
            ax_node_type=axsys_node_type,
            ax_node_max=axsys_node_max,
            cluster_type=cluster_type
        )
        if self._cfg.cluster_size == AXClusterSize.CLUSTER_MVC:
            # MVC cluster does not follow the heuristics we used to configure master
            config["cloud"]["configure"]["master_type"] = "m3.xlarge"
        else:
            config["cloud"]["configure"]["master_type"] = master_config.master_instance_type
        config["cloud"]["configure"]["master_config_env"] = master_config.kube_up_env

        # TODO (#121) Need to revise the relationship between user_on_demand_nodes and node minimum, system node count
        config["cloud"]["configure"]["axuser_on_demand_nodes"] = self._cfg.user_on_demand_nodes

        # Get AMI information
        ami_name = self._cfg.software_info.ami_name
        ami_id = AMI(
            aws_profile=self._cfg.cloud_profile,
            aws_region=self._cfg.cloud_region
        ).get_ami_id_from_name(ami_name=ami_name)
        config["cloud"]["configure"]["ami_id"] = ami_id

        # Other configurations
        config["cloud"]["configure"]["autoscaler_scan_interval"] = str(self._cfg.autoscaling_interval) + "s"
        config["cloud"]["configure"]["support_object_store_name"] = str(self._cfg.support_object_store_name)

        return config

    def _generate_raw_cluster_config_dict_gcp(self, config):
        """
        Generate GCP specific cluster config.
        :param config:
        :return:
        """
        config["cloud"]["trusted_cidr"] = self._cfg.trusted_cidrs
        return config
