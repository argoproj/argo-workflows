package application

import (
	"applatix.io/axamm/deployment"
	"applatix.io/axamm/heartbeat"
	"applatix.io/axerror"
	"applatix.io/lock"
)

var AppLockGroup lock.LockGroup

type Result struct {
	Result Detail `json:"result,omitempty"`
}

type Detail struct {
	Monitor   *bool `json:"monitor,omitempty"`
	Namespace *bool `json:"namespace,omitempty"`
	Registry  *bool `json:"registry,omitempty"`
}

func Init() *axerror.AXError {
	AppLockGroup.Name = "AppLock"
	AppLockGroup.Init()

	deployment.InitLock()

	applications, axErr := GetLatestApplications(nil, false)
	if axErr != nil {
		return axErr
	}

	for _, app := range applications {
		switch app.Status {
		case AppStateTerminated:
		case AppStateTerminating:
		default:
			heartbeat.RegisterHandler(app.Key(), GetHeartBeatHandler())
		}
	}

	return monitorApplications()
}

//func initApplication(app *Application) {
//	var axErr *axerror.AXError
//	switch app.Status {
//	case AppStateInit:
//		heartbeat.RegisterHandler(app.Key(), GetHeartBeatHandler())
//		_, axErr, _ = app.createBackend(true)
//		if axErr != nil {
//			common.ErrorLog.Printf("Error create application %v: %v", app.Name, axErr)
//			app.StatusDetail = map[string]interface{}{
//				"code":    ErrCreatingApp,
//				"message": axErr.Message,
//				"detail":  axErr.Detail,
//			}
//
//			_, axErr, _ = app.UpdateObject()
//			if axErr != nil {
//				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
//			}
//		}
//	case AppStateTerminating:
//		_, axErr, _ = app.deleteBackend()
//		if axErr != nil {
//			common.ErrorLog.Printf("Error delete application %v: %v", app.Name, axErr)
//			app.StatusDetail = map[string]interface{}{
//				"code":    ErrDeletingApp,
//				"message": axErr.Message,
//				"detail":  axErr.Detail,
//			}
//
//			_, axErr, _ = app.UpdateObject()
//			if axErr != nil {
//				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
//			}
//		}
//	case AppStateError:
//		heartbeat.RegisterHandler(app.Key(), GetHeartBeatHandler())
//		_, axErr, _ = app.createBackend(false)
//		if axErr != nil {
//			common.ErrorLog.Printf("Error recreate application %v status: %v", app.Name, axErr)
//		}
//
//		app.Status = AppStateWaiting
//		app.StatusDetail = map[string]interface{}{}
//		_, axErr, _ = app.UpdateObject()
//		if axErr != nil {
//			common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
//		}
//	case AppStateActive:
//		heartbeat.RegisterHandler(app.Key(), GetHeartBeatHandler())
//		var result Result
//		axErr := utils.AxmonCl.Get("axmon/application/"+app.Name, nil, &result)
//		if axErr != nil {
//			common.ErrorLog.Printf("Error get application %v status: %v", app.Name, axErr)
//		}
//
//		error := false
//		if result.Result.Namespace == nil || *result.Result.Namespace != true {
//			error = true
//			app.Status = AppStateError
//			app.StatusDetail = map[string]interface{}{
//				"code": ErrMissingNameSpace,
//			}
//			_, axErr, _ = app.UpdateObject()
//			if axErr != nil {
//				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
//			}
//		} else if result.Result.Monitor == nil || *result.Result.Monitor != true {
//			error = true
//			app.Status = AppStateError
//			app.StatusDetail = map[string]interface{}{
//				"code": ErrMissingMonitor,
//			}
//			_, axErr, _ = app.UpdateObject()
//			if axErr != nil {
//				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
//			}
//		} else if result.Result.Registry == nil || *result.Result.Registry != true {
//			error = true
//			app.Status = AppStateError
//			app.StatusDetail = map[string]interface{}{
//				"code": ErrMissingRegistrySecret,
//			}
//			_, axErr, _ = app.UpdateObject()
//			if axErr != nil {
//				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
//			}
//		} else {
//			// TODO: check the AM status
//		}
//
//		if error {
//			// Try to resolve situation once
//			_, axErr, _ = app.createBackend(false)
//			if axErr != nil {
//				common.ErrorLog.Printf("Error recreate application %v: %v", app.Name, axErr)
//			}
//
//			app.Status = AppStateWaiting
//			app.StatusDetail = map[string]interface{}{}
//			_, axErr, _ = app.UpdateObject()
//			if axErr != nil {
//				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
//			}
//		}
//	case AppStateStopping, AppStateStopped, AppStateWaiting:
//		heartbeat.RegisterHandler(app.Key(), GetHeartBeatHandler())
//	default:
//		// Do nothing
//	}
//}
