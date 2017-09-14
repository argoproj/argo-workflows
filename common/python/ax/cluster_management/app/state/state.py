# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


"""
The cluster state machine provides a mechanism to avoid cluster administrator initiate
an inappropriate cluster management operation when cluster is not ready for that. For
example, cluster upgrader would assume cluster is in running state as it needs to talk
with cluster master for upgrade. If the cluster is paused, or the cluster is even in
its install phase, we need to abort upgrade.

Implementation of cluster state transitions. Note that is not a constantly running
state machine, and needs an external blob store, such as S3 to check/persist cluster
state. This class is used by cluster management apps such as cluster pauser, upgrader,
etc.

This implementation assumes there is only 1 instance of the state machine manipulating
cluster state, as there is no proper locking mechanism in normal blob stores such as S3,
nor is such locking mechanism needed in use cases (e.g. there is only one person upgrading
cluster at one time)

Cluster has 2 types of states:
    - Mutating state (Installing, Pausing, Upgrading, Resuming, Uninstalling)
    - Stable state (Running, Paused, Unknown)

Mutating state can be triggered from stable state, in addition, for backward compatibility,
if a cluster is currently having no state record (Unknown), we can transit to a mutating
state as well. One exception is that cluster can enter uninstalling from any state, as
we support force uninstall.

Stable state can be triggered from mutating state ONLY
"""

from transitions import Machine

from ax.platform.ax_cluster_info import AXClusterInfo
from .consts import ClusterState


class ClusterStateMachine(object):
    def __init__(self, cluster_name_id, cloud_profile):
        self._cluster_info = AXClusterInfo(cluster_name_id=cluster_name_id, aws_profile=cloud_profile)
        current_state = self._cluster_info.download_cluster_current_state() or ClusterState.UNKNOWN

        self.machine = Machine(model=self, states=ClusterState.VALID_CLUSTER_STATES, initial=current_state)
        self._add_transitions()

    def _add_transitions(self):
        self.machine.add_transition(
            trigger="do_install",
            source=[ClusterState.UNKNOWN, ClusterState.INSTALLING],
            dest=ClusterState.INSTALLING
        )

        self.machine.add_transition(
            trigger="done_install",
            source=ClusterState.INSTALLING,
            dest=ClusterState.RUNNING
        )

        self.machine.add_transition(
            trigger="do_pause",
            source=[ClusterState.UNKNOWN, ClusterState.RUNNING, ClusterState.PAUSING],
            dest=ClusterState.PAUSING
        )

        self.machine.add_transition(
            trigger="done_pause",
            source=ClusterState.PAUSING,
            dest=ClusterState.PAUSED
        )

        self.machine.add_transition(
            trigger="do_resume",
            source=[ClusterState.UNKNOWN, ClusterState.PAUSED, ClusterState.RESUMING],
            dest=ClusterState.RESUMING
        )

        self.machine.add_transition(
            trigger="done_resume",
            source=ClusterState.RESUMING,
            dest=ClusterState.RUNNING
        )

        self.machine.add_transition(
            trigger="do_upgrade",
            source=[ClusterState.UNKNOWN, ClusterState.RUNNING, ClusterState.UPGRADING],
            dest=ClusterState.UPGRADING
        )

        self.machine.add_transition(
            trigger="done_upgrade",
            source=ClusterState.UPGRADING,
            dest=ClusterState.RUNNING
        )

        self.machine.add_transition(
            trigger="do_uninstall",
            source=[
                ClusterState.RUNNING, ClusterState.UNKNOWN, ClusterState.UPGRADING,
                ClusterState.PAUSING, ClusterState.PAUSED, ClusterState.RESUMING,
                ClusterState.UNINSTALLING, ClusterState.INSTALLING
            ],
            dest=ClusterState.UNINSTALLING
        )

    @property
    def current_state(self):
        return self.state

    def is_installing(self):
        return self.current_state == ClusterState.INSTALLING

    def is_running(self):
        return self.current_state == ClusterState.RUNNING

    def is_pausing(self):
        return self.current_state == ClusterState.PAUSING

    def is_paused(self):
        return self.current_state == ClusterState.PAUSED

    def is_upgrading(self):
        return self.current_state == ClusterState.UPGRADING

    def is_resuming(self):
        return self.current_state == ClusterState.RESUMING

    def is_uninstalling(self):
        return self.current_state == ClusterState.UNINSTALLING

    def is_unknown(self):
        return self.current_state == ClusterState.UNKNOWN

    def persist_state(self):
        self._cluster_info.upload_cluster_current_state(self.current_state)
