#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import botocore
import sys
import base64
import logging
import os
import json
import time
import urllib3
import zlib
import boto3
from botocore.client import ClientError
from retrying import retry

from ax.aws.meta_data import AWSMetaData
from ax.cloud.aws import AMI, EC2InstanceState
from ax.cloud.aws import default_aws_retry
from ax.devops.kafka.kafka_client import EventNotificationClient
from ax.meta import AXCustomerId, AXClusterConfigPath
from ax.platform.exceptions import AXPlatformException
from ax.platform.cluster_config import AXClusterConfig
from ax.notification_center import FACILITY_PLATFORM, CODE_PLATFORM_ERROR, CODE_PLATFORM_CRITICAL
from ax.platform.ax_cluster_info import AXClusterInfo
from ax.platform.kube_env_config import default_kube_up_env
from ax.platform.cluster_instance_profile import AXClusterInstanceProfile
from ax.kubernetes.client import KubernetesApiClient, retry_unless
from ax.util import const

from distutils.log import INFO

logger = logging.getLogger("ax.master_manager")
logging.getLogger('boto3').setLevel(logging.WARNING)
logging.getLogger('botocore').setLevel(logging.WARNING)
logging.getLogger('s3transfer').setLevel(logging.WARNING)
logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S", stream=sys.stdout, level=INFO)

# S3 file stores current content from S3 buckets.
# New file is used for new user data based on upgrade request.
USER_DATA_FILE_S3 = "/tmp/user_data_s3"
USER_DATA_FILE_NEW = "/tmp/user_data_new"

WAIT_TIME_POST_RESTART_MIN = 10


def print_exception(e):
    logger.exception("Got the following exception {}".format(e))
    # this func returns true so that retry is triggered.
    return True


