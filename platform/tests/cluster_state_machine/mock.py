# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from transitions import Machine
from ax.cluster_management.app.state import ClusterStateMachine, ClusterState


class ClusterStateMachineMock(ClusterStateMachine):
    def __init__(self, current_state):
        self.machine = Machine(model=self, states=ClusterState.VALID_CLUSTER_STATES, initial=current_state)
        self._add_transitions()

    def persist_state(self):
        pass
