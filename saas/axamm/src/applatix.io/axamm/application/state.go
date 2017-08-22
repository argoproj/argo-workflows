package application

import (
	"applatix.io/axamm/adc"
	"applatix.io/axamm/deployment"
	"applatix.io/axerror"
	"time"
)

func (a *Application) MarkObjectInit(status map[string]interface{}) (*axerror.AXError, int) {
	return a.markObject(AppStateInit, status)
}

func (a *Application) MarkObjectWaiting(status map[string]interface{}) (*axerror.AXError, int) {
	return a.markObject(AppStateWaiting, status)
}

func (a *Application) MarkObjectActive(status map[string]interface{}) (*axerror.AXError, int) {
	return a.markObject(AppStateActive, status)
}

func (a *Application) MarkObjectError(status map[string]interface{}) (*axerror.AXError, int) {
	return a.markObject(AppStateError, status)
}

func (a *Application) MarkObjectStopping(status map[string]interface{}) (*axerror.AXError, int) {
	return a.markObject(AppStateStopping, status)
}

func (a *Application) MarkObjectStopped(status map[string]interface{}) (*axerror.AXError, int) {
	return a.markObject(AppStateStopped, status)
}

func (a *Application) MarkObjectTerminating(status map[string]interface{}) (*axerror.AXError, int) {
	return a.markObject(AppStateTerminating, status)
}

func (a *Application) MarkObjectTerminated(status map[string]interface{}) (*axerror.AXError, int) {

	// release resource
	if axErr, code := adc.Release(a.ID); axErr != nil {
		return axErr, code
	}

	a.Mtime = int64(time.Now().UnixNano() / 1e3)
	a.Status = AppStateTerminated
	a.StatusDetail = status
	a.Endpoints = []string{}
	a.DeploymentsTerminated = int64(len(a.Deployments))
	a.DeploymentsInit = 0
	a.DeploymentsActive = 0
	a.DeploymentsStopped = 0
	a.DeploymentsStopping = 0
	a.DeploymentsTerminating = 0
	a.DeploymentsWaiting = 0
	a.DeploymentsError = 0

	if axErr := a.updateObject(ApplicationLatestTable); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_CREATE_OK
}

func (a *Application) markObject(status string, detail map[string]interface{}) (*axerror.AXError, int) {
	a.Mtime = int64(time.Now().UnixNano() / 1e3)

	a.Status = status

	if detail != nil {
		a.StatusDetail = detail
	} else {
		a.StatusDetail = map[string]interface{}{}
	}

	a.updateDeploymentCount()

	if axErr := a.updateObject(ApplicationLatestTable); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_CREATE_OK
}

func (a *Application) updateDeploymentCount() *axerror.AXError {
	var init, waiting, error, active, terminating, terminated, stopping, stopped int64

	if a.Deployments != nil {
		for _, d := range a.Deployments {
			switch d.Status {
			case deployment.DeployStateActive:
				active++
			case deployment.DeployStateTerminating:
				terminating++
			case deployment.DeployStateTerminated:
				terminated++
			case deployment.DeployStateError:
				error++
			case deployment.DeployStateInit:
				init++
			case deployment.DeployStateStopped:
				stopped++
			case deployment.DeployStateWaiting:
				waiting++
			case deployment.DeployStateStopping:
				stopping++
			}
		}

		a.DeploymentsInit = init
		a.DeploymentsWaiting = waiting
		a.DeploymentsActive = active
		a.DeploymentsError = error
		a.DeploymentsTerminating = terminating
		a.DeploymentsTerminated = terminated
		a.DeploymentsStopping = stopping
		a.DeploymentsStopped = stopped
	}

	return nil
}
