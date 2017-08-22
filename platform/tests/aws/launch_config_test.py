#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import zlib
import unittest

from ax.aws.launch_config import LaunchConfig

test_lc = {
    "ImageId": "ami-91a623f1",
    "KeyName": "ax-dev-test-vpc",
    "SecurityGroups": ["sg-39d8a441"],
    "InstanceType": "m3.medium",
    "BlockDeviceMappings":  [
        {
            "DeviceName": "/dev/sdz",
            "VirtualName": "ephemeral0"
        },
    ],
}

class LaunchConfigTest(unittest.TestCase):
    # TODO: Get unique name for concurrent tests.
    lc_name_old = "lc-test"
    lc_name_new = "lc-test-new"
    user_data_old = "user data old"
    user_data_new = "user data new"
    profile = "dev"

    @classmethod
    def setupClass(cls):
        LaunchConfig(cls.lc_name_old, aws_profile=cls.profile).delete()
        LaunchConfig(cls.lc_name_new, aws_profile=cls.profile).delete()

    @classmethod
    def tearDownClass(cls):
        LaunchConfig(cls.lc_name_old, aws_profile=cls.profile).delete()
        LaunchConfig(cls.lc_name_new, aws_profile=cls.profile).delete()

    def test_launch_config(self):
        # Create LC and verify data.
        lc = LaunchConfig(self.lc_name_old, aws_profile=self.profile)
        test_lc["UserData"] = self.user_data_old
        lc.create(test_lc)
        user_data = lc.get()["UserData"]
        assert user_data == self.user_data_old, "User data mismatch {} {}".format(user_data, self.user_data_old)
        lc.delete()

        # Make sure it's really deleted
        config = lc.get()
        assert config is None, "Still exist after delete {}".format(config)

        # Create LC and verify data, with compressed user data.
        comp = zlib.compressobj(9, zlib.DEFLATED, zlib.MAX_WBITS | 16)
        test_lc["UserData"] = comp.compress(self.user_data_old) + comp.flush()
        lc.create(test_lc)
        user_data = zlib.decompressobj(32 + zlib.MAX_WBITS).decompress(lc.get()["UserData"])
        assert user_data == self.user_data_old, "User data mismatch {} {}".format(user_data, self.user_data_old)

        # Make a updated copy of launch config and verify it's updated correctly.
        comp = zlib.compressobj(9, zlib.DEFLATED, zlib.MAX_WBITS | 16)
        test_lc["InstanceType"] = "m3.large"
        test_lc["UserData"] = comp.compress(self.user_data_new) + comp.flush()
        new_lc = lc.copy(self.lc_name_new, test_lc, delete_old=True)
        config = new_lc.get()
        user_data = zlib.decompressobj(32 + zlib.MAX_WBITS).decompress(config["UserData"])
        assert config["InstanceType"] == "m3.large", "New config {}".format(config)
        assert user_data == self.user_data_new, "New user data {} != {}".format(user_data, self.user_data_new)
        config = lc.get()
        assert config is None, "Still exist after copy and delete {}".format(config)

        # Clean up.
        new_lc.delete()
