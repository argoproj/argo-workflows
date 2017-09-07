#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

"""
Module for cluster info
"""

import json
import logging
import os

from retrying import retry
from future.utils import with_metaclass

from ax.cloud.aws import EC2, EC2InstanceState
from ax.cloud import Cloud
from ax.platform_client.env import AXEnv
from ax.meta import AXClusterConfigPath
from ax.platform.consts import COMMON_CLOUD_RESOURCE_TAG_KEY
from ax.platform.exceptions import AXPlatformException
from ax.util.singleton import Singleton
from .cluster_config import AXClusterConfig

logger = logging.getLogger(__name__)


class AXClusterInfo(with_metaclass(Singleton, object)):

    default_config_path = "/tmp/ax_kube/cluster_{}.conf"
    default_key_path = os.path.expanduser("~/.ssh/kube_id_{}")
    default_cluster_meta_path = "/tmp/cluster_meta/metadata.yaml"

    def __init__(self, cluster_name_id, kube_config=None, key_file=None, metadata=None, aws_profile=None):
        """
        Config file initialization

        :param cluster_name_id: Cluster name_id in format of name-uuid, lcj-cluster-515d9828-7515-11e6-9b3e-a0999b1b4e15
        :param kube_config: kubernetes saved config file.
        :param key_file: cluster ssh key path
        :param metadata: path to cluster metadata
        :param aws_profile: AWS profile to access S3.
        """
        assert AXEnv().is_in_pod() or cluster_name_id, "Must specify cluster name from outside cluster"
        self._aws_profile = aws_profile
        self._cluster_name_id = cluster_name_id

        self._config = AXClusterConfig(cluster_name_id=cluster_name_id, aws_profile=aws_profile)
        self._kube_config = kube_config if kube_config else self.default_config_path.format(cluster_name_id)
        self._key_file = key_file if key_file else self.default_key_path.format(cluster_name_id)
        self._metadata_file = metadata if metadata else self.default_cluster_meta_path

        config_path = AXClusterConfigPath(name_id=cluster_name_id)
        self._bucket_name = config_path.bucket()
        self._bucket = Cloud().get_bucket(self._bucket_name, aws_profile=aws_profile)
        self._s3_kube_config_key = config_path.kube_config()
        self._s3_cluster_ssh_key = config_path.kube_ssh()
        self._s3_cluster_state_before_pause = config_path.state_before_pause()
        self._s3_cluster_meta = config_path.cluster_metadata()
        self._s3_cluster_software_info = config_path.versions()
        self._s3_platform_manifest_dir = config_path.platform_manifest_dir()
        self._s3_platform_config = config_path.platform_config()
        self._s3_cluster_current_state = config_path.current_state()

        self._s3_master_config_prefix = config_path.master_config_dir()
        self._s3_master_attributes_path = config_path.master_attributes_path()
        self._s3_master_user_data_path = config_path.master_user_data_path()

        # For cluster staging info, stage1 and stage2 can be uploaded, downloaded, deleted with AXClusterInfo
        # stage0 will can only be downloaded with AXClusterInfo. It will be uploaded during cluster information
        # initialization (i.e. upload cluster id an cluster config), and deleted during cluster information
        # clean up (i.e. during axinstaller uninstall)
        self._staging_info = {
            "stage0": config_path.cluster_install_stage0_key(),
            "stage1": config_path.cluster_install_stage1_key(),
            "stage2": config_path.cluster_install_stage2_key()
        }

    def upload_kube_config(self):
        """
        Save content in kube config file to S3
        """
        logger.info("Saving kubeconfig to s3 ...")
        with open(self._kube_config, "r") as f:
            data = f.read()
        self._bucket.put_object(self._s3_kube_config_key, data)
        logger.info("Saved kubeconfig %s at %s/%s", self._kube_config, self._bucket_name, self._s3_kube_config_key)

    def upload_kube_key(self):
        """
        Save content in ssh key file to S3
        """
        logger.info("Saving cluster ssh key to s3 ...")
        with open(self._key_file, "r") as f:
            data = f.read()
        self._bucket.put_object(self._s3_cluster_ssh_key, data)
        logger.info("Saved ssh key %s at %s/%s", self._key_file, self._bucket_name, self._s3_cluster_ssh_key)

    def upload_staging_info(self, stage, msg):
        assert stage in ["stage1", "stage2"], "Only stage1, and stage2 information is available"
        logger.info("Uploading Argo install %s info to s3 ...", stage)
        if not self._bucket.put_object(key=self._staging_info[stage], data=msg):
            raise AXPlatformException("Failed to upload Argo install {} info for {}".format(stage, self._cluster_name_id))
        logger.info("Uploading Argo install %s info %s to s3 ... DONE", stage, msg)

    def upload_cluster_status_before_pause(self, status):
        """
        We upload cluster asg configures once for idempotency. i.e. when pause cluster failed but we have already
        scaled asg to 0, the next time we execute pause-cluster should use the status it uploaded before it even
        tried to scale cluster down
        """
        logger.info("Uploading Argo cluster status before pause ...")
        if self._bucket.get_object(key=self._s3_cluster_state_before_pause):
            logger.info("Status before pause already uploaded")
            return

        if not self._bucket.put_object(key=self._s3_cluster_state_before_pause, data=status):
            raise AXPlatformException("Failed to upload cluster status before pause")
        logger.info("Uploading Argo cluster status before pause ... DONE")

    def upload_cluster_metadata(self):
        logger.info("Uploading Argo cluster metadata ...")
        with open(self._metadata_file, "r") as f:
            data = f.read()
        # User pods should be able to curl it so we have to set ACL to public-read
        if not self._bucket.put_object(self._s3_cluster_meta, data, ACL="public-read"):
            raise AXPlatformException("Failed to upload cluster metadata for {}".format(self._cluster_name_id))
        logger.info("Uploading Argo cluster metadata ... DONE")

    def upload_platform_manifests_and_config(self, platform_manifest_root, platform_config):
        """
        Upload platform manifests from given directory and platform config from given file path
        to S3 cluster bucket
        :param platform_manifest_root:
        :param platform_config:
        :return:
        """
        assert os.path.isdir(platform_manifest_root), "platform_manifest_root must be a directory"
        assert os.path.isfile(platform_config), "platform_config must be a file"
        logger.info("Uploading platform manifests and config ...")

        # Upload all manifests
        for f in os.listdir(platform_manifest_root):
            full_path = os.path.join(platform_manifest_root, f)
            if os.path.isfile(full_path):
                s3_path = self._s3_platform_manifest_dir + f
                logger.info("Uploading platform manifest %s -> %s", full_path, s3_path)
                self._bucket.put_file(
                    local_file_name=full_path,
                    s3_key=s3_path
                )

        # Upload platform config
        logger.info("Uploading platform config %s", platform_config)
        self._bucket.put_file(
            local_file_name=platform_config,
            s3_key=self._s3_platform_config
        )

        logger.info("Uploading platform manifests and config ... Done")

    def download_platform_manifests_and_config(self, target_platform_manifest_root, target_platform_config_path):
        """
        Download previously persisted platform manifests from S3 to given directory, and download previously
        persisted platform config file to given path
        :param target_platform_manifest_root:
        :param target_platform_config_path:
        :return:
        """
        assert os.path.isdir(target_platform_manifest_root), "target_platform_manifest_root must be a directory"
        logger.info("Downloading platform manifests and config ...")

        for obj in self._bucket.list_objects_by_prefix(prefix=self._s3_platform_manifest_dir):
            s3_key = obj.key
            full_path = os.path.join(target_platform_manifest_root, s3_key.split("/")[-1])
            logger.info("Downloading platform manifest %s -> %s", s3_key, full_path)
            self._bucket.download_file(key=s3_key, file_name=full_path)

        logger.info("Downloading platform config %s", target_platform_config_path)
        self._bucket.download_file(
            key=self._s3_platform_config,
            file_name=target_platform_config_path
        )
        logger.info("Uploading platform manifests and config ... Done")

    def download_kube_config(self):
        """
        Get kube config from S3 and save it in file
        """
        logger.info("Downloading kubeconfig from s3 ...")
        data = self._bucket.get_object(self._s3_kube_config_key)
        assert data is not None, "No kube config at {}/{}".format(self._bucket_name, self._s3_kube_config_key)
        dir = os.path.dirname(self._kube_config)
        if not os.path.exists(dir):
            os.makedirs(dir)
        with open(self._kube_config, "w") as f:
            f.write(data)
        logger.info("Downloaded kubeconfig from %s/%s to %s", self._bucket_name, self._s3_kube_config_key, self._kube_config)
        return self._kube_config

    def download_kube_key(self):
        """
        Get kube ssh key from S3 and save it in file
        """
        if Cloud().target_cloud_gcp():
            return
        logger.info("Downloading cluster ssh key from s3 ...")
        data = self._bucket.get_object(self._s3_cluster_ssh_key)
        assert data is not None, "No kube ssh key at {}/{}".format(self._bucket_name, self._s3_cluster_ssh_key)
        dir = os.path.dirname(self._key_file)
        if not os.path.exists(dir):
            os.makedirs(dir)
        with open(self._key_file, "w") as f:
            f.write(data)
        os.chmod(self._key_file, 0o0600)
        logger.info("Downloaded kube ssh key from %s/%s to %s", self._bucket_name, self._s3_cluster_ssh_key, self._key_file)
        return self._key_file

    def download_staging_info(self, stage):
        assert stage in ["stage0", "stage1", "stage2"], "Only stage0, stage1, and stage2 information is available"
        logger.info("Downloading Argo install %s info from s3 ...", stage)
        data = self._bucket.get_object(key=self._staging_info[stage])
        assert data is not None, "No Argo install {} info get at {}/{}".format(stage, self._bucket_name, self._staging_info[stage])
        return data

    def download_cluster_status_before_pause(self):
        logger.info("Downloading cluster status before pause ...")
        return self._bucket.get_object(key=self._s3_cluster_state_before_pause)

    def download_cluster_metadata(self):
        logger.info("Downloading cluster metadata")
        return self._bucket.get_object(key=self._s3_cluster_meta)

    def download_cluster_software_info(self):
        logger.info("Downloading cluster software info")
        data = self._bucket.get_object(key=self._s3_cluster_software_info)
        assert data, "No software info at {}/{}".format(self._bucket_name, self._s3_cluster_software_info)
        return data

    def delete_cluster_status_before_pause(self):
        logger.info("Deleting Argo cluster status before last pause ...")
        if not self._bucket.delete_object(key=self._s3_cluster_state_before_pause):
            raise AXPlatformException("Failed to delete {} information".format(self._s3_cluster_state_before_pause))
        logger.info("Deleted Argo cluster status before last pause")

    def delete_staging_info(self, stage):
        assert stage in ["stage1", "stage2"], "Only stage1, and stage2 information is available"
        logger.info("Deleting Argo install %s info from s3 ...", stage)
        if not self._bucket.delete_object(key=self._staging_info[stage]):
            raise AXPlatformException("Failed to delete {} information".format(stage))
        logger.info("Deleted Argo install %s info from s3 ...", stage)

    def download_cluster_current_state(self):
        logger.info("Downloading cluster current state ...")
        return self._bucket.get_object(key=self._s3_cluster_current_state)

    def upload_cluster_current_state(self, state):
        logger.info("Uploading cluster current state ...")
        if not self._bucket.put_object(key=self._s3_cluster_current_state, data=state):
            raise AXPlatformException("Failed to upload cluster current state info for {}".format(self._cluster_name_id))
        logger.info("Uploading cluster current state ... DONE")

    def get_kube_config_file_path(self):
        """
        Get local config file path after saving.
        """
        return self._kube_config

    def get_key_file_path(self):
        return self._key_file

    def get_bucket_name(self):
        return self._bucket_name

    @retry(wait_exponential_multiplier=5000, stop_max_attempt_number=2)
    def get_master_config(self, user_data_file):
        """
        Checks whether the config for the master instance is present in S3. This is done
        by checking if the directory specific to the given cluster name is present or not.

        :return Master config json if the config was in S3. None otherwise.
        """
        # Check if the master_config was previously stored in S3. If so, download it.
        object_list = list(self._bucket.list_objects_by_prefix(prefix=self._s3_master_config_prefix))
        if len(object_list) > 0:
            # Objects should already be in s3. No need to store.
            config_exists_in_s3 = True
            logger.info("Master config already exists in S3. Downloading ...")
            self._bucket.download_file(self._s3_master_user_data_path, user_data_file)
            return self._bucket.get_object(self._s3_master_attributes_path)

        logger.info("Master config not found in s3")
        return None

    @retry(wait_exponential_multiplier=5000, stop_max_attempt_number=3)
    def upload_master_config_to_s3(self, master_attributes, master_user_data):
        """
        Uploads the master attributes and user-data into a directory in the s3 bucket.
        """
        # Upload the attributes file.
        self._bucket.put_object(key=self._s3_master_attributes_path, data=json.dumps(master_attributes))
        # Upload the user-data file.
        self._bucket.put_object(key=self._s3_master_user_data_path, data=master_user_data)

    def generate_cluster_metadata_from_provider(self):
        ec2 = EC2(profile=self._aws_profile, region=self._config.get_region())
        minion_name = "{}-minion".format(self._cluster_name_id)

        # Assume minion has same network configurations
        minion = ec2.get_instances(name=minion_name, states=[EC2InstanceState.Running])[0]
        vpc_id = minion["NetworkInterfaces"][0]["VpcId"]
        subnet_id = minion["NetworkInterfaces"][0]["SubnetId"]
        zone = minion["Placement"]["AvailabilityZone"]
        sg_id = None
        for sg in minion["SecurityGroups"]:
            if sg["GroupName"] == "kubernetes-minion-{}".format(self._cluster_name_id):
                sg_id = sg["GroupId"]
        assert sg_id, "Unable to find security group for cluster minions"

        rtbs = ec2.get_routetables(
            tags={
                COMMON_CLOUD_RESOURCE_TAG_KEY: [self._cluster_name_id]
            }
        )
        assert len(rtbs) == 1, "Cluster has 0 or more than 1 routetables: {}".format(rtbs)
        rtb_id = rtbs[0]["RouteTableId"]

        subnets = ec2.get_subnets(
            zones=[zone],
            tags={
                COMMON_CLOUD_RESOURCE_TAG_KEY: [self._cluster_name_id]
            }
        )
        # Assume cluster has 1 subnet in 1 zone now, and 1 master node runs inside the same subnet
        assert len(subnets) == 1, "Cluster has 0 or more than 1 subnets in zone {}: {}".format(zone, subnets)
        subnet_cidr = subnets[0]["CidrBlock"]
        max_instance_count = int(self._config.get_max_node_count()) + 1

        igws = ec2.get_vpc_igws(vpc_id=vpc_id)
        assert len(igws) == 1, "VPC should have only 1 internet gateways. {}".format(igws)
        igw_id = igws[0]["InternetGatewayId"]

        return {
            "cluster_name": self._cluster_name_id,
            "vpc": vpc_id,
            "internet_gateway": igw_id,
            "route_table": rtb_id,
            "security_group": sg_id,
            "subnets": {
                zone: {
                    "subnet_id": subnet_id,
                    "subnet_cidr": subnet_cidr,
                    "max_instance_count": max_instance_count
                }
            }
        }
