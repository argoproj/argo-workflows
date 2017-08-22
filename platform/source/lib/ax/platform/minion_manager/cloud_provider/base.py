#!/usr/bin/env python

"""
Define the base class for the minion-manager. Cloud provider specific
implementations should derive from this.
"""

import abc
from future.utils import with_metaclass

class MinionManagerBase(with_metaclass(abc.ABCMeta)):
    """ Base class for MinionManager. """
    _scaling_groups = []
    _region = None

    def __init__(self, scaling_groups, region):
        self._scaling_groups = scaling_groups
        self._region = region

    @abc.abstractmethod
    def run(self):
        """Main method for the minion-manager functionality."""
        return

    @abc.abstractmethod
    def check_scaling_group_instances(self, scaling_group):
        """
        Checks whether desired number of instances are running in a scaling
        group.
        """
        return

    @abc.abstractmethod
    def update_scaling_group(self, scaling_group, new_bid_info):
        """
        Updates the scaling group config.
        """
        return

