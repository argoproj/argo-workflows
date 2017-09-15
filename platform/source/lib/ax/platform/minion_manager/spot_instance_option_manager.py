#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import logging

from ax.platform.ax_asg import AXUserASGManager
from ax.platform.cluster_config import SpotInstanceOption
from ax.util.singleton import Singleton
from future.utils import with_metaclass


logger = logging.getLogger(__name__)


class SpotInstanceOptionManager(with_metaclass(Singleton, object)):
    def __init__(self, cluster_name_id, region):
        self._cluster_name_id = cluster_name_id
        self._region = region

    def option_to_asgs(self, option):
        """
        Returns the names of the ASGs based on the provided config option.
        """
        assert option in SpotInstanceOption.VALID_SPOT_INSTANCE_OPTIONS, \
            "{} is not a valid spot instance option".format(option)
        asg_manager = AXUserASGManager(self._cluster_name_id, self._region)
        if option == SpotInstanceOption.ALL_SPOT:
            asg_names = asg_manager.get_all_asg_names()
            return asg_names
        elif option == SpotInstanceOption.NO_SPOT:
            return []
        else:
            return [asg_manager.get_variable_asg()["AutoScalingGroupName"]]
        return

    def asgs_to_option(self, asgs):
        """
        Returns the config option based on the names of the ASGs.
        """
        asg_manager = AXUserASGManager(self._cluster_name_id, self._region)
        all_asg_names = asg_manager.get_all_asg_names()
        if asgs is None or len(asgs) == 0:
            return SpotInstanceOption.NO_SPOT
        elif set(asgs) == set(all_asg_names):
            return SpotInstanceOption.ALL_SPOT
        else:
            return SpotInstanceOption.PARTIAL_SPOT
        return
