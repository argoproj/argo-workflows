package application

import (
	"applatix.io/axamm/axam"
	"applatix.io/axamm/deployment"
	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
)

// Create will create the Application object in the database and create the platform backend if necessary
// A lock against the application name is assumed.
func (a *Application) Create() (*Application, *axerror.AXError, int) {
	utils.InfoLog.Printf("Creating Application %s", a.Name)
	old, axErr := GetLatestApplicationByName(a.Name, false)
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	if old != nil {
		// Handle case where existing application exists but is terminating/terminated. In this case,
		// we wish to delete the exiting entry from the latest table so that we can recreate it.
		if old.Status == AppStateTerminating {
			utils.InfoLog.Printf("Deleting existing backend for %s %s", old.Status, old)
			deleted, axErr, code := old.deleteBackend()
			if axErr != nil {
				return nil, axErr, code
			}
			old = deleted
		}
		if old.Status == AppStateTerminated {
			// Delete the old application during the same name to make room for the new coming.
			// The old application has a copy in history table the time it was terminated.
			utils.InfoLog.Printf("Deleting existing object for %s %s", old.Status, old)
			if axErr, code := old.DeleteObject(); axErr != nil {
				return nil, axErr, code
			}
			old = nil
		}
	}

	var app *Application
	var terminateOnError bool
	if old == nil {
		// Application doesn't exist yet (or we just deleted it from above). (Re)create the application database object
		new, axErr, code := a.CreateObject()
		if axErr != nil {
			return nil, axErr, code
		}
		app = new
		terminateOnError = true
	} else {
		// Existing application already exists. Reuse the app.
		// NOTE: it might be in init, active, waiting, or even error state
		utils.InfoLog.Printf("Reusing existing %s (status: %s)", old, old.Status)
		app = old
		terminateOnError = false
	}

	heartbeat.RegisterHandler(app.Key(), GetHeartBeatHandler())

	if app.Reachable() {
		// Bypass platform creation if we can successfully ping AXAM
		utils.InfoLog.Printf("%s reachable (status: %s). Bypassing backend creation", app, app.Status)
		return app, nil, axerror.REST_CREATE_OK
	}

	// Optimize the case where we launch multiple concurrent deployments at the same time for a new app.
	// In this case, the backend namespace/axam is likely just starting to come up and we failed
	// the reachable check. Still decide to skip backend creation if we detect the application age
	// is relatively young (3 min).
	if app.Status == AppStateWaiting && app.AgeSeconds() < 3*60 {
		utils.InfoLog.Printf("%s (status: %s) concurrent creations detected. Bypassing backend creation", app, app.Status)
		return app, nil, axerror.REST_CREATE_OK
	}

	return app.createBackend(terminateOnError)
}

func (a *Application) Delete() (*Application, *axerror.AXError, int) {
	utils.InfoLog.Printf("Deleting %s", a)
	if a.Status == AppStateTerminated {
		return a, nil, axerror.REST_STATUS_OK
	}

	if axErr, code := a.MarkObjectTerminating(utils.GetStatusDetail("TERMINATING", "Application will be terminated shortly.", "")); axErr != nil {
		return nil, axErr, code
	}

	heartbeat.UnregisterHandler(a.Key())
	return a.deleteBackend()
}

func (a *Application) Stop() (*Application, *axerror.AXError, int) {
	utils.InfoLog.Printf("Stopping %s", a)
	if a.Status == AppStateStopped {
		return a, nil, axerror.REST_STATUS_OK
	}

	switch a.Status {
	case AppStateActive, AppStateError, AppStateStopping:
	default:
		return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("The application with status %v can not be stopped.", a.Status), axerror.REST_BAD_REQ
	}

	if a.Status != AppStateStopping {
		if axErr, code := a.MarkObjectStopping(utils.GetStatusDetail("STOPPPING", "Application will be stopped shortly.", "")); axErr != nil {
			return nil, axErr, code
		}
	}

	return a.stopBackend()
}

func (a *Application) Start() (*Application, *axerror.AXError, int) {
	utils.InfoLog.Printf("Starting %s", a)
	switch a.Status {
	case AppStateStopped:
	// flow through
	case AppStateInit, AppStateWaiting, AppStateActive, AppStateError:
		return a, nil, axerror.REST_STATUS_OK
	default:
		return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("The application with status %v can not be started.", a.Status), axerror.REST_BAD_REQ
	}

	if a.Status != AppStateWaiting {
		if axErr, code := a.MarkObjectWaiting(utils.GetStatusDetail("WAITING", "Application is coming up.", "")); axErr != nil {
			return nil, axErr, code
		}
	}

	heartbeat.RegisterHandler(a.Key(), GetHeartBeatHandler())
	return a.startBackend()
}

