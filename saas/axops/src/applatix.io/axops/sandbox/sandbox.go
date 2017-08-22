package sandbox

import (
	"applatix.io/axops/service"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	sandboxEnvVar                   = "SANDBOX_ENABLED"
	monitorJobInterval              = 10 * time.Minute
	emailCacheRefreshInterval       = 10 * time.Minute
	maxJobRunInterval         int64 = 60 * 60 // 60 minute
	maxConcurrentJobPerUser   int   = 10
)

type emailCacheEntry struct {
	FN string
	Id string
}

var sandboxEmailCache = map[string]*emailCacheEntry{}
var sandboxFlag bool
var sandboxInited bool
var monitorJobTicker *time.Ticker
var emailCacheRefreshTicker *time.Ticker
var mux sync.Mutex

func InitSandbox() {
	mux.Lock()
	defer mux.Unlock()
	if sandboxInited {
		utils.ErrorLog.Println("initSandbox called more than once, will skip reinitializing")
		return
	}
	readSandboxEnvVar()
	if IsSandboxEnabled() {
		utils.InfoLog.Println("Running in sandbox mode")
		monitorJobTicker = time.NewTicker(monitorJobInterval)
		emailCacheRefreshTicker = time.NewTicker(emailCacheRefreshInterval)
		go monitorJobsInSandbox()
		go refreshSandboxEmailCache()
	}
	sandboxInited = true
}

func ResetSandbox() {
	mux.Lock()
	defer mux.Unlock()
	if !sandboxInited || !IsSandboxEnabled() {
		return
	}
	monitorJobTicker.Stop()
	emailCacheRefreshTicker.Stop()
	sandboxFlag = false
	sandboxInited = false
	sandboxEmailCache = map[string]*emailCacheEntry{}
}

func readSandboxEnvVar() {
	utils.DebugLog.Printf("reading env var:%v", sandboxEnvVar)
	sandboxFlag = strings.ToLower(strings.TrimSpace(os.Getenv(sandboxEnvVar))) == "true"
}

func IsSandboxEnabled() bool {
	return sandboxFlag
}

func ReplaceEmailInSandbox(email string) string {
	if !IsSandboxEnabled() {
		return email
	}
	if entry, ok := sandboxEmailCache[email]; ok {
		return entry.FN
	} else if strings.Contains(email, "@") {
		return email[:strings.LastIndex(email, "@")]
	} else {
		return email
	}
}

func GetUserIdForEmail(email string) string {
	if !IsSandboxEnabled() {
		return email
	}
	if entry, ok := sandboxEmailCache[email]; ok {
		return entry.Id
	}
	return email
}

func MaxConcurrentJobLimitReached(userId string) bool {

	utils.DebugLog.Printf("checking nunber of running jobs for user:%v", userId)
	// get all running jobs for a user
	params := map[string]interface{}{
		service.ServiceIsTask: true,
		service.ServiceUserId: userId,
	}
	serviceArray, axErr := service.GetServicesFromTable(service.RunningServiceTable, false, params)

	if axErr != nil {
		utils.InfoLog.Printf("check for max running jobs for a user failed with: %v, will skip the check", axErr)
		return true
	} else {
		utils.DebugLog.Printf("found %v running jobs for user:%v", len(serviceArray), userId)
		return len(serviceArray) >= maxConcurrentJobPerUser
	}

}

func monitorJobsInSandbox() {
	for {
		utils.DebugLog.Println("checking for long running jobs")
		// get all running jobs
		params := map[string]interface{}{
			service.ServiceIsTask: true,
		}
		serviceArray, axErr := service.GetServicesFromTable(service.RunningServiceTable, false, params)

		if axErr != nil {
			utils.InfoLog.Printf("check for long running jobs failed with: %v, will retry later", axErr)
		} else {
			for _, s := range serviceArray {
				if s.RunTime > maxJobRunInterval {
					utils.InfoLog.Printf("deleting service with id: %v running for %v sec "+
						"and exceeding the max duration of %v sec", s.Id, s.RunTime, maxJobRunInterval)
					_, axErr := utils.WorkflowAdcCl.Delete("workflows/"+s.Id, nil)
					if axErr != nil {
						utils.InfoLog.Printf("delete failed for service: %v with error: %v, will retry later", s.Id, axErr)
					}
				}
			}
		}
		_, ok := <-monitorJobTicker.C
		if !ok {
			return
		}
	}
}

func refreshSandboxEmailCache() {
	for {
		utils.DebugLog.Println("refreshing email to name cache")
		userArray, axErr := user.GetUsers(nil)
		if axErr != nil {
			utils.InfoLog.Printf("refresh of email to name cache failed with:%v, will retry", axErr)
		} else {
			cache := make(map[string]*emailCacheEntry)
			for _, u := range userArray {
				entry := &emailCacheEntry{FN: u.FirstName, Id: u.ID}
				cache[u.Username] = entry
			}
			sandboxEmailCache = cache
		}
		_, ok := <-emailCacheRefreshTicker.C
		if !ok {
			return
		}

	}
}
