# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
AXMon state machine
"""

from transitions.extensions import LockedMachine as Machine


class AXMonState(object):
    UNKNOWN = 'UNKNOWN' # Initial AXMon state on boot
    STOPPED = 'STOPPED'
    STARTING = 'STARTING'
    RUNNING = 'RUNNING'
    RUNNING_SCALING = 'RUNNING_SCALING'
    STOPPING = 'STOPPING'


AXMON_STATES = [state for state in dir(AXMonState) if not state.startswith('_')]


class AXMonStateMachine(object):

    def __init__(self):
        self.machine = Machine(name='AXMon', model=self, states=AXMON_STATES, initial=AXMonState.UNKNOWN)

        self.machine.add_transition('detected_running',  [AXMonState.UNKNOWN, AXMonState.STOPPED, AXMonState.STARTING, AXMonState.RUNNING], AXMonState.RUNNING)
        self.machine.add_transition('detected_upgrade', AXMonState.UNKNOWN, AXMonState.STARTING)
        self.machine.add_transition('detected_stopped', AXMonState.UNKNOWN, AXMonState.STOPPED)

        self.machine.add_transition('cluster_start_begin',[AXMonState.STOPPED, AXMonState.RUNNING], AXMonState.STARTING)
        self.machine.add_transition('cluster_start_end',
                                    # TODO: RUNNING is here because axmon state can transition to RUNNING
                                    # from both cluster_start and the @state property. If cluster_start is
                                    # broken into separate platform and devops pieces, RUNNING can/should
                                    # be removed from the source state.
                                    [AXMonState.STARTING, AXMonState.RUNNING],
                                    AXMonState.RUNNING)
        self.machine.add_transition('cluster_start_end',
                                    [AXMonState.RUNNING_SCALING],
                                    AXMonState.RUNNING_SCALING)
        self.machine.add_transition('cluster_stop_begin', '*', AXMonState.STOPPING)
        self.machine.add_transition('cluster_stop_end', AXMonState.STOPPING, AXMonState.STOPPED)

        self.machine.add_transition('cluster_upgrade_begin',
                                    [AXMonState.RUNNING, AXMonState.STOPPED, AXMonState.STARTING],
                                    AXMonState.STOPPING)
        # NOTE: There is no need for cluster_upgrade_end since axmon will be killed by cluster-upgrade, at which
        # point, the state of AXMon will be determined on boot up of the next axmon

        self.machine.add_transition('cluster_scaling_start', AXMonState.RUNNING, AXMonState.RUNNING_SCALING)
        self.machine.add_transition('cluster_scaling_end', AXMonState.RUNNING_SCALING, AXMonState.RUNNING)
