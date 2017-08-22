package deployment

import (
	"fmt"
	"strconv"
	"time"

	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/common"
)

func monitorDeployments() *axerror.AXError {

	params := map[string]interface{}{}
	params[DeploymentAppName] = utils.APPLICATION_NAME

	common.DebugLog.Printf("[HBM-%v] Deployment monitor starting\n", utils.APPLICATION_NAME)

	deployments, axErr := GetLatestDeployments(params, false)
	if axErr != nil {
		return axErr
	}

	common.DebugLog.Printf("[HBM-%v] Find %v deployments\n", utils.APPLICATION_NAME, len(deployments))

	for _, d := range deployments {
		monitorDeployment(d)
	}

	common.DebugLog.Printf("[HBM-%v] Deployment monitor finished\n", utils.APPLICATION_NAME)
	return nil
}

func monitorDeployment(d *Deployment) {

	DeployLockGroup.Lock(d.Key())
	defer DeployLockGroup.Unlock(d.Key())

	latest, axErr := GetLatestDeploymentByID(d.Id, false)
	if axErr != nil {
		common.ErrorLog.Printf("Error fetch deployment %v/%v: %v", d.ApplicationName, d.Name, axErr)
	}

	if latest == nil {
		return
	}

	d = latest

	common.DebugLog.Printf("[HBM-%v] Check deployment %v/%v starting\n", utils.APPLICATION_NAME, d.ApplicationName, d.Name)

	fresh := heartbeat.GetFreshness(d.HeartBeatKey())
	switch d.Status {
	case DeployStateInit:
		if axErr, _ := d.Create(); axErr != nil {
			common.ErrorLog.Printf("Failed to create deployment %v/%v in platform: %v.\n", d.ApplicationName, d.Name, axErr)
			if time.Now().Unix()-d.LaunchTime/1e6 > DEPLOYMENT_INIT_TIMEOUT_SEC {
				// If we fail to create the deployment after some time (30 min), auto-terminate the deployment.
				// NOTE: this only applies to newly created deployments, so auto-terminate is acceptable here.
				common.DebugLog.Printf("Timed out (%ds) attempting to create deployment %s. Terminating...\n", DEPLOYMENT_INIT_TIMEOUT_SEC, d)
				if axErr, _ := d.MarkTerminated(utils.GetStatusDetail(ErrCreatingDeployment, axErr.Message, axErr.Detail)); axErr != nil {
					common.ErrorLog.Printf("Failed terminate deployment %s: %v\n", d, axErr)
				}
			}
		}
	case DeployStateWaiting, DeployStateUpgrading:
		if status, axErr := getKubeDeploymentStatus(d); axErr == nil {
			d.Instances = status.GetPods(d.Id)
			if (status.AvailableReplicas == status.DesiredReplicas) && (len(d.Instances) >= d.Template.Scale.Min) {
				axErr, _ = d.MarkActive(utils.GetStatusDetail(InfoDeploymentActive, "Deployment is active.", ""))
				if axErr != nil {
					common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
				}
				utils.InfoLog.Println("[Notification]", d.ApplicationName, d.Name, d.Status, d.SendEventToNotificationCenter())
			} else {
				if time.Now().Unix()-d.LaunchTime/1e6 < 1800 {

					failures := status.GetFailures(d.Id)
					if time.Now().Unix()-d.LaunchTime/1e6 > 300 && len(failures) != 0 {
						failure := failures[0]
						if axErr, _ := d.MarkError(utils.GetStatusDetail(failure.Reason, failure.Message, "")); axErr != nil {
							common.ErrorLog.Printf("Failed to update deployment %v/%v in platform: %v.\n", d.ApplicationName, d.Name, axErr)
						}

					} else {

						if d.Status == DeployStateUpgrading {
							if axErr, _ = d.MarkUpgrading(nil); axErr != nil {
								common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
							}
						} else {
							if axErr, _ = d.MarkWaiting(nil); axErr != nil {
								common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
							}

						}
					}

				} else {
					axErr, _ = d.MarkError(utils.GetStatusDetail(ErrDeploymentTimeout, fmt.Sprintf("Timeout. %v out of %v instances are available.", len(d.Instances), d.Template.Scale.Min), ""))
					if axErr != nil {
						common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
					}
				}
			}
		}
	case DeployStateActive:
		if time.Now().Unix()-fresh > HEART_BEAT_GRACE_PERIOD {

			common.ErrorLog.Printf("[HB] Heartbeats for %v/%v is missing for %v seconds.\n", d.ApplicationName, d.Name, time.Now().Unix()-fresh)

			// heart beat missing, try to query for the status directly
			if status, axErr := getKubeDeploymentStatus(d); axErr != nil {
				axErr, _ = d.MarkError(utils.GetStatusDetail(axErr.Code, axErr.Message, axErr.Detail))
				if axErr != nil {
					common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
				}
				utils.InfoLog.Println("[Notification]", d.ApplicationName, d.Name, d.Status, d.SendEventToNotificationCenter())
			} else {
				d.Instances = status.GetPods(d.Id)
				if (status.AvailableReplicas == status.DesiredReplicas) && (len(d.Instances) >= d.Template.Scale.Min) {
				} else {
					axErr, _ = d.MarkError(utils.GetStatusDetail(ErrDeploymentDegraded, fmt.Sprintf("%v out of %v instances are available.", len(d.Instances), d.Template.Scale.Min), ""))
					if axErr != nil {
						common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
					}
					utils.InfoLog.Println("[Notification]", d.ApplicationName, d.Name, d.Status, d.SendEventToNotificationCenter())
				}
				UpdatePodCache(d, status)
			}

		}
	case DeployStateError:
		if time.Now().Unix()-fresh > HEART_BEAT_GRACE_PERIOD {

			common.ErrorLog.Printf("[HB] Heartbeats for %v/%v is missing for %v seconds.\n", d.ApplicationName, d.Name, time.Now().Unix()-fresh)

			if status, axErr := getKubeDeploymentStatus(d); axErr != nil {
				axErr, _ = d.MarkError(utils.GetStatusDetail(axErr.Code, axErr.Message, axErr.Detail))
				if axErr != nil {
					common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
				}
			} else {
				d.Instances = status.GetPods(d.Id)
				if (status.AvailableReplicas == status.DesiredReplicas) && (len(d.Instances) >= d.Template.Scale.Min) {
					axErr, _ = d.MarkActive(utils.GetStatusDetail(InfoDeploymentActive, "Deployment is active.", ""))
					if axErr != nil {
						common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
					}
				} else {
					axErr, _ = d.MarkError(utils.GetStatusDetail(ErrDeploymentDegraded, fmt.Sprintf("%v out of %v instances are available.", len(d.Instances), d.Template.Scale.Min), ""))
					if axErr != nil {
						common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
					}
				}
				UpdatePodCache(d, status)
			}
		}
	case DeployStateTerminating:
		if axErr, _ := d.Delete(nil); axErr != nil {
			common.ErrorLog.Printf("Failed to delete deployment %v/%v in platform: %v.\n", d.ApplicationName, d.Name, axErr)
		}
	case DeployStateStopping:
		if axErr, _ := d.Stop(); axErr != nil {
			common.ErrorLog.Printf("Failed to stop deployment %v/%v in platform: %v.\n", d.ApplicationName, d.Name, axErr)
		}
	}

	//switch d.Status {
	//case DeployStateInit, DeployStateTerminated, DeployStateStopped:
	//default:
	//	if axErr, _ := d.updateCost(); axErr != nil {
	//		common.ErrorLog.Printf("Failed to update deployment %v/%v cost: %v.\n", d.ApplicationName, d.Name, axErr)
	//	}
	//}

	if terminate, reason := ShouldTerminate(d, d.Cost, float64(d.RunTime/1e6)); terminate && d.Status != DeployStateTerminated && d.Status != DeployStateTerminating {
		if axErr, _ := d.MarkTerminating(utils.GetStatusDetail(reason, reason, "")); axErr != nil {
			common.ErrorLog.Printf("Error update deployment %v/%v status: %v", d.ApplicationName, d.Name, axErr)
		}
	}

	common.DebugLog.Printf("[HBM-%v] Check deployment %v/%v finished\n", utils.APPLICATION_NAME, d.ApplicationName, d.Name)
}

