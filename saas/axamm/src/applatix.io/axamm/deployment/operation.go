package deployment

import (
	"encoding/json"
	"fmt"
	"strings"

	"applatix.io/axamm/adc"
	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/common"
	"applatix.io/template"
)

func (d *Deployment) Create() (*axerror.AXError, int) {

	if axErr, code := d.checkStateChange(DeployStateWaiting); axErr != nil {
		return axErr, code
	}

	heartbeat.RegisterHandler(d.HeartBeatKey(), GetHeartBeatHandler())

	// reserve fixture via fixture manager
	d, axErr, code := d.reserveFixtures()
	if axErr != nil {
		if code >= 400 && code < 500 {
			if axErr, code := d.MarkTerminated(utils.GetStatusDetail(ErrReservingFixture, axErr.Message, axErr.Detail)); axErr != nil {
				return axErr, code
			}
		}
		return axErr, code
	}

	// create backend via axmon
	if axErr, code := d.createBackend(); axErr != nil {
		if code >= 400 && code < 500 {
			if axErr, code := d.MarkTerminated(utils.GetStatusDetail(ErrCreatingDeployment, axErr.Message, axErr.Detail)); axErr != nil {
				return axErr, code
			}
		}

		if strings.Contains(axErr.Message, "not enough") {
			if axErr, code := d.MarkTerminated(utils.GetStatusDetail(ErrNotEnoughResource, axErr.Message, axErr.Detail)); axErr != nil {
				axErr.Code = ErrNotEnoughResource
				return axErr, code
			}
		}

		return axErr, code
	}

	if axErr, code := d.MarkWaiting(nil); axErr != nil {
		return axErr, code
	}

	return nil, axerror.REST_STATUS_OK
}

func (d *Deployment) Upgrade(old *Deployment) (*Deployment, *axerror.AXError, int) {

	var prev *Deployment
	var axErr *axerror.AXError
	var code int

	if axErr, code := old.checkStateChange(DeployStateUpgrading); axErr != nil {
		common.ErrorLog.Printf("Invalid upgrade state transition for %v/%v.\n", d.ApplicationName, d.Name)
		return d, axErr, code
	}

	if d.Id != old.Id {
		common.InfoLog.Printf("upgrade to a new spec for %v/%v.\n", d.ApplicationName, d.Name)
		prev = old
		// acquire resources for upgrade
		d, axErr, code = d.ReserveResourcesForUpgrade(prev)
		if axErr != nil {
			return d, axErr, code
		}
		// copy to history
		if axErr, code = old.CopyToHistory(); axErr != nil {
			return d, axErr, code
		}

	} else {
		common.InfoLog.Printf("upgrade resubmit for %v/%v.\n", d.ApplicationName, d.Name)
		prev, axErr = GetHistoryDeploymentByID(old.PreviousDeploymentId, false)
		if axErr != nil {
			return d, axErr, axerror.REST_INTERNAL_ERR
		}
		if prev == nil {
			// this shouldn't happen
			common.ErrorLog.Printf("upgrade request for %v/%v with no previous deployment.\n", d.ApplicationName, d.Name)
			return d, axerror.ERR_API_INTERNAL_ERROR, axerror.REST_INTERNAL_ERR
		}
		// reacquire resources for upgrade
		d, axErr, code = d.ReserveResourcesForUpgrade(prev)
		if axErr != nil {
			return d, axErr, code
		}
	}

	d.PreviousDeploymentId = prev.Id
	if axErr, code := d.MarkUpgrading(nil); axErr != nil {
		return d, axErr, code
	}
	heartbeat.RegisterHandler(d.HeartBeatKey(), GetHeartBeatHandler())
	axErr, code = d.upgradeBackend()
	if axErr != nil {
		common.ErrorLog.Printf("upgrade request for %v/%v failed due to backend error:%v.\n", d.ApplicationName, d.Name, axErr)
		if axErr, _ := d.MarkError(utils.GetStatusDetail(ErrUpgradingDeployment, axErr.Message, axErr.Detail)); axErr != nil {
			common.ErrorLog.Printf("Failed to update status to upgrade error for %v/%v due to error:%v.\n", d.ApplicationName, d.Name, axErr)
		}
	}
	return d, axErr, code
}

func (d *Deployment) ReserveResourcesForUpgrade(old *Deployment) (*Deployment, *axerror.AXError, int) {

	// reserve fixture via fixture manager
	d, axErr, code := d.reserveFixtures()
	if axErr != nil {
		return d, axErr, code
	}

	axErr, code = d.reserveAdcResourcesForUpgrade(old)
	if axErr != nil {
		return d, axErr, code
	}

	return d, nil, axerror.REST_STATUS_OK
}