class AXMasterManager:
    def __init__(self, cluster_name_id, region=None, profile=None):
        self.cluster_name_id = cluster_name_id

        # Region and profile info can be passed in with upgrade code path,
        # when this is run from axclustermanager outside cluster.
        self.region = AWSMetaData().get_region() if region is None else region
        self.profile = profile
        if profile is None:
            session = boto3.Session(region_name=self.region)
        else:
            session = boto3.Session(region_name=self.region, profile_name=profile)

        self.ec2 = session.resource('ec2')
        self.client = session.client('ec2')
        self.cluster_info = AXClusterInfo(cluster_name_id=cluster_name_id, aws_profile=profile)
        self.cluster_config = AXClusterConfig(cluster_name_id=cluster_name_id, aws_profile=profile)
        cluster_config_path = AXClusterConfigPath(cluster_name_id)
        self.s3_bucket = cluster_config_path.bucket()
        self.s3_config_prefix = cluster_config_path.master_config_dir()
        self.s3_attributes_path = cluster_config_path.master_attributes_path()
        self.s3_user_data_path = cluster_config_path.master_user_data_path()

        logger.info("Create MasterManager in region %s, attributes path: %s and user_data_path: %s", self.region,
                    self.s3_attributes_path, self.s3_user_data_path)

        # The EC2 instance object for the current master
        self.master_instance = None

        # Properties/attributes to use when launching the new master
        self.attributes = {}

        # For upgrades.
        # The following values are set to None from master manager but to not None from upgrade code.
        self.aws_image = None
        self.instance_profile = None

        self.event_notification_client = EventNotificationClient(FACILITY_PLATFORM)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def discover_master(self, state=None):
        """
        Discover's the currently running master for the given cluster name.
        """
        if not state:
            state = [EC2InstanceState.Running]
        response = self.client.describe_instances(
            Filters=[
                {'Name': 'tag:Name', 'Values': [self.cluster_name_id + '-master']},
                {'Name': 'instance-state-name', 'Values': state}
            ]
        )
        # Confirm that there is only 1 master
        if len(response['Reservations']) == 0:
            logger.info("Master with state %s not found", state)
            return None

        assert len(response['Reservations']) == 1, "More than 1 master running (reservations != 1)!"
        assert len(response['Reservations'][0]['Instances']) == 1, "Not exactly 1 master instance is running! {}".format(response['Reservations'][0]['Instances'])
        return response['Reservations'][0]['Instances'][0]['InstanceId']

    def user_data_fixup(self, user_data):
        """
        The original user-data used for creating the master can become obsolete after upgrades. There are 5
        fields from the original user-data that need to be "fixed". The SERVER_BINARY_TAR_URL, SALT_TAR_URL,
        SERVER_BINARY_TAR_HASH, SALT_TAR_HASH and the wget command that downloads the bootstrap-script.
        """
        from ax.platform.kube_env_config import kube_env_update
        # TODO: It's not ideal to use env variables for passing arguments.
        # Env variables could be different between running as server and from upgrade.
        kube_version = os.getenv('KUBE_VERSION', os.getenv('NEW_KUBE_VERSION')).strip()
        cluster_install_version = os.getenv('AX_CLUSTER_INSTALL_VERSION', os.getenv('NEW_CLUSTER_INSTALL_VERSION')).strip()
        server_binary_tar_hash = os.getenv('SERVER_BINARY_TAR_HASH', os.getenv('NEW_KUBE_SERVER_SHA1')).strip()
        salt_tar_hash = os.getenv('SALT_TAR_HASH', os.getenv('NEW_KUBE_SALT_SHA1')).strip()
        updates = {
            "new_kube_version": kube_version,
            "new_cluster_install_version": cluster_install_version,
            "new_kube_server_hash": server_binary_tar_hash,
            "new_kube_salt_hash": salt_tar_hash,
            "new_api_servers": self.attributes['private_ip_address'],
        }
        dec = zlib.decompressobj(32 + zlib.MAX_WBITS)  # offset 32 to skip the header
        unzipped_user_data = dec.decompress(base64.b64decode(user_data))

        # Zip output buffer. For details: http://bit.ly/2gv3WKt
        comp = zlib.compressobj(9, zlib.DEFLATED, zlib.MAX_WBITS | 16)
        zipped_data = comp.compress(kube_env_update(unzipped_user_data, updates)) + comp.flush()

        # Convert output to base64 encoded
        logger.info("User data fixup completed")
        return base64.b64encode(zipped_data)

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def get_user_data(self):
        """
        Get's the user-data for the current master. Note that the user-data is base64 encoded when it is
        downloaded. Writes the data into a file.
        """
        # The user-data is base64 encoded.
        user_data = self.client.describe_instance_attribute(
            Attribute='userData', InstanceId=self.master_instance.instance_id)['UserData']['Value']
        # Download user-data and store it into a temporary file. This data is base64 encoded.
        # It is better to use a well-known location for this file rather than one generated by mkstemp (or variants).
        # That way, this file could be populated the first time this pod run or even later by simply downloading
        # the user-data from S3.
        try:
            user_data = self.user_data_fixup(user_data)
        except Exception as e:
            raise AXPlatformException("Failed while fixing up user-data: " + str(e))

        with open(USER_DATA_FILE_NEW, "w") as f:
            f.write(user_data)
        return USER_DATA_FILE_NEW

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def get_master_pd_volume_metadata(self):
        """
        Get's the metadata for the Master's persistent disk (EBS volume).
        """
        volume_metadata = self.client.describe_volumes(
            Filters=[
                {'Name': 'attachment.instance-id', 'Values': [self.master_instance.instance_id,]},
                {'Name': 'tag:Name', 'Values' : [self.cluster_name_id + "-master-pd"]}
            ])
        assert volume_metadata is not None, "Failed to retries volume_metadata"
        return volume_metadata

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def get_route_table_id(self):
        """
        Get's the route table used by the given cluster.
        """
        route_table_id = None
        response = self.client.describe_route_tables(
            Filters=[{'Name': 'tag:KubernetesCluster', 'Values': [self.cluster_name_id]}])
        assert len(response["RouteTables"]) == 1, "There should be a single route-table!"
        assert "RouteTableId" in response["RouteTables"][0], "RouteTableId not in response"

        route_table_id = response["RouteTables"][0]["RouteTableId"]
        logger.debug("Using route-table-id %s", route_table_id)
        return route_table_id

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def get_root_dev_attrs(self):
        assert self.master_instance, "Master instance not set"
        root_dev_id = None
        for dev in self.master_instance.block_device_mappings:
            if dev['DeviceName'] == self.master_instance.root_device_name:
                root_dev_id = dev['Ebs']['VolumeId']

                dev_metadata = self.client.describe_volumes(VolumeIds=[root_dev_id])
                root_dev_size = dev_metadata['Volumes'][0]['Size']
                root_dev_type = dev_metadata['Volumes'][0]['VolumeType']

                break

        assert self.master_instance.root_device_name and str(root_dev_size) and root_dev_type, "Failed to get root device attributes"
        logger.info("Root device attributes: %s, %s, %s", self.master_instance.root_device_name, str(root_dev_size), root_dev_type)
        self.attributes['root_dev_name'] = self.master_instance.root_device_name
        self.attributes['root_dev_size'] = str(root_dev_size)
        self.attributes['root_dev_type'] = root_dev_type

    def populate_attributes(self):
        """
        Collects attributes that will be persisted and used for spinning up the new master instance.
        Populates the "attributes" member dict with all the values.
        """
        # Upgrade might overwrite these attributes. Use them if set.
        # Otherwise get them from existing master instance.
        image_id = self.aws_image if self.aws_image else self.master_instance.image_id
        instance_profile = self.instance_profile if self.instance_profile else self.master_instance.iam_instance_profile["Arn"]
        self.attributes['image_id'] = image_id
        self.attributes['instance_type'] = self.master_instance.instance_type
        self.attributes['vpc_id'] = self.master_instance.vpc_id
        self.attributes['key_name'] = self.master_instance.key_name
        self.attributes['placement'] = self.master_instance.placement
        self.attributes['arn'] = instance_profile
        self.attributes['subnet_id'] = self.master_instance.subnet_id
        self.attributes['private_ip_address'] = self.master_instance.private_ip_address
        target_sgs = []
        for sg in self.master_instance.security_groups:
            target_sgs.append(sg["GroupId"])
        self.attributes['security_group_ids'] = target_sgs
        self.attributes['user_data_file'] = self.get_user_data()
        self.attributes['master_tags'] = self.master_instance.tags

        # Retrieve master-pd and master-eip from the volume_metadata
        volume_metadata = self.get_master_pd_volume_metadata()
        if volume_metadata['Volumes'] and volume_metadata['Volumes'][0]:
            if volume_metadata['Volumes'][0]['VolumeId']:
                vol_id = volume_metadata['Volumes'][0]['VolumeId']
                self.attributes['master_pd_id'] = vol_id
                self.attributes['master_pd_device'] = volume_metadata['Volumes'][0]['Attachments'][0]['Device']

            # Retrieve tags of master-pd. Get EIP from master.
            for tag in volume_metadata['Volumes'][0]['Tags']:
                if tag['Key'] == "kubernetes.io/master-ip":
                    master_eip = tag["Value"]
                    self.attributes['master_eip'] = master_eip
                    break
        assert self.attributes['master_pd_id'] is not None, "Failed to find Master's persistent disk"
        assert self.attributes['master_pd_device'] is not None, "Failed to find attachment info for Master's persistent disk"
        assert self.attributes['master_eip'] is not None, "Failed to find Master's Elastic IP"

        self.attributes['route_table_id'] = self.get_route_table_id()
        self.attributes['pod_cidr'] = self.cluster_config.get_master_pod_cidr()
        self.attributes['ebs_optimized'] = self.master_instance.ebs_optimized

        # Get root device attributes
        self.get_root_dev_attrs()

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def add_tags(self, instance):
        """
        Adds tags to the new master instance.

        :param instance: The new master ec2 instance.
        """
        response = self.client.create_tags(
            Resources=[instance.instance_id],
            Tags=self.attributes['master_tags']
        )

        logger.info("Attached tags to new master")

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def attach_eip(self, instance):
        """
        Attaches the EIP to the master instance.

        :param instance: The new master ec2 instance.
        """
        eip_meta = self.client.describe_addresses(PublicIps=[self.attributes['master_eip']])
        assert eip_meta is not None, "Failed to get details about EIP " + self.attributes['master_eip']
        assert eip_meta['Addresses'] and len(eip_meta['Addresses']) == 1, "Error getting EIP address details"
        response = self.client.associate_address(
            InstanceId=instance.instance_id,
            AllocationId=eip_meta['Addresses'][0]['AllocationId'],
            AllowReassociation=True
        )
        logger.info("Attached EIP to new master: %s", response['ResponseMetadata']['HTTPStatusCode'])

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def attach_volume(self, instance):
        """
        Attaches the EBS volume to the master instance.

        :param instance: The new master ec2 instance.
        """
        response = instance.attach_volume(
            VolumeId=self.attributes['master_pd_id'],
            Device=self.attributes['master_pd_device'])
        logger.info("Attached volume to new master: %s", response['ResponseMetadata']['HTTPStatusCode'])

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def replace_route(self, instance):
        response = self.client.replace_route(
            RouteTableId=self.attributes['route_table_id'],
            DestinationCidrBlock=self.attributes['pod_cidr'],
            InstanceId=instance.instance_id
        )
        logger.info("Replaced master route %s with %s: %s", self.attributes['pod_cidr'], instance.instance_id, response['ResponseMetadata']['HTTPStatusCode'])

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=5)
    def run_new_master(self, user_data):
        """
        Uses the boto APIs to run new instances of the master. Retries in case of failure.

        :param user_data: The user-data to use for the new instance.
        """
        try:
            response = self.client.run_instances(
                ImageId=self.attributes['image_id'],
                MinCount=1,
                MaxCount=1,
                KeyName=self.attributes['key_name'],
                UserData=user_data,
                InstanceType=self.attributes['instance_type'],
                Placement=self.attributes['placement'],
                IamInstanceProfile={"Arn": self.attributes['arn']},
                NetworkInterfaces=[
                    {
                    'DeviceIndex': 0,
                    'SubnetId': self.attributes['subnet_id'],
                    'PrivateIpAddress': self.attributes['private_ip_address'],
                    'AssociatePublicIpAddress': True,
                    'Groups': self.attributes['security_group_ids']
                    },
                ],
                BlockDeviceMappings=[
                    {
                        'DeviceName': self.attributes['root_dev_name'],
                        'Ebs': {
                            'VolumeSize': int(self.attributes['root_dev_size']),
                            'VolumeType': self.attributes['root_dev_type']
                        }
                    },
                    # Ephemeral devices to match kube-up behavior to get SSD attached.
                    {
                        'DeviceName': '/dev/sdc',
                        'VirtualName': 'ephemeral0'
                    },
                    {
                        'DeviceName': '/dev/sdd',
                        'VirtualName': 'ephemeral1'
                    },
                    {
                        'DeviceName': '/dev/sde',
                        'VirtualName': 'ephemeral2'
                    },
                    {
                        'DeviceName': '/dev/sdf',
                        'VirtualName': 'ephemeral3'
                    },
                ],
                EbsOptimized=self.attributes['ebs_optimized']
            )
            return response
        except Exception as e:
            logger.exception("Error running instances: %s", str(e))

    def launch_new_master(self):
        """
        Launches the new master instance.
        """
        logger.info("Launching new master ...")
        # Read the base64 encoded data and decode it before using it. AWS will
        # base64 encode it again.
        with open(self.attributes['user_data_file'], 'r') as user_data_file:
            user_data = base64.b64decode(user_data_file.read())

        response = self.run_new_master(user_data)
        new_master_id = response["Instances"][0]['InstanceId']
        logger.info("Waiting for new master %s to start", new_master_id)
        new_master = self.ec2.Instance(new_master_id)

        # Each call to ec2_instance.wait_until_running below will wait for a max of 15 minutes.
        # Give enough time for the instance to start...
        counter = 0
        while (counter < 2):
            try:
                new_master.wait_until_running()
                counter = counter + 1
            except botocore.exceptions.WaiterError as we:
                logger.debug("Still waiting for new master to run...")
                pass

        logger.info("New master with instance id %s is up!", new_master.instance_id)

        self.add_tags(new_master)
        self.attach_eip(new_master)
        self.attach_volume(new_master)
        self.replace_route(new_master)

        return new_master

    @retry(wait_fixed=2000)
    def wait_for_termination(self):
        """
        Waits the termination of the currently running master instance.
        """
        # check if master api server is alive and if not terminate master
        try:
            logger.info("Checking if Master API server is alive...")
            self.check_master_api_server()
            logger.info("Master API server is alive...")
        except Exception as e:
            if isinstance(e, urllib3.exceptions.HTTPError):
                logger.error("Got the following exception while trying to check master api server {}".format(e))
                logger.info("Assuming master is bad and terminating it...")
                self.terminate_master()
                logger.info("Done terminating master")
                return
            else:
                logger.warn("Got the following error from Kubernetes Master API Server {}. Looks like it is alive so ignoring this temporary error".format(e))

        logger.debug("Waiting for master termination signal ...")
        self.master_instance.wait_until_terminated()
        logger.info("Master down!")

    @retry(wait_exponential_multiplier=2000, stop_max_attempt_number=3, retry_on_exception=print_exception)
    def terminate_master(self):
        """
        Terminate current master instance and wait until it's done.
        """
        logger.info("Terminating master %s.", self.master_instance)
        self.client.terminate_instances(InstanceIds=[self.master_instance.instance_id])
        self.master_instance.wait_until_terminated()

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def stop_master(self):
        stop_master_requested = False
        master_instance_id = self.discover_master(state=[EC2InstanceState.Stopping, EC2InstanceState.Stopped])
        if master_instance_id:
            stop_master_requested = True

        if not stop_master_requested:
            master_instance_id = self.discover_master(state=["*"])
            if not master_instance_id:
                raise AXPlatformException("Cannot find master instance")
            try:
                self.client.stop_instances(InstanceIds=[master_instance_id])
            except ClientError as ce:
                if "UnsupportedOperation" in str(ce) and "StopInstances" in str(ce):
                    logger.warning("Master instance %s a spot instance, which cannot be stopped.")
                    return
                elif "IncorrectInstanceState" in str(ce):
                    # Master could be in "terminating", "terminated", or "stopped" state. It does not
                    # make sense that first 2 states could kick in, unless there is some human intervention
                    # so the code will stuck in waiting for master to go into "stopped" state, which is
                    # a good indication for checking manually
                    pass
                else:
                    raise ce
        logger.info("Waiting for master %s to get into state \"stopped\"", master_instance_id)
        while True:
            stopped_master = self.discover_master(state=[EC2InstanceState.Stopped])
            if stopped_master:
                logger.info("Master %s successfully stopped", master_instance_id)
                return
            else:
                time.sleep(5)

    @retry(retry_on_exception=default_aws_retry, wait_exponential_multiplier=2000, stop_max_attempt_number=3)
    def restart_master(self):
        started_master_id = self.discover_master(state=[EC2InstanceState.Running])
        if started_master_id:
            logger.info("Master %s is already running", started_master_id)
            return

        stopped_master_id = self.discover_master(state=[EC2InstanceState.Stopped])
        if not stopped_master_id:
            raise AXPlatformException("Cannot find a previously stopped master instance")

        # As we can always start a "stopped" instance, any other exception will be thrown out
        self.client.start_instances(InstanceIds=[stopped_master_id])

        logger.info("Waiting for master %s to get into state \"running\"", stopped_master_id)
        while True:
            running_master_id = self.discover_master(state=[EC2InstanceState.Running])
            if running_master_id:
                logger.info("Master %s successfully started", running_master_id)
                return
            else:
                time.sleep(5)

    def save_master_config(self, file_path):
        """
        Uploads the master attributes and user-data (in base64encoded format) into a directory
        in the s3 bucket.
        """
        with open(file_path, 'r') as user_data_file:
            user_data = user_data_file.read()
        self.cluster_info.upload_master_config_to_s3(self.attributes, user_data)

    def user_data_updated(self):
        """
        Get both old and new user data file content and compare them.
        Return True if they are different.
        """
        with open(USER_DATA_FILE_S3, "r") as f:
            old = f.read()
        with open(USER_DATA_FILE_NEW, "r") as f:
            new = f.read()
        return old != new

    def send_notification(self, code, message):
        try:
            self.event_notification_client.send_message_to_notification_center(
                code, detail={'message': "[master_manager] " + message})
        except Exception as exc:
            logger.exception("Failed to send event to notification center: %s", exc)
        return

    def run(self):
        """
        The main method for the MasterManager.
        """
        logger.info("Running the MasterManager!")
        attr_str = self.cluster_info.get_master_config(USER_DATA_FILE_S3)
        if attr_str is not None:
            self.attributes = json.loads(attr_str)
            self.attributes['user_data_file'] = USER_DATA_FILE_S3

        # Check if the master is running. Update the self.master_instance object.
        try:
            instance_id = self.discover_master()
            if instance_id is not None:
                self.master_instance = self.ec2.Instance(instance_id)
                logger.info("Master instance discovered: %s", self.master_instance.instance_id)

                # this will retry for a while and then throw an exception if master api server is unreachable
                self.check_master_api_server()

                if not self.attributes:
                    # This is needed only for first startup when cluster is created.
                    logger.debug("Populating attributes")
                    self.populate_attributes()
                    logger.debug("Saving master's config into S3")
                    self.save_master_config(USER_DATA_FILE_NEW)
                    logger.info("Master config uploaded to s3")
        except Exception as e:
            raise AXPlatformException("Failed to discover master: " + str(e))

        while(True):
            if self.master_instance is not None:
                self.wait_for_termination()
                message = "Master instance with id " + \
                    self.master_instance.instance_id + " terminated. A " + \
                    "new master instance will be created. This should " + \
                    "take a few minutes"
            else:
                logger.info("Master not running")
                message = "Master instance not found" + \
                    "A new master instance will be created. This should " + \
                    "take a few minutes."

            self.send_notification(CODE_PLATFORM_ERROR, message)
            new_master = self.launch_new_master()
            self.master_instance = self.ec2.Instance(new_master.instance_id)
            logger.info("New master instance %s running", self.master_instance.instance_id)
            self.send_notification(CODE_PLATFORM_CRITICAL, "New master " + \
                                   "instance with id {} started".format(
                                       self.master_instance.instance_id))
            logger.info("Wait for {} minutes before running checks...".format(WAIT_TIME_POST_RESTART_MIN))
            time.sleep(WAIT_TIME_POST_RESTART_MIN * const.SECONDS_PER_MINUTE)
            logger.info("Done waiting. Now back to checks")

    @retry_unless()
    def check_master_api_server(self):
            c = KubernetesApiClient()
            c.api.read_namespaced_service("default", "kubernetes")

    def upgrade(self):
        """
        Entry point for master upgrade.
        Support upgrade of:
            - Kubernetes versions;
            - AMI image;
            - Selected list of kube_env variables.
        """
        logger.info("Starting master upgrade!")
        ami_name = os.getenv("AX_AWS_IMAGE_NAME")
        assert ami_name, "Fail to detect AMI name from environ"
        ami_id = AMI(aws_region=self.region, aws_profile=self.profile).get_ami_id_from_name(ami_name=ami_name)
        logger.info("Using ami %s for new master", ami_id)

        s3_data = self.cluster_info.get_master_config(USER_DATA_FILE_S3)
        if s3_data is None:
            attr = None
        else:
            attr = json.loads(self.cluster_info.get_master_config(USER_DATA_FILE_S3))
        instance_id = self.discover_master()
        terminating = False
        launching = False
        if instance_id is None:
            # This is possible if previous upgrade fails after termination but before new master start.
            # Simply restart master in this case.
            # This could also happen when master crashes in the first place and upgrade is started.
            # We would use old config to start master and rerun upgrade again.
            logger.info("No running master. S3 attr %s.", USER_DATA_FILE_S3)
            assert attr is not None, "No master instance and no master config."
            self.attributes = attr
            self.attributes['user_data_file'] = USER_DATA_FILE_S3

            self.ensure_master_tags()
            self.save_master_config(USER_DATA_FILE_S3)
            launching = True
        else:
            self.master_instance = self.ec2.Instance(instance_id)
            logger.info("Running master %s.", instance_id)
            self.aws_image = ami_id
            self.instance_profile = AXClusterInstanceProfile(self.cluster_name_id, aws_profile=self.profile).get_master_arn()
            self.populate_attributes()
            master_tag_updated = self.ensure_master_tags()
            # TODO: Possible race here.
            # If upgrade is interrupted after config saving but before master termination,
            # Next upgrade attempt would assume master is already upgraded.
            # Manually hack to terminate instance is needed then.
            if attr != self.attributes or self.user_data_updated() or master_tag_updated:
                self.save_master_config(USER_DATA_FILE_NEW)
                terminating = True
                launching = True

        if terminating:
            self.terminate_master()
            logger.info("Done terminating %s", instance_id)
        if launching:
            logger.info("Done launching %s", self.launch_new_master())

    def ensure_master_tags(self):
        """
        During upgrade, we need to ensure master has AXClusterNameID,AXCustomerID,AXTier tags (#23)
        :return: True if we updated master tags
        """
        for tag in self.attributes['master_tags']:
            if tag["Key"] == "AXTier":
                # Master has updated tags
                return False

        self.attributes['master_tags'] += [
            {
                "Key": "AXCustomerID",
                "Value": AXCustomerId().get_customer_id()
            },
            {
                "Key": "AXTier",
                "Value": "master"
            },
            {
                "Key": "AXClusterNameID",
                "Value": self.cluster_name_id
            },
        ]
        return True