func ScheduleDeploymentMonitor() {
	ticker := time.NewTicker(time.Minute * 1)
	go func() {
		for _ = range ticker.C {
			monitorDeployments()
		}
	}()
}

func ScheduleSendingHeartbeatToAMM() {
	ticker := time.NewTicker(time.Minute * 1)
	go func() {
		for _ = range ticker.C {
			common.DebugLog.Printf("[HBM-%v] Sending heartbeat to AMM\n", utils.APPLICATION_NAME)
			axErr := SendHeartBeat(utils.APPLICATION_NAME)
			if axErr != nil {
				common.ErrorLog.Printf("[HBM-%v] Failed to send heartbeat to AMM due to: %v\n", utils.APPLICATION_NAME, axErr)
			}

		}
	}()
}

const (
	LimitSpendingExceed = "LimitSpendingExceed"
	LimitTimeExceed     = "LimitTimeExceed"
)

func ShouldTerminate(s *Deployment, cost float64, runTime float64) (bool, string) {
	if s == nil {
		return false, ""
	}

	if s.TerminationPolicy == nil {
		return false, ""
	}

	var spendingLimitCents, timeLimitSeconds float64
	var err error
	if s.TerminationPolicy.SpendingCents != "" {
		spendingLimitCents, err = strconv.ParseFloat(s.TerminationPolicy.SpendingCents, 64)
		if err != nil {
			utils.ErrorLog.Printf("[DeployMonitor] Error parsing the spending limit in cents (%v): %v.\n", s.TerminationPolicy.SpendingCents, err)
		}
	}

	if s.TerminationPolicy.TimeSeconds != "" {
		timeLimitSeconds, err = strconv.ParseFloat(s.TerminationPolicy.TimeSeconds, 64)
		if err != nil {
			utils.ErrorLog.Printf("[DeployMonitor] Error parsing the time limit in seconds (%v): %v.\n", s.TerminationPolicy.TimeSeconds, err)
		}
	}

	utils.DebugLog.Printf("[DeployMonitor] Limit(%v %v) Actual(%v, %v) Deployment(%v %v %v)", spendingLimitCents, timeLimitSeconds, cost, runTime, s.ApplicationName, s.Name, s.Id)
	if spendingLimitCents > 0 && cost > spendingLimitCents {
		return true, LimitSpendingExceed
	}

	if timeLimitSeconds > 0 && runTime > timeLimitSeconds {
		return true, LimitTimeExceed
	}

	return false, ""
}