func (d *Deployment) Delete(detail map[string]interface{}) (*axerror.AXError, int) {
	utils.DebugLog.Printf("Deleting %s", d)

	if axErr, code := d.checkStateChange(DeployStateTerminated); axErr != nil {
		return axErr, code
	}

	if d.Status == DeployStateTerminated {
		return nil, axerror.REST_STATUS_OK
	}

	if d.Status != DeployStateTerminating {
		if detail == nil {
			if axErr, code := d.MarkTerminating(utils.GetStatusDetail("TERMINATING", "Deployment will be terminated shortly.", "")); axErr != nil {
				return axErr, code
			}
		} else {
			if axErr, code := d.MarkTerminating(detail); axErr != nil {
				return axErr, code
			}
		}
	}

	heartbeat.UnregisterHandler(d.HeartBeatKey())

	// delete backend via axmon
	if axErr, code := d.deleteBackend(); axErr != nil {

		if axErr, code := d.MarkTerminating(utils.GetStatusDetail(ErrDeletingDeployment, axErr.Message, axErr.Detail)); axErr != nil {
			return axErr, code
		}

		return axErr, code
	}

	DeletePodCache(d.Id)

	if detail == nil {
		detail = utils.GetStatusDetail("TERMINATED", "Deployment is terminated.", "")
	}

	return d.MarkTerminated(detail)
}

func (d *Deployment) Start() (*axerror.AXError, int) {
	utils.DebugLog.Printf("Starting %s", d)

	if axErr, code := d.checkStateChange(DeployStateWaiting); axErr != nil {
		return axErr, code
	}

	heartbeat.RegisterHandler(d.HeartBeatKey(), GetHeartBeatHandler())

	if axErr, code := d.startBackend(); axErr != nil {
		if axErr, code := d.MarkStopped(utils.GetStatusDetail(ErrScalingDeployment, axErr.Message, axErr.Detail)); axErr != nil {
			return axErr, code
		}
		return axErr, code
	}

	if axErr, code := d.MarkWaiting(nil); axErr != nil {
		return axErr, code
	}

	return nil, axerror.REST_STATUS_OK
}

func (d *Deployment) Stop() (*axerror.AXError, int) {
	utils.DebugLog.Printf("Stopping %s", d)

	if axErr, code := d.checkStateChange(DeployStateStopped); axErr != nil {
		return axErr, code
	}

	if d.Status == DeployStateStopped {
		return nil, axerror.REST_STATUS_OK
	}

	if d.Status != DeployStateStopping {
		if axErr, code := d.MarkStopping(utils.GetStatusDetail("STOPPING", "Deployment will be stopped shortly.", "")); axErr != nil {
			return axErr, code
		}
	}

	// delete backend via axmon
	if axErr, code := d.stopBackend(); axErr != nil {
		if axErr, code := d.MarkStopping(utils.GetStatusDetail(ErrScalingDeployment, axErr.Message, axErr.Detail)); axErr != nil {
			return axErr, code
		}
		return axErr, code
	}

	if axErr, code := d.MarkStopped(utils.GetStatusDetail("STOPPED", "Deployment is stopped.", "")); axErr != nil {
		return axErr, code
	}

	DeletePodCache(d.Id)

	return nil, axerror.REST_STATUS_OK
}

func (d *Deployment) Scale(scale *template.Scale) (*axerror.AXError, int) {
	utils.DebugLog.Printf("Scaling %s: min: %d, max: %d", d, scale.Min, scale.Max)

	if len(d.Template.Volumes) != 0 {
		if scale.Min > 1 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Scale deployment with volumes is not supported temporarily."), axerror.REST_FORBIDDEN
		}
	}

	switch d.Status {
	case DeployStateWaiting, DeployStateActive, DeployStateError:
	default:
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Can not scale the deployment with state: %v.", d.Status), axerror.REST_FORBIDDEN
	}

	if axErr, code := d.checkStateChange(DeployStateWaiting); axErr != nil {
		return axErr, code
	}

	// delete backend via axmon
	if axErr, code := d.scaleBackend(scale.Min); axErr != nil {
		if axErr, code := d.MarkError(utils.GetStatusDetail(ErrScalingDeployment, axErr.Message, axErr.Detail)); axErr != nil {
			return axErr, code
		}
		return axErr, code
	}

	d.Template.Scale = scale
	d.getMaxResources()

	if axErr, code := d.MarkWaiting(nil); axErr != nil {
		return axErr, code
	}

	return nil, axerror.REST_STATUS_OK
}

