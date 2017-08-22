#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

# TODO: package it in a class

import logging
import time

from ax.util.const import COLOR_GREEN, COLOR_NORM
import boto3
from retrying import retry

from .util import default_aws_retry


logger = logging.getLogger(__name__)


# AWS is not consistent w.r.t. minimum sizes of the volume types.
VOL_TYPE_MIN_SIZE_GB = {"io1": 4, "gp2": 1}
DISK_CREATION_WAIT_TIME_SEC = 30

def delete_tagged_ebs(aws_profile, tag_key, tag_value, region):
    logger.info("Deleting all cluster remaining volumes ...")
    ec2 = boto3.Session(profile_name=aws_profile).client("ec2", region_name=region)

    # aws ec2 does not support batch volume deletion, since an existing cluster would have hundreds
    # of volumes, and deleting them one by one would result in a potential RequestLimitExceeded error
    # we need to backoff if this happens
    @retry(retry_on_exception=default_aws_retry,
           wait_exponential_multiplier=1000,
           wait_exponential_max=64000,
           stop_max_attempt_number=10)
    def desc_vol_with_backoff(tag_key, tag_value):
        # Not setting paginator as describe_volumes can return at most 1000 volumes
        # which is more than enough for our use
        return ec2.describe_volumes(
                    Filters=[
                        {"Name": "tag-key", "Values": [tag_key]},
                        {"Name": "tag-value", "Values": [tag_value]},
                    ]
                )["Volumes"]

    @retry(retry_on_exception=default_aws_retry,
           wait_exponential_multiplier=1000,
           wait_exponential_max=64000,
           stop_max_attempt_number=10)
    def del_vol_with_backoff(vid):
        logger.info("Deleting ebs volume %s", vid)
        ec2.delete_volume(VolumeId=vid)

    while True:
        volumes = desc_vol_with_backoff(tag_key, tag_value)
        if len(volumes) == 0:
            logger.info("%sAll cluster ebs volumes deleted ...%s", COLOR_GREEN, COLOR_NORM)
            return
        logger.info("%s ebs volumes left", len(volumes))
        for v in volumes:
            vid = v["VolumeId"]
            if v["State"] == "available":
                del_vol_with_backoff(vid)
            else:
                logger.warning("Volume %s has state \"%s\"", vid, v["State"])
        # back-off a bit
        time.sleep(5)

