#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import pytest
import unittest

from ax.cloud.aws.ebs import RawEBSVolume
import boto3
from moto import mock_sts, mock_ec2


class RawEBSVolumeTest(unittest.TestCase):
    aws_profile = "dev"
    region = "us-west-2"
    session = boto3.Session(region_name=region, profile_name=aws_profile)

    @mock_sts
    @mock_ec2
    def test_raw_ebs_volume(self):
        volume_opts = {}
        volume_opts["size_gb"] = 30
        volume_opts["type"] = "ebs"
        volume_opts["volume_type"] = "io1"
        volume_opts["zone"] = "us-west-2b"
        volume_opts["iops"] = 500

        self.ec2 = self.session.client('ec2')
        ebs_volume = RawEBSVolume(self.ec2, "test-volume", "my-cluster-id")

        # Verify that the volume doesn't exist.
        assert ebs_volume.exists() is False

        # Create a new volume
        volume_id = ebs_volume.create(volume_opts)
        assert volume_id is not None, "Failed to create volume!"

        # Verify that the volume exists now.
        assert ebs_volume.exists() is True

        # Try creating the same volume again. This should return the same volume id.
        ebs_volume_2 = RawEBSVolume(self.ec2, "test-volume", "my-cluster-id")
        volume_id_2 = ebs_volume_2.create(volume_opts)
        assert volume_id_2 == volume_id, "Multiple volumes of the same name created!"

        # Verify that the volume exists now.
        assert ebs_volume_2.exists() is True

        # Verify that deletes succeed.
        ebs_volume.delete()
        return

    @mock_sts
    @mock_ec2
    def test_volume_type(self):
        volume_opts = {}
        volume_opts["size_gb"] = 30
        volume_opts["type"] = "ebs"
        volume_opts["volume_type"] = "io1"
        volume_opts["zone"] = "us-west-2b"

        self.ec2 = self.session.client('ec2')

        ebs_volume = RawEBSVolume(self.ec2, "test-volume", "my-cluster-id")
        for vtype in ["standard", "sc1", "st1", "blah"]:
            volume_opts["volume_type"] = vtype
            with pytest.raises(AssertionError):
                ebs_volume.create(volume_opts)

        # io1 and gp2 are allowed
        volume_opts["volume_type"] = "io1"
        ebs_volume.create(volume_opts)

        volume_opts["volume_type"] = "gp2"
        ebs_volume.create(volume_opts)