func (d *Deployment) reserveFixtures() (*Deployment, *axerror.AXError, int) {

	code := axerror.REST_STATUS_OK

	f := &Fixture{}
	f.ServiceId = d.DeploymentID
	f.RootWorkflowId = d.TaskID
	f.Requester = "axamm"
	f.User = d.User
	f.ApplicationName = d.ApplicationName
	f.AppID = d.ApplicationID
	f.AppGeneration = d.ApplicationGeneration
	f.DeploymentName = d.Name
	f.DeploymentID = d.DeploymentID
	f.DeploymentGeneration = d.Id

	// Normal fixtures
	requirements := map[string]*template.FixtureRequirement{}
	for _, parallelFixtures := range d.Template.Fixtures {
		for name, fixReq := range parallelFixtures {
			if fixReq.IsDynamicFixture() {
				return nil, axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("Deployments cannot request dynamic fixtures (%s)", name), axerror.REST_BAD_REQ
			}
			requirements[name] = &fixReq.FixtureRequirement
		}
	}
	f.Requirements = requirements

	// Prepare the volume request
	f.VolRequirements = d.Template.Volumes
	if len(f.VolRequirements) > 0 {
		for _, vol := range f.VolRequirements {
			if vol.Name != "" {
				vol.AXRN = fmt.Sprintf("vol:/%v", vol.Name)
			}
		}
	}

	// reserve the resource in sync way
	f.Synchronous = utils.NewTrue()

	if len(f.Requirements) == 0 && len(f.VolRequirements) == 0 {
		return d, nil, code
	}
	fBytes, _ := json.Marshal(f)
	utils.DebugLog.Printf("Reserving fixtures for %s: %s", d, string(fBytes))

	f, axErr, code := f.reserve()
	if axErr != nil {
		return d, axErr, code
	}

	if f.Assignment != nil {
		d.Fixtures = f.Assignment
		d, axErr = d.Substitute()
		if axErr != nil {
			return d, axErr, axerror.REST_INTERNAL_ERR
		}
	}

	if f.VolAssignment != nil {
		for refName, vol := range d.Template.Volumes {
			if _, ok := f.VolAssignment[refName]; ok {
				vol.Details = f.VolAssignment[refName]
			}
		}
		// The following will set the .Details field for inputs.volumes.VOLNAME in the child containers
		for refName, details := range f.VolAssignment {
			varName := fmt.Sprintf("%%%%volumes.%s%%%%", refName)
			for _, svc := range d.Template.Containers {
				for argName, argValue := range svc.Arguments {
					if argValue != nil && *argValue == varName {
						inputs := svc.Template.GetInputs()
						if inputs != nil && inputs.Volumes != nil {
							parts := strings.Split(argName, ".")
							volName := parts[len(parts)-1]
							if inputVol, ok := inputs.Volumes[volName]; ok {
								utils.DebugLog.Printf("Setting details in child: %s", details)
								inputVol.Details = details
							}
						}

					}
				}
			}
		}
	}

	return d, nil, axerror.REST_STATUS_OK
}

func (d *Deployment) releaseFixtures() (*Deployment, *axerror.AXError, int) {

	code := axerror.REST_STATUS_OK

	if len(d.Template.Fixtures) == 0 && len(d.Template.Volumes) == 0 {
		return d, nil, code
	}
	utils.DebugLog.Printf("Releasing fixtures for %s", d)

	f := &Fixture{}
	f.ServiceId = d.DeploymentID
	f.RootWorkflowId = d.TaskID
	f.Requester = "axamm"
	f.User = d.User
	f.ApplicationName = d.ApplicationName
	f.AppID = d.ApplicationID
	f.AppGeneration = d.ApplicationGeneration
	f.DeploymentName = d.Name
	f.DeploymentID = d.DeploymentID
	f.DeploymentGeneration = d.Id

	// release the resource in sync way
	f.Synchronous = utils.NewTrue()

	axErr, code := f.release()
	return d, axErr, code
}

func (d *Deployment) reserveAdcResourcesForUpgrade(old *Deployment) (*axerror.AXError, int) {
	if TEST_ENABLED {
		return nil, 200
	}

	detail := map[string]string{
		"name": d.Name,
		"app":  d.ApplicationName,
	}

	cpu, mem := old.GetMaxResourcesForUpgrade(d)
	return adc.Reserve(d.DeploymentID, "deployment", cpu, mem, adc.AdcDefaultTtl, detail)
}

func (d *Deployment) reserveAdcResources() (*axerror.AXError, int) {
	if TEST_ENABLED {
		return nil, 200
	}

	detail := map[string]string{
		"name": d.Name,
		"app":  d.ApplicationName,
	}

	cpu, mem := d.getMaxResources()
	return adc.Reserve(d.DeploymentID, "deployment", cpu, mem, adc.AdcDefaultTtl, detail)
}

func (d *Deployment) releaseAdcResources() (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	return adc.Release(d.DeploymentID)
}

func (d *Deployment) createBackend() (*axerror.AXError, int) {
	return createKubeDeployment(d)
}

func (d *Deployment) upgradeBackend() (*axerror.AXError, int) {
	return upgradeKubeDeployment(d)
}

func (d *Deployment) deleteBackend() (*axerror.AXError, int) {
	return deleteKubeDeployment(d)
}

func (d *Deployment) stopBackend() (*axerror.AXError, int) {
	return scaleKubeDeployment(d, 0)
}

func (d *Deployment) startBackend() (*axerror.AXError, int) {
	return scaleKubeDeployment(d, d.Template.Scale.Min)
}

func (d *Deployment) scaleBackend(replicas int) (*axerror.AXError, int) {
	return scaleKubeDeployment(d, replicas)
}
