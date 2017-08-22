#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#

"""
Try to not add things here.
This is gross hack. We need a better way to ssh or run tasks on other hosts.
"""

import subprocess
import logging

logger = logging.getLogger(__name__)


def send_file(local, host, remote, user=None, key=None):
    """

    :param local:
    :param host:
    :param remote:
    :param user:
    :param key:
    :return:
    """
    # For now use scp
    scp_cmd = ["scp"]
    if key is not None:
        scp_cmd += ["-i", str(key)]
    scp_cmd += [local]
    if user is None:
        scp_cmd += ["%s:%s" % (host, remote)]
    else:
        scp_cmd += ["%s@%s:%s" % (user, host, remote)]
    subprocess.check_output(scp_cmd)


def run_command(cmd, host, user=None, key=None):
    """
    For now just call ssh command.

    :param cmd: command line
    :param host:
    :param user:
    :param key:
    :return:
    """
    # Force to use list, not string.
    # assert isinstance(cmd, list)

    ssh_cmd = ["ssh"]
    if key is not None:
        ssh_cmd += ["-i", str(key)]
    if user is None:
        ssh_cmd += [host]
    else:
        ssh_cmd += ["%s@%s" % (user, host)]

    ssh_cmd += cmd
    logger.debug("Run command [%s]", ssh_cmd)
    return subprocess.check_output(ssh_cmd)