# RawEBSVolumes are EBS volumes created and destroyed directly by AX. They are not mounted
# and do not have any filesystem on it.
class RawEBSVolume(object):
    def __init__(self, ec2_client, ax_volume_id, cluster_name_id):
        """
        Constructor for the RawEBSVolume object.

        :param ec2_client: The boto client for accessing AWS.
        :param ax_volume_id: AX generated unique id of the volume. This gets added as a tag to the volume.
        :param cluster_name_id: Kubernetes cluster name.
        :params kwargs: Additional arguments required for the volume.
        """
        self.ec2_client = ec2_client
        self.cluster_name_id = cluster_name_id
        self.ax_volume_id = ax_volume_id
        self.type = None
        self.size = None
        self.resource_id = None
        self.zone = None
        self.misc_vol_opts = {}

    def populate_attrs(self, vol_opts):
        """
        Populates the attributes required for creating the EBS volume.
        """
        assert "volume_type" in vol_opts, "Type of the volume io1/gp2 absent"
        self.type = vol_opts["volume_type"]
        assert self.type in ("io1", "gp2"), "Only io1 or gp2 volume types are supported"

        assert "size_gb" in vol_opts, "Size should be specified in GB"
        self.size = int(vol_opts["size_gb"])
        assert self.size >= VOL_TYPE_MIN_SIZE_GB[self.type]

        assert "zone" in vol_opts, "Availability zone not specified"
        self.zone = vol_opts["zone"]

        if "iops" in vol_opts:
            self.misc_vol_opts['iops'] = int(vol_opts['iops'])

        if "axrn" in vol_opts:
            self.misc_vol_opts['axrn'] = vol_opts['axrn']

    def create(self, vol_opts):
        """
        Creates the EBS volume on AWS if it doesn't exist. The ax_volume_id is used for uniquely
        identifying the volume.

        :returns AWS created resource_id. None if volume creation failed.
        """
        # If a Volume exists with this name, return it.
        volume_info = self.query_aws_volume_info()
        if volume_info:
            self.resource_id = volume_info.get("VolumeId", None)
            logger.info("Found existing volume: %s", self)
            return self.resource_id

        self.populate_attrs(vol_opts)

        logger.info("Creating EBS volume: %s", self)

        # Create the volume with retries.
        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
        def _create_ebs_volume():
            if self.type == "gp2":
                return self.ec2_client.create_volume(Size=self.size,
                                                     AvailabilityZone=self.zone,
                                                     VolumeType=self.type)
            else:
                return self.ec2_client.create_volume(Size=self.size,
                                                     AvailabilityZone=self.zone,
                                                     VolumeType=self.type,
                                                     Iops=self.misc_vol_opts.get('iops', 0))

        response = _create_ebs_volume()
        if response is None:
            logger.info("Failed to create EBS volume")
            return None

        # Store the volume id.
        self.resource_id = response["VolumeId"]

        # Create tags.
        tags = [{'Key': 'AXVolumeID', 'Value': self.ax_volume_id}, \
                {'Key': 'KubernetesCluster', 'Value': self.cluster_name_id}, \
                {'Key': 'axrn', 'Value': self.misc_vol_opts.get('axrn', 'VOL_AXRN')}]
        self.create_or_update_tags(tags)

        # Verify that the disk actually exists (to guard against spurious AWS behavior where
        # disk creates succeed but actually it doesn't exist).
        self.wait_for_creation()
        logger.info("Created EBS volume with name: %s", self)

        return self.resource_id

    def create_or_update_tags(self, tags):
        """
        Create or update tags for the AWS volume.

        :param: tags: The dictionary of the key-value pairs to use as tags.
        """
        @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
        def _create_tags():
            assert self.resource_id is not None, "Volume resource id is none!"
            return self.ec2_client.create_tags(Resources=[self.resource_id], Tags=tags)

        if self.resource_id is None:
            volume_info = self.query_aws_volume_info()
            self.resource_id = volume_info.get("VolumeId", None)
        _create_tags()

    def wait_for_creation(self):
        """
        Waits for the raw disk to be created in AWS and the exists() call to return TRUE.
        """
        attempts = 10
        created = self.exists()
        while not created and attempts > 0:
            try:
                created = self.exists()
            finally:
                attempts = attempts - 1
                logger.info("Waiting for disk creation to complete ...")
                time.sleep(DISK_CREATION_WAIT_TIME_SEC)
        assert created, "Volume with resource_id " + self.resource_id + " not found!"

    def delete(self):
        """
        Deletes the EBS volume from AWS.
        """
        volume_info = self.query_aws_volume_info()
        if volume_info is None:
            logger.info("Volume not found: %s", self)
            return

        resource_id = volume_info.get("VolumeId", None)
        if resource_id is None:
            logger.info("ResourceId not found for volume: %s", self)
            return

        response = self.ec2_client.delete_volume(VolumeId=resource_id)
        if response['ResponseMetadata']['HTTPStatusCode'] == 200:
            logger.info("Deleted EBS volume %s: ", self)

    @retry(wait_exponential_multiplier=1000, stop_max_attempt_number=5)
    def query_aws_volume_info(self):
        """
        Queries AWS for the volume name and returns the ax_volume_id if the volume exists in AWS.

        :returns AWS created ax_volume_id. None if volume doesn't exist.
        """
        response = self.ec2_client.describe_volumes(Filters=[
            {'Name': 'tag:AXVolumeID', 'Values': [self.ax_volume_id]},
            {'Name': 'tag:KubernetesCluster', 'Values': [self.cluster_name_id]}
        ])

        for volume in response.get("Volumes", None):
            for tag in volume["Tags"]:
                if tag["Key"] == "KubernetesCluster" and tag["Value"] == self.cluster_name_id:
                    return volume

        return None

    def exists(self):
        """
        :returns Whether the volume exists or not.
        """
        return self.query_aws_volume_info() != None

    def __str__(self):
        return "Raw EBS Volume:: AX volume_id {}, resource_id {}, size: {}, type: {}, zone: {}, misc: {}". \
            format(self.ax_volume_id, self.resource_id, self.size, self.type, self.zone, self.misc_vol_opts)

