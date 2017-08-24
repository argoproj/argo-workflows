"""The file has unit tests for the cloud broker."""

import unittest
import pytest
from ax.platform.minion_manager.cloud_broker import Broker
from ax.platform.minion_manager.cloud_provider.aws.aws_minion_manager import AWSMinionManager


class BrokerTest(unittest.TestCase):
    """
    Tests for cloud broker.
    """

    def test_get_impl_object(self):
        """
        Tests that the get_impl_object method works as expected.
        """

        # Verify that a minion-manager object is returned for "aws"
        mgr = Broker.get_impl_object("aws", ["asg_1"], "us-west-2")
        assert mgr is not None,"No minion-manager returned!"
        assert isinstance(mgr, AWSMinionManager), "Wrong minion-manager instance returned."

        # For non-aws clouds, a NotImplementedError is returned.
        with pytest.raises(NotImplementedError):
            mgr = Broker.get_impl_object("google", [], "")