// createBackend makes the call to both ADC and platform to reserve the resources in ADC and create the platform namespace and axam deployment
// if terminateOnError is true, will terminate the application if platform or adc returned error
func (a *Application) createBackend(terminateOnError bool) (*Application, *axerror.AXError, int) {
	// Create the application in platform
	utils.InfoLog.Printf("Creating backend for %s", a)
	if axErr, code := CreateApp(a.Name, a.ID); axErr != nil {
		utils.InfoLog.Printf("Backend creation for %s failed: %v", a, axErr)
		detail := map[string]interface{}{
			"code":    axErr.Code,
			"detail":  axErr.Detail,
			"message": axErr.Message,
		}

		if terminateOnError {
			// Clean up desired, keep the same status detail for terminated state
			ignoreErr, _ := a.MarkObjectTerminated(detail)
			if ignoreErr != nil {
				utils.ErrorLog.Printf("Delete application object failed:%v\n", ignoreErr)
			}
			heartbeat.UnregisterHandler(a.Key())
		} else {
			// preserve the current status but add error detail (best effort - we want to preserve adc/platform error)
			a.markObject(a.Status, detail)
		}

		return nil, axErr, code
	}

	if axErr, code := a.MarkObjectWaiting(utils.GetStatusDetail("WAITING", "Application is coming up.", "")); axErr != nil {
		return nil, axErr, code
	}

	return a, nil, axerror.REST_CREATE_OK
}

func (a *Application) deleteBackend() (*Application, *axerror.AXError, int) {
	utils.InfoLog.Printf("Deleting backend of %s", a)
	// Terminate all its deployments
	deployments, axErr := deployment.GetLatestDeploymentsByApplication(a.Name, false)
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	summary, axErr := GetSystemAppStatus(a.Name, a.ID)
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	if summary.Result.Namespace == true {
		axamReachable := a.Reachable()
		if !axamReachable {
			utils.ErrorLog.Printf("AXAM for %s is not reachable", a)
		}
		for i := range deployments {
			if deployments[i].Status != deployment.DeployStateTerminated {
				if axamReachable {
					// Ask application monitor to terminate the deployment
					if axErr, code := axam.DeleteAmDeployment(deployments[i]); axErr != nil {
						return nil, axErr, code
					}
				} else {
					// if axam is not reachable, we need to do it ourselves
					if axErr, code := deployments[i].Delete(nil); axErr != nil {
						return nil, axErr, code
					}
				}

				// Get the latest copy of it
				d, axErr := deployment.GetLatestDeploymentByID(deployments[i].Id, false)
				if axErr != nil {
					return nil, axErr, axerror.REST_INTERNAL_ERR
				}
				if d != nil {
					deployments[i] = d
				}

				if axErr := d.DeleteObject(); axErr != nil {
					return nil, axErr, axerror.REST_INTERNAL_ERR
				}
			} else {
				if axErr := deployments[i].DeleteObject(); axErr != nil {
					return nil, axErr, axerror.REST_INTERNAL_ERR
				}
			}
		}

		// Delete the application in platform
		if axErr, code := DeleteApp(a.Name, a.ID); axErr != nil {
			return nil, axErr, code
		}
	} else {
		// In rare case, the namespace has been deleted, all the deployments must be gone, so just need
		// mark all of them to be terminated.
		for i := range deployments {
			if deployments[i].Status != deployment.DeployStateTerminated {
				if axErr, code := deployments[i].MarkTerminated(utils.GetStatusDetail("TERMINATED", "Deployment is terminated.", "")); axErr != nil {
					return nil, axErr, code
				}
			}

			if axErr := deployments[i].DeleteObject(); axErr != nil {
				return nil, axErr, axerror.REST_INTERNAL_ERR
			}
		}
	}

	a.Deployments = deployments
	// Make the application object
	axErr, code := a.MarkObjectTerminated(utils.GetStatusDetail("TERMINATED", "Application is terminated.", ""))
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	// Keep the terminated application in the history
	axErr, code = a.CopyToHistory()
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	return a, nil, code
}

func (a *Application) stopBackend() (*Application, *axerror.AXError, int) {
	utils.InfoLog.Printf("Stopping backend of %s", a)
	// Stop all its deployments
	deployments, axErr := deployment.GetLatestDeploymentsByApplication(a.Name, false)
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	for i, d := range deployments {
		switch d.Status {
		case deployment.DeployStateInit:
		case deployment.DeployStateStopped:
		case deployment.DeployStateTerminating:
		case deployment.DeployStateTerminated:
		// Do nothing
		default:
			if axErr, code := axam.StopAmDeployment(d); axErr != nil {
				return nil, axErr, code
			}

			if d, axErr := deployment.GetLatestDeploymentByID(d.Id, false); axErr != nil {
				return nil, axErr, axerror.REST_INTERNAL_ERR
			} else {
				if d != nil {
					deployments[i] = d
				}
			}
		}
	}

	a.Deployments = deployments
	axErr, code := a.MarkObjectStopped(utils.GetStatusDetail("STOPPED", "Application is stopped.", ""))
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, nil, code
}

func (a *Application) startBackend() (*Application, *axerror.AXError, int) {
	utils.InfoLog.Printf("Starting backend of %s", a)
	// Start all its deployments
	deployments, axErr := deployment.GetLatestDeploymentsByApplication(a.Name, false)
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	for i, d := range deployments {
		switch d.Status {
		case deployment.DeployStateStopped:
			if axErr, code := axam.StartAmDeployment(d); axErr != nil {
				return nil, axErr, code
			}

			if d, axErr := deployment.GetLatestDeploymentByID(d.Id, false); axErr != nil {
				return nil, axErr, axerror.REST_INTERNAL_ERR
			} else {
				if d != nil {
					deployments[i] = d
				}
			}
		default:
			// Do nothing
		}
	}

	a.Deployments = deployments
	axErr, code := a.MarkObjectWaiting(utils.GetStatusDetail("WAITING", "Application is coming up.", ""))
	if axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, nil, code
}
