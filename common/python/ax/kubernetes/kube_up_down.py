#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Wrapper for kube-up and kube-down
"""

import copy
import os
import subprocess
import logging

logger = logging.getLogger(__name__)


class KubeUpDown(object):
    """
    Class to access kube-up.sh and kube-down.sh
    """
    def __init__(self, root, env):
        """
        :param root: Root directory for kubernetes
        :param env: dict of Kubernetes environment variables.
        """
        self._root = root
        self._env = copy.deepcopy(env)
        self._env.update(os.environ)

    def up(self):
        """
        Call kube-up.sh
        """
        logger.debug("Calling kube-up with %s", self._env)
        subprocess.check_call(["cluster/kube-up.sh"], env=self._env, cwd=self._root)

    def down(self):
        """
        Call kube-down.sh
        """
        logger.debug("Calling kube-down with %s", self._env)
        subprocess.check_call(["cluster/kube-down.sh"], env=self._env, cwd=self._root)
