package application

import (
	"fmt"
	"time"

	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/common"
)

// Maximum in seconds in which an application can miss deployment heartbeats
// before the application is considered in an error state.
const HEART_BEAT_GRACE_PERIOD = 5 * 60

// IdleApplicationTerminationTimeout is the time in seconds before an idle application becomes terminated.
// An application is considered "idle" if it has zero deployments.
const IdleApplicationTerminationTimeout = 10 * 60

func GetHeartBeatHandler() heartbeat.HeartBeatHandler {
	return func(hb *heartbeat.HeartBeat) *axerror.AXError {

		//utils.DebugLog.Println(*hb)

		appName := hb.Key

		AppLockGroup.Lock(appName)
		defer AppLockGroup.Unlock(appName)

		app, axErr := GetLatestApplicationByName(appName, false)
		if axErr != nil {
			return axErr
		}

		if app == nil {
			common.InfoLog.Printf("HeartBeat Drop: Cannot find application %v.\n", appName)
			return nil
		}

		var total int64
		var changed bool

		if hb.Data != nil {
			deploymentsInit := int64(hb.Data["deployments_init"].(float64))
			if app.DeploymentsInit != deploymentsInit {
				common.InfoLog.Printf("%v deployments_init: %d -> %d", appName, app.DeploymentsInit, deploymentsInit)
				app.DeploymentsInit = deploymentsInit
				changed = true
			}
			deploymentsWaiting := int64(hb.Data["deployments_waiting"].(float64))
			if app.DeploymentsWaiting != deploymentsWaiting {
				common.InfoLog.Printf("%v deployments_waiting: %d -> %d", appName, app.DeploymentsWaiting, deploymentsWaiting)
				app.DeploymentsWaiting = deploymentsWaiting
				changed = true
			}
			deploymentsError := int64(hb.Data["deployments_error"].(float64))
			if app.DeploymentsError != deploymentsError {
				common.InfoLog.Printf("%v deployments_error: %d -> %d", appName, app.DeploymentsError, deploymentsError)
				app.DeploymentsError = deploymentsError
				changed = true
			}
			deploymentsActive := int64(hb.Data["deployments_active"].(float64))
			if app.DeploymentsActive != deploymentsActive {
				common.InfoLog.Printf("%v deployments_active: %d -> %d", appName, app.DeploymentsActive, deploymentsActive)
				app.DeploymentsActive = int64(hb.Data["deployments_active"].(float64))
				changed = true
			}
			deploymentsTerminating := int64(hb.Data["deployments_terminating"].(float64))
			if app.DeploymentsTerminating != deploymentsTerminating {
				common.InfoLog.Printf("%v deployments_terminating: %d -> %d", appName, app.DeploymentsTerminating, deploymentsTerminating)
				app.DeploymentsTerminating = deploymentsTerminating
				changed = true
			}
			deploymentsTerminated := int64(hb.Data["deployments_terminated"].(float64))
			if app.DeploymentsTerminated != deploymentsTerminated {
				common.InfoLog.Printf("%v deployments_terminated: %d -> %d", appName, app.DeploymentsTerminated, deploymentsTerminated)
				app.DeploymentsTerminated = deploymentsTerminated
				changed = true
			}
			deploymentsStopping := int64(hb.Data["deployments_stopping"].(float64))
			if app.DeploymentsStopping != deploymentsStopping {
				common.InfoLog.Printf("%v deployments_stopping: %d -> %d", appName, app.DeploymentsStopping, deploymentsStopping)
				app.DeploymentsStopping = deploymentsStopping
				changed = true
			}
			deploymentsStopped := int64(hb.Data["deployments_stopped"].(float64))
			if app.DeploymentsStopped != deploymentsStopped {
				common.InfoLog.Printf("%v deployments_stopped: %d -> %d", appName, app.DeploymentsStopped, deploymentsStopped)
				app.DeploymentsStopped = deploymentsStopped
				changed = true
			}
			deploymentsUpgrading := int64(hb.Data["deployments_upgrading"].(float64))
			if app.DeploymentsUpgrading != deploymentsUpgrading {
				common.InfoLog.Printf("%v deployments_upgrading: %d -> %d", appName, app.DeploymentsUpgrading, deploymentsUpgrading)
				app.DeploymentsUpgrading = deploymentsUpgrading
				changed = true
			}

			total = app.DeploymentsInit + app.DeploymentsWaiting + app.DeploymentsError + app.DeploymentsActive + app.DeploymentsTerminating + app.DeploymentsTerminated + app.DeploymentsStopping + app.DeploymentsStopped + app.DeploymentsUpgrading
		}

		newStatus := app.Status
		newDetail := app.StatusDetail

		if app.DeploymentsError != 0 {
			newStatus = AppStateError
			newDetail = utils.GetStatusDetail("ERROR", fmt.Sprintf("%v deployments are in error state.", app.DeploymentsError), "")
		} else if app.DeploymentsInit != 0 || app.DeploymentsWaiting != 0 {
			newStatus = AppStateWaiting
			newDetail = utils.GetStatusDetail("WAITING", fmt.Sprintf("%v deployments are coming up.", app.DeploymentsInit+app.DeploymentsWaiting), "")
		} else if app.DeploymentsUpgrading != 0 {
			newStatus = AppStateUpgrading
			newDetail = utils.GetStatusDetail("UPGRADING", fmt.Sprintf("%v deployments are upgrading.", app.DeploymentsUpgrading), "")
		} else if app.DeploymentsActive != 0 {
			newStatus = AppStateActive
			newDetail = utils.GetStatusDetail("ACTIVE", "", "")
		} else if app.DeploymentsStopped != 0 {
			newStatus = AppStateStopped
			newDetail = utils.GetStatusDetail("STOPPED", "Application is stopped.", "")
		} else {
			if !changed && total == app.DeploymentsTerminated {
				idleSeconds := time.Now().Unix() - app.Mtime/1e6
				utils.DebugLog.Printf("[HB] %v has been idle for %v seconds, it will be terminated in %v seconds.\n", app.Name, idleSeconds, IdleApplicationTerminationTimeout-idleSeconds)
				if idleSeconds > IdleApplicationTerminationTimeout {
					// Terminate the empty application after it is idled for 10 minutes
					newStatus = AppStateTerminating
					message := fmt.Sprintf("Application is idle for more than %d minutes.", IdleApplicationTerminationTimeout/60)
					newDetail = utils.GetStatusDetail("TERMINATING", message, "")
					utils.DebugLog.Printf("[HB] %v has been idle for %v seconds, terminating.\n", app.Name, idleSeconds)
				}
			}
		}

		if app.Status != newStatus {
			app.Status = newStatus
			app.StatusDetail = newDetail
			changed = true
		}

		if changed {

			if hb.Data != nil {
				endpoints := []string{}
				if hb.Data[ApplicationEndpoints] != nil {
					endPoints := hb.Data[ApplicationEndpoints].([]interface{})
					for _, endPoint := range endPoints {
						endpoints = append(endpoints, endPoint.(string))
					}
				}
				app.Endpoints = endpoints
			}

			_, axErr, _ = app.UpdateObject()
			if axErr != nil {
				return axErr
			}
		}

		return nil
	}
}
