# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import pytest
from transitions import MachineError

from ax.cluster_management.app.state import ClusterState
from .mock import ClusterStateMachineMock


def test_cluster_state_normal_transition():
    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)

    # Install
    csm.do_install()
    assert csm.is_installing()
    csm.done_install()
    assert csm.is_running()

    # Pause
    csm.do_pause()
    assert csm.is_pausing()
    csm.done_pause()
    assert csm.is_paused()

    # Resume
    csm.do_resume()
    assert csm.is_resuming()
    csm.done_resume()
    assert csm.is_running()

    # Upgrade
    csm.do_upgrade()
    assert csm.is_upgrading()
    csm.done_upgrade()
    assert csm.is_running()

    # Uninstall
    csm.do_uninstall()
    assert csm.is_uninstalling()


def test_valid_cluster_state_transition_from_unknown():
    # We can transit from Unknown to Installing, Pausing, Upgrading, Resuming, Uninstalling
    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)
    csm.do_install()
    assert csm.is_installing()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)
    csm.do_pause()
    assert csm.is_pausing()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)
    csm.do_resume()
    assert csm.is_resuming()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)
    csm.do_upgrade()
    assert csm.is_upgrading()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)
    csm.do_uninstall()
    assert csm.is_uninstalling()


def test_invalid_cluster_state_transition_from_unknown():
    # With Unknown state, we cannot transit it to RUNNING / PAUSED
    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)
    with pytest.raises(MachineError):
        csm.done_install()
    assert csm.is_unknown()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNKNOWN)
    with pytest.raises(MachineError):
        csm.done_pause()
    assert csm.is_unknown()


def test_uninstall_from_any_state():
    csm = ClusterStateMachineMock(current_state=ClusterState.INSTALLING)
    csm.do_uninstall()
    assert csm.is_uninstalling()

    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSING)
    csm.do_uninstall()
    assert csm.is_uninstalling()

    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSED)
    csm.do_uninstall()
    assert csm.is_uninstalling()

    csm = ClusterStateMachineMock(current_state=ClusterState.RESUMING)
    csm.do_uninstall()
    assert csm.is_uninstalling()


def test_transit_to_same_state():
    # This is to make sure we can retry operations
    csm = ClusterStateMachineMock(current_state=ClusterState.INSTALLING)
    csm.do_install()
    assert csm.is_installing()

    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSING)
    csm.do_pause()
    assert csm.is_pausing()

    csm = ClusterStateMachineMock(current_state=ClusterState.RESUMING)
    csm.do_resume()
    assert csm.is_resuming()

    csm = ClusterStateMachineMock(current_state=ClusterState.UPGRADING)
    csm.do_upgrade()
    assert csm.is_upgrading()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNINSTALLING)
    csm.do_uninstall()
    assert csm.is_uninstalling()


def test_other_common_invalid_state_transitions():
    # Invalid transitions from Installing
    csm = ClusterStateMachineMock(current_state=ClusterState.INSTALLING)
    with pytest.raises(MachineError):
        csm.do_pause()
    assert csm.is_installing()

    csm = ClusterStateMachineMock(current_state=ClusterState.INSTALLING)
    with pytest.raises(MachineError):
        csm.do_upgrade()
    assert csm.is_installing()

    csm = ClusterStateMachineMock(current_state=ClusterState.INSTALLING)
    with pytest.raises(MachineError):
        csm.do_resume()
    assert csm.is_installing()

    # Invalid transitions from Pausing
    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSING)
    with pytest.raises(MachineError):
        csm.do_install()
    assert csm.is_pausing()

    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSING)
    with pytest.raises(MachineError):
        csm.do_resume()
    assert csm.is_pausing()

    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSING)
    with pytest.raises(MachineError):
        csm.do_upgrade()
    assert csm.is_pausing()

    # Invalid transitions from Resuming
    csm = ClusterStateMachineMock(current_state=ClusterState.RESUMING)
    with pytest.raises(MachineError):
        csm.do_install()
    assert csm.is_resuming()

    csm = ClusterStateMachineMock(current_state=ClusterState.RESUMING)
    with pytest.raises(MachineError):
        csm.do_upgrade()
    assert csm.is_resuming()

    csm = ClusterStateMachineMock(current_state=ClusterState.RESUMING)
    with pytest.raises(MachineError):
        csm.do_pause()
    assert csm.is_resuming()

    # Invalid transitions from Upgrading
    csm = ClusterStateMachineMock(current_state=ClusterState.UPGRADING)
    with pytest.raises(MachineError):
        csm.do_install()
    assert csm.is_upgrading()

    csm = ClusterStateMachineMock(current_state=ClusterState.UPGRADING)
    with pytest.raises(MachineError):
        csm.do_pause()
    assert csm.is_upgrading()

    csm = ClusterStateMachineMock(current_state=ClusterState.UPGRADING)
    with pytest.raises(MachineError):
        csm.do_resume()
    assert csm.is_upgrading()

    # Invalid transitions from Paused
    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSED)
    with pytest.raises(MachineError):
        csm.do_install()
    assert csm.is_paused()

    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSED)
    with pytest.raises(MachineError):
        csm.do_pause()
    assert csm.is_paused()

    csm = ClusterStateMachineMock(current_state=ClusterState.PAUSED)
    with pytest.raises(MachineError):
        csm.do_upgrade()
    assert csm.is_paused()

    # Invalid transitions from Uninstalling
    csm = ClusterStateMachineMock(current_state=ClusterState.UNINSTALLING)
    with pytest.raises(MachineError):
        csm.do_install()
    assert csm.is_uninstalling()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNINSTALLING)
    with pytest.raises(MachineError):
        csm.do_pause()
    assert csm.is_uninstalling()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNINSTALLING)
    with pytest.raises(MachineError):
        csm.do_upgrade()
    assert csm.is_uninstalling()

    csm = ClusterStateMachineMock(current_state=ClusterState.UNINSTALLING)
    with pytest.raises(MachineError):
        csm.do_resume()
    assert csm.is_uninstalling()
