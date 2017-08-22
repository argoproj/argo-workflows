package application

import (
	"time"

	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/common"
)

// AppDeletionGracePeriod is the time period in seconds in which we will delete a terminated application.
const AppDeletionGracePeriod = 2 * 24 * 60 * 60

func ScheduleApplicationMonitor() {
	ticker := time.NewTicker(time.Minute * 1)
	go func() {
		for _ = range ticker.C {
			monitorApplications()
		}
	}()
}

func monitorApplications() *axerror.AXError {
	common.DebugLog.Printf("Begin application monitor loop")
	applications, axErr := GetLatestApplications(nil, false)
	if axErr != nil {
		return axErr
	}

	for _, app := range applications {
		monitorApplication(app)
	}
	common.DebugLog.Printf("Application monitor loop completed (%d applications)", len(applications))
	return nil
}

func monitorApplication(app *Application) {
	//locked := AppLockGroup.TryLock(app.Key(), time.Duration(-1))
	//if locked {
	//	defer AppLockGroup.Unlock(app.Key())
	//} else {
	//	return
	//}

	AppLockGroup.Lock(app.Key())
	defer AppLockGroup.Unlock(app.Key())

	latest, axErr := GetLatestApplicationByID(app.ID, false)
	if axErr != nil {
		common.ErrorLog.Printf("Error fetch application %v: %v", app.Name, axErr)
	}

	if latest == nil {
		return
	}
	app = latest

	if app.Status != AppStateActive && app.Status != AppStateTerminated {
		// Don't log spurious messages in the common (Active) case
		common.DebugLog.Printf("[HB] Check application %v/%v starting\n", app.Name, app.Status)
	}

	switch app.Status {
	case AppStateInit:
		_, axErr, _ = app.createBackend(false)
		if axErr != nil {
			common.ErrorLog.Printf("Error create application %v: %v", app.Name, axErr)
			axErr, _ = app.MarkObjectInit(utils.GetStatusDetail(ErrCreatingApp, axErr.Message, axErr.Detail))
			if axErr != nil {
				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
			}
		}
	case AppStateActive:
		fresh := heartbeat.GetFreshness(app.Key())
		if time.Now().Unix()-fresh > HEART_BEAT_GRACE_PERIOD {
			common.ErrorLog.Printf("[HB] Heartbeats for %v is missing for %v seconds.\n", app.Name, time.Now().Unix()-fresh)
			axErr, _ = app.MarkObjectError(utils.GetStatusDetail(ErrMissingHeartBeat, "", ""))
			if axErr != nil {
				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
			}
		}
	case AppStateStopping:
		_, axErr, _ := app.stopBackend()
		if axErr != nil {
			common.ErrorLog.Printf("Error stop application %v: %v", app.Name, axErr)
			axErr, _ = app.MarkObjectStopping(utils.GetStatusDetail(ErrStoppingApp, axErr.Message, axErr.Detail))
			if axErr != nil {
				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
			}
		}
	case AppStateTerminating:
		_, axErr, _ = app.deleteBackend()
		if axErr != nil {
			common.ErrorLog.Printf("Error delete application %v: %v", app.Name, axErr)
			axErr, _ = app.MarkObjectTerminating(utils.GetStatusDetail(ErrDeletingApp, axErr.Message, axErr.Detail))
			if axErr != nil {
				common.ErrorLog.Printf("Error update application %v status: %v", app.Name, axErr)
			}
		}
	case AppStateTerminated:
		terminatedSeconds := time.Now().Unix() - int64(app.Mtime/1e6)
		if terminatedSeconds > AppDeletionGracePeriod {
			utils.InfoLog.Printf("Deleting old terminated %s (terminated for %d > %d sec)", app, terminatedSeconds, AppDeletionGracePeriod)
			if axErr, _ := app.DeleteObject(); axErr != nil {
				common.ErrorLog.Printf("Error deleting %s: %v", app, axErr)
			}
		}
	}
	if app.Status != AppStateActive && app.Status != AppStateTerminated {
		// Don't log spurious messages in the common (Active) case
		common.DebugLog.Printf("[HB] Check application %v/%v finished\n", app.Name, app.Status)
	}
}
