# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

"""
Platform specific AX Exceptions
"""

from ax.exceptions import AXException
from ax.platform.ax_monitor_helper import KubeObjStatusCode


class AXPlatformException(AXException):
    code = "ERR_PLAT_INTERNAL"

class AXLoadImageException(AXPlatformException):
    code = "ERR_PLAT_LOAD_IMAGE"

class AXStuckInDeploymentException(AXPlatformException):
    code = "ERR_PLAT_STUCK_DEPLOYMENT"


class AXServiceCreationTimeoutException(AXPlatformException):
    code = "ERR_PLAT_SERVICE_CREATE_TIMEOUT"


class AXClusterActionException(AXPlatformException):
    code = "ERR_PLAT_CLUSTER_ACTION"

class AXStopMonitorException(AXPlatformException):
    code = "ERR_PLAT_STOP_MONITOR"

class AXStartMonitorException(AXPlatformException):
    code = "ERR_PLAT_START_MONITOR"

class AXStopDefaultException(AXPlatformException):
    code = "ERR_PLAT_STOP_DEFAULT"

class AXStartDefaultException(AXPlatformException):
    code = "ERR_PLAT_START_DEFAULT"

class AXUpgradeInProgress(AXPlatformException):
    code = "ERR_PLAT_UPGRADE_IN_PROGRESS"

class AXInsufficientResourceException(AXPlatformException):
    code = KubeObjStatusCode.ERR_INSUFFICIENT_RESOURCE

class AXTaskCreationTimeoutException(AXPlatformException):
    code = KubeObjStatusCode.ERR_PLAT_TASK_CREATE_TIMEOUT

class AXVolumeException(AXPlatformException):
    code = "ERR_PLAT_VOLUME_EXCEPTION"

class AXVolumeExistsException(AXVolumeException):
    code = "ERR_PLAT_VOLUME_EXISTS_EXCEPTION"

class AXVolumeOwnershipException(AXVolumeException):
    code = "ERR_PLAT_VOLUME_OWNERSHIP_EXCEPTION"
