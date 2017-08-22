package deployment

import (
	"time"

	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/common"
	"encoding/json"
	"fmt"
)

const HEART_BEAT_GRACE_PERIOD = 3 * 60

func GetHeartBeatHandler() heartbeat.HeartBeatHandler {
	return func(hb *heartbeat.HeartBeat) *axerror.AXError {

		id := hb.Key

		DeployLockGroup.Lock(id)
		defer DeployLockGroup.Unlock(id)

		d, axErr := GetLatestDeploymentByID(id, false)
		if axErr != nil {
			return axErr
		}

		if d == nil {
			// check and update deployment in history
			d, axErr = GetHistoryDeploymentByID(id, false)
			if axErr != nil {
				return axErr
			}

			if d == nil {
				common.InfoLog.Printf("[HB] HeartBeat Drop: Cannot find deployment %v.\n", id)
				return nil
			}

			if d.ApplicationName != utils.APPLICATION_NAME {
				common.InfoLog.Printf("[HB] HeartBeat Drop: Cannot handle %v/%v in the wrong application %v.\n", d.ApplicationName, d.Name, utils.APPLICATION_NAME)
				return nil
			}
			var phb *PodHeartBeat
			err := json.Unmarshal(hb.Origin, &phb)
			if err != nil {
				common.InfoLog.Printf("[HB] HeartBeat Drop: %v.\n", err.Error())
				return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
			}

			if phb.Data == nil || phb.Data.PodStatus == nil {
				common.InfoLog.Printf("[HB] HeartBeat Drop: Missing pod status in the heartbeat.\n")
				return nil
			}

			common.DebugLog.Printf("[HB] HeartBeat Content: %v \n", string(hb.Origin))

			status, _ := UpdatePodCacheDelta(d, phb)

			if status.AvailableReplicas == 0 {
				d.MarkTerminatedInHistory(nil)
			} else {
				// update pod instances
				d.Instances = status.Pods
				d.UpdateHistoryObject()
			}
			return nil
		}

		if d.ApplicationName != utils.APPLICATION_NAME {
			common.InfoLog.Printf("[HB] HeartBeat Drop: Cannot handle %v/%v in the wrong application %v.\n", d.ApplicationName, d.Name, utils.APPLICATION_NAME)
			return nil
		}

		switch d.Status {
		case DeployStateTerminated, DeployStateTerminating, DeployStateStopping, DeployStateStopped:
			return nil
		}

		var phb *PodHeartBeat
		err := json.Unmarshal(hb.Origin, &phb)
		if err != nil {
			common.InfoLog.Printf("[HB] HeartBeat Drop: %v.\n", err.Error())
			return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
		}

		if phb.Data == nil || phb.Data.PodStatus == nil {
			common.InfoLog.Printf("[HB] HeartBeat Drop: Missing pod status in the heartbeat.\n")
			return nil
		}

		common.DebugLog.Printf("[HB] HeartBeat Content: %v \n", string(hb.Origin))

		status, hbType := UpdatePodCacheDelta(d, phb)

		d.Instances = status.Pods

		if (status.AvailableReplicas == status.DesiredReplicas) && (len(d.Instances) >= d.Template.Scale.Min) {
			if axErr, _ := d.MarkActive(utils.GetStatusDetail(InfoDeploymentActive, "", "")); axErr != nil {
				return axErr
			}
		} else if d.Status == DeployStateWaiting || d.Status == DeployStateUpgrading {
			switch hbType {
			case TypeArtifactLoadFailed:
				if axErr, _ := d.MarkError(utils.GetStatusDetail(TypeArtifactLoadFailed, "", "")); axErr != nil {
					return axErr
				}
			default:
				if d.Status == DeployStateUpgrading {
					if axErr, _ := d.MarkUpgrading(nil); axErr != nil {
						return axErr
					}
				} else {
					if axErr, _ := d.MarkWaiting(nil); axErr != nil {
						return axErr
					}
				}
			}
		} else {
			if axErr, _ := d.MarkError(utils.GetStatusDetail(ErrDeploymentDegraded, fmt.Sprintf("%v out of %v instances are available.", status.AvailableReplicas, d.Template.Scale.Min), "")); axErr != nil {
				return axErr
			}
		}

		return nil
	}
}

func SendHeartBeat(application string) *axerror.AXError {
	deployments, axErr := GetLatestDeploymentsByApplication(application, true)
	if axErr != nil {
		return axErr
	}

	var init, waiting, error, active, terminating, terminated, stopping, stopped, upgrading int
	endpoints := []string{}

	for _, d := range deployments {
		switch d.Status {
		case DeployStateActive:
			active++
		case DeployStateTerminating:
			terminating++
		case DeployStateTerminated:
			terminated++
		case DeployStateError:
			error++
		case DeployStateInit:
			init++
		case DeployStateStopped:
			stopped++
		case DeployStateWaiting:
			waiting++
		case DeployStateStopping:
			stopping++
		case DeployStateUpgrading:
			upgrading++
		}

		if d.Status != DeployStateTerminated && d.Status != DeployStateStopped && len(d.Endpoints) != 0 {
			endpoints = append(endpoints, d.Endpoints...)
		}
	}

	hb := &heartbeat.HeartBeat{
		Date: time.Now().Unix(),
		Key:  utils.APPLICATION_NAME,
		Data: map[string]interface{}{
			"deployments":             deployments,
			"deployments_init":        init,
			"deployments_waiting":     waiting,
			"deployments_error":       error,
			"deployments_active":      active,
			"deployments_terminating": terminating,
			"deployments_terminated":  terminated,
			"deployments_stopping":    stopping,
			"deployments_stopped":     stopped,
			"deployments_upgrading":   upgrading,
			"endpoints":               endpoints,
		},
	}

	_, axErr = utils.AmmCl.Post("heartbeats", hb)

	return axErr
}
