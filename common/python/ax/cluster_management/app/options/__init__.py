# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from .install_options import add_install_flags, ClusterInstallConfig, ClusterInstallDefaults
from .misc_operation_config import add_misc_flags, ClusterMiscOperationConfig
from .pause_options import add_pause_flags, ClusterPauseConfig
from .restart_options import add_restart_flags, ClusterRestartConfig
from .uninstall_options import add_uninstall_flags, ClusterUninstallConfig
from .upgrade_options import add_upgrade_flags, ClusterUpgradeConfig
from .common import ClusterOperationDefaults
