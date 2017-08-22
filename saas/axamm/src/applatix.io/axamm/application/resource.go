package application

import (
	"applatix.io/axamm/adc"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/common"
	"time"
)

func ScheduleApplicationResourceExtender() {
	ticker := time.NewTicker(time.Minute * 20)
	go func() {
		for _ = range ticker.C {
			ExtendApplicationResource()
		}
	}()
}

func ExtendApplicationResource() *axerror.AXError {
	applications, axErr := GetLatestApplications(nil, false)
	if axErr != nil {
		return axErr
	}

	for _, app := range applications {
		extendApplicationResource(app)
	}

	return nil
}

func extendApplicationResource(app *Application) {
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

	common.DebugLog.Printf("[ADC] Procoess application %v/%v starting.\n", app.Name, app.Status)

	switch app.Status {
	case AppStateWaiting, AppStateActive, AppStateError, AppStateStopping, AppStateStopped:

		detail := map[string]string{
			"name": app.Name,
		}

		if axErr, _ := adc.Reserve(app.ID, "application", GetAppMonitorCpuCores(), GetAppMonitorMemMiB(), adc.AdcDefaultTtl, detail); axErr != nil {
			utils.ErrorLog.Printf("[ADC] Error extend application %v resource reservation: %v", app.Name, axErr)
		}
	}

	common.DebugLog.Printf("[ADC] Procoess application %v/%v finished.\n", app.Name, app.Status)
}
