# -*- coding: utf-8 -*-
#
# Copyright 2016 Applatix, Inc. All rights reserved.
#

"""
adc state machine
"""

from transitions.extensions import LockedMachine as Machine


class ADCState(object):
    UNKNOWN = 'UNKNOWN' # Initial AXMon state on boot
    STARTING = 'STARTING'
    RUNNING = 'RUNNING'
    SUSPENDED_ALLOW_NEW = 'SUSPENDED_ALLOW_NEW'
    SUSPENDED_NO_NEW = 'SUSPENDED_NO_NEW'
    STOPPED = 'STOPPED'

ADC_STATES = [state for state in dir(ADCState) if not state.startswith('_')]


class ADCStateMachine(object):

    def __init__(self):
        running_states = [ADCState.RUNNING,
                          ADCState.SUSPENDED_ALLOW_NEW,
                          ADCState.SUSPENDED_NO_NEW]

        self.machine = Machine(name='ADC', model=self, states=ADC_STATES, initial=ADCState.UNKNOWN)

        self.machine.add_transition('init_starting', [ADCState.UNKNOWN], ADCState.STARTING)
        self.machine.add_transition('done_starting', [ADCState.STARTING], ADCState.RUNNING)
        self.machine.add_transition('shutdown',
                                    [ADCState.STARTING, ADCState.STOPPED] + running_states,
                                    ADCState.STOPPED)
        self.machine.add_transition('request_running', running_states, ADCState.RUNNING)
        self.machine.add_transition('request_suspended_allow_new', running_states, ADCState.SUSPENDED_ALLOW_NEW)
        self.machine.add_transition('request_suspended_no_new', running_states, ADCState.SUSPENDED_NO_NEW)
