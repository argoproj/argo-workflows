package deployment

import (
	"applatix.io/axamm/adc"
	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/axops/service"
	"fmt"
	"time"
)

func (d *Deployment) MarkInit(detail map[string]interface{}) (*axerror.AXError, int) {
	return d.markObject(DeployStateInit, detail)
}

func (d *Deployment) MarkUpgrading(detail map[string]interface{}) (*axerror.AXError, int) {
	if d.LaunchTime == 0 {
		d.LaunchTime = int64(time.Now().UnixNano() / 1e3)
	}
	return d.markObject(DeployStateUpgrading, detail)
}

func (d *Deployment) MarkWaiting(detail map[string]interface{}) (*axerror.AXError, int) {
	if d.Template != nil && d.Template.Scale != nil {
		count := 0
		if len(d.Instances) != 0 {
			for _, pod := range d.Instances {
				if pod != nil {
					if pod.Ready() {
						count++
					}
				}
			}
		}

		if count < d.Template.Scale.Min {
			detail = utils.GetStatusDetail(ErrInstanceComingUp, fmt.Sprintf("%v out of %v instances are available.", count, d.Template.Scale.Min), "")
		} else {
			detail = utils.GetStatusDetail(ErrInstanceScalingDown, fmt.Sprintf("Instances are scaling down"), "")
		}
	}

	// No change, just return
	if d.Status == DeployStateWaiting && d.StatusDetail["message"].(string) == detail["message"].(string) && d.StatusDetail["code"].(string) == detail["code"].(string) {
		return nil, axerror.REST_STATUS_OK
	}

	if d.LaunchTime == 0 {
		d.LaunchTime = int64(time.Now().UnixNano() / 1e3)
	}

	return d.markObject(DeployStateWaiting, detail)
}

func (d *Deployment) MarkError(detail map[string]interface{}) (*axerror.AXError, int) {

	if d.LaunchTime == 0 {
		d.LaunchTime = int64(time.Now().UnixNano() / 1e3)
	}

	return d.markObject(DeployStateError, detail)
}

func (d *Deployment) MarkActive(detail map[string]interface{}) (*axerror.AXError, int) {

	if d.LaunchTime == 0 {
		d.LaunchTime = int64(time.Now().UnixNano() / 1e3)
	}

	// change ADC resources if moving to active from upgrading
	if d.Status == DeployStateUpgrading {
		if axErr, _ := d.reserveAdcResources(); axErr != nil {
			utils.InfoLog.Printf("Missed opportunity to change ADC resource after upgrade for %v/%v due to error:%v", d.ApplicationName, d.Name, axErr)
		}
	}

	return d.markObject(DeployStateActive, detail)
}

func (d *Deployment) MarkStopping(detail map[string]interface{}) (*axerror.AXError, int) {
	d.EndTime = int64(time.Now().UnixNano() / 1e3)
	if d.LaunchTime == 0 {
		d.RunTime = 0
	} else {
		d.RunTime = d.EndTime - d.LaunchTime - d.WaitTime
	}
	return d.markObject(DeployStateStopping, detail)
}

func (d *Deployment) MarkStopped(detail map[string]interface{}) (*axerror.AXError, int) {
	d.Instances = []*Pod{}
	return d.markObject(DeployStateStopped, detail)
}

func (d *Deployment) MarkTerminating(detail map[string]interface{}) (*axerror.AXError, int) {

	d.EndTime = int64(time.Now().UnixNano() / 1e3)
	if d.LaunchTime == 0 {
		d.RunTime = 0
	} else {
		d.RunTime = d.EndTime - d.LaunchTime - d.WaitTime
	}

	if axErr, code := d.markObject(DeployStateTerminating, detail); axErr != nil {
		return axErr, code
	}

	return nil, axerror.REST_STATUS_OK
}

func (d *Deployment) MarkTerminatedInHistory(detail map[string]interface{}) (*axerror.AXError, int) {

	heartbeat.UnregisterHandler(d.Key())

	d.EndTime = int64(time.Now().UnixNano() / 1e3)
	if d.LaunchTime == 0 {
		d.RunTime = 0
	} else {
		d.RunTime = d.EndTime - d.LaunchTime - d.WaitTime
	}

	d.Instances = []*Pod{}
	return d.markObjectInHistory(DeployStateTerminated, detail)
}

func (d *Deployment) MarkTerminated(detail map[string]interface{}) (*axerror.AXError, int) {

	heartbeat.UnregisterHandler(d.Key())

	// release fixtures
	if _, axErr, code := d.releaseFixtures(); axErr != nil {
		if axErr, code := d.MarkTerminating(utils.GetStatusDetail(ErrReleasingFixture, axErr.Message, axErr.Detail)); axErr != nil {
			return axErr, code
		}
		return axErr, code
	}

	// release resource
	if axErr, code := adc.Release(d.DeploymentID); axErr != nil {
		return axErr, code
	}

	d.Instances = []*Pod{}
	if axErr, code := d.markObject(DeployStateTerminated, detail); axErr != nil {
		return axErr, code
	}

	utils.InfoLog.Println("[Notification]", d.ApplicationName, d.Name, d.Status, d.SendEventToNotificationCenter())

	return d.CopyToHistory()
}

func (d *Deployment) markObjectInHistory(status string, detail map[string]interface{}) (*axerror.AXError, int) {

	d.Status = status

	if detail != nil {
		d.StatusDetail = detail
	} else {
		d.StatusDetail = map[string]interface{}{}
	}

	switch d.Status {
	case DeployStateStopped, DeployStateStopping, DeployStateTerminating, DeployStateTerminated:
		d.Cost = service.GetSpendingCents(d.CPU, d.Mem, float64(d.RunTime))
	}

	if _, axErr := d.UpdateHistoryObject(); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_STATUS_OK
}

func (d *Deployment) markObject(status string, detail map[string]interface{}) (*axerror.AXError, int) {

	d.Status = status

	if detail != nil {
		d.StatusDetail = detail
	} else {
		d.StatusDetail = map[string]interface{}{}
	}

	switch d.Status {
	case DeployStateStopped, DeployStateStopping, DeployStateTerminating, DeployStateTerminated:
		d.Cost = service.GetSpendingCents(d.CPU, d.Mem, float64(d.RunTime))
	}

	if _, axErr := d.updateObject(); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_STATUS_OK
}
