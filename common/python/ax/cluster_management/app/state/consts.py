# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


class ClusterState:
    INSTALLING = "Installing"
    RUNNING = "Running"
    PAUSING = "Pausing"
    PAUSED = "Paused"
    RESUMING = "Resuming"
    UPGRADING = "Upgrading"
    UNINSTALLING = "Uninstalling"

    # Used before cluster state is firstly set. i.e. an stale cluster that does not have
    # state information, or a cluster before installation. Cluster can transit from
    # this state to any other state
    UNKNOWN = "Unknown"

    VALID_CLUSTER_STATES = [INSTALLING, RUNNING, PAUSING, PAUSED, RESUMING, UPGRADING, UNINSTALLING, UNKNOWN]

