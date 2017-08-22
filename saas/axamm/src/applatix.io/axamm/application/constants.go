package application

import "applatix.io/common"

const (
	AppStateInit        = "Init"
	AppStateWaiting     = "Waiting"
	AppStateError       = "Error"
	AppStateActive      = "Active"
	AppStateTerminating = "Terminating"
	AppStateTerminated  = "Terminated"

	AppStateStopping  = "Stopping"
	AppStateStopped   = "Stopped"
	AppStateUpgrading = "Upgrading"
)

const (
	ErrMissingHeartBeat      = "ERR_MISSING_HEART_BEAT"
	ErrMissingNameSpace      = "ERR_MISSING_NAME_SPACE"
	ErrMissingMonitor        = "ERR_MISSING_MONITOR"
	ErrMissingRegistrySecret = "ERR_MISSING_REGISTRY_SECRET"
	ErrCreateNameSpace       = "ERR_CREATE_NAME_SPACE"
	ErrNameSpaceNotActive    = "ERR_NAME_SPACE_NOT_ACTIVE"
	ErrCreatingApp           = "ERR_CREATING_APPLICATION"
	ErrDeletingApp           = "ERR_DELETING_APPLICATION"
	ErrStoppingApp           = "ERR_STOPPING_APPLICATION"
)

const (
	appMonitorCpuCores = 0.1
	appMonitorMemMiB   = 100.0
)

func GetAppMonitorCpuCores() float64 {
	return appMonitorCpuCores * common.GetEnvFloat64(common.ENV_CPU_MULT, 1.0)
}

func GetAppMonitorMemMiB() float64 {
	return appMonitorMemMiB * common.GetEnvFloat64(common.ENV_MEM_MULT, 1.0)
}
