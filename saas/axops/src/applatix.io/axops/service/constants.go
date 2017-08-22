package service

import "applatix.io/axops/utils"

var StatusStringMap map[int]string = map[int]string{
	utils.ServiceStatusSuccess:    "Success",
	utils.ServiceStatusWaiting:    "Waiting",
	utils.ServiceStatusRunning:    "Running",
	utils.ServiceStatusFailed:     "Failed",
	utils.ServiceStatusCancelled:  "Cancelled",
	utils.ServiceStatusSkipped:    "Skipped",
	utils.ServiceStatusInitiating: "Init",
	utils.ServiceStatusCanceling:  "Canceling",
}
