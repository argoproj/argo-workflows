#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#
import unittest
import logging
from ax.platform.volumes import VolumeManager, Volume, VolumePool
from ax.platform.exceptions import AXVolumeException, AXVolumeExistsException

logger = logging.getLogger("ax")

class VolumeManagerTests(unittest.TestCase):
    """
    Test for volume manager
    Requires kubectl proxy
    """
    @classmethod
    def setUpClass(self):
        pass

    @classmethod
    def tearDownClass(self):
        pass

    def test_1(self):
        """
        Test volume create conditions used by volume pools
        """
        manager = VolumeManager()
        manager.create("testvol", "1")
        with self.assertRaises(AXVolumeExistsException):
            manager.create("testvol", "1")

        manager.add_ref("testvol", "ref1")
        with self.assertRaises(AXVolumeException):
           manager.delete("testvol")

        # now make the reservation exclusive. this should pass
        manager.add_ref("testvol", "ref1", exclusive=True)

        # try to add a new ref now
        with self.assertRaises(AXVolumeException):
            manager.add_ref("testvol", "ref2")

        # delete ref1
        manager.delete_ref("testvol", "ref1")

        # test shared state
        manager.add_ref("testvol", "ref3")
        manager.add_ref("testvol", "ref4")

        # now try to put an exclusive lock ref
        with self.assertRaises(AXVolumeException):
            manager.add_ref("testvol", "ref5", exclusive=True)

        # now try to convert ref3 to exclusive
        with self.assertRaises(AXVolumeException):
            manager.add_ref("testvol", "ref3", exclusive=True)

        # now remove ref4
        manager.delete_ref("testvol", "ref4")

        # make ref3 exclusive
        manager.add_ref("testvol", "ref3", exclusive=True)

        # delete ref and then delete vol
        manager.delete_ref("testvol", "ref3")
        manager.delete("testvol")

    def test_2(self):
        """
        Basic test for volume pools
        """
        manager = VolumeManager()
        manager.create_pool("testpool1", "1", None)
        volname1 = manager.get_from_pool("testpool1", "test1")
        logger.debug("Got the following volume for test1 {}".format(volname1))
        volname2 = manager.get_from_pool("testpool1", "test2")
        logger.debug("Got the following volume for test2 {}".format(volname2))
        manager.put_in_pool("testpool1", volname1)
        manager.put_in_pool("testpool1", volname2)
        manager.delete_pool("testpool1")

    def test_3(self):
        """
        Test error conditions in volume pools
        """
        manager = VolumeManager()
        manager.create_pool("testpool2", "1", None)
        volname1 = manager.get_from_pool("testpool2", "t1")
        logger.debug("Got the following volume {}".format(volname1))

        # get from pool with the same ref should give a different vol
        volname2 = manager.get_from_pool("testpool2", "t1")
        self.assertNotEquals(volname1, volname2)
        manager.put_in_pool("testpool2", volname2)

        # now try to delete a pool with an active reference
        with self.assertRaises(AXVolumeException):
            manager.delete_pool("testpool2")
        logger.debug(manager.get_pools_for_unit_test())

        # put an invalid pvc back in pool
        with self.assertRaises(AssertionError):
            manager.put_in_pool("testpool2", "somevolname")
        manager.put_in_pool("testpool2", volname1)

        # delete a non existent pool should not raise an error
        manager.delete_pool("testpool")
        manager.delete_pool("testpool2")

        # now that pool is deleted, get from it
        with self.assertRaises(KeyError):
            manager.get_from_pool("testpool2", "t2")

    def test_4(self):
        """
        Test conditions that involve marking a volume for deletion
        """
        manager = VolumeManager()
        manager.create_pool("testdeletion", "2", None)
        volname1 = manager.get_from_pool("testdeletion", "ref1")

        # this resizes the pool causing volname1 to be marked for deletion
        manager.create_pool("testdeletion", "3", None)
        volname2 = manager.get_from_pool("testdeletion", "ref2")

        # volname1 should not be deleted
        manager.put_in_pool("testdeletion", volname1)

        # volname2 is put back in pool
        manager.put_in_pool("testdeletion", volname2)

        # volume pool can be deleted as it has no refs
        manager.delete_pool("testdeletion")






