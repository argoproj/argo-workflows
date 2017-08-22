// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axamm

import (
	"os"
	"time"

	"applatix.io/axamm/application"
	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axops/schema_internal"
	axopsutils "applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/restcl"
	"encoding/json"
	"strings"
)

var AX_NAMESPACE string
var AX_VERSION string

func Init(componet string,
	database string,
	amm string,
	axmon string,
	adc string,
	fixtureManager string,
	axops string) {

	utils.InitLoggers(componet)
	utils.InitAdcRedis()
	utils.InitSaaSRedis()

	if len(database) != 0 {
		utils.DbCl = axdbcl.NewAXDBClientWithTimeout(database, 5*time.Minute)
		axopsutils.Dbcl = utils.DbCl
	}

	if len(amm) != 0 {
		utils.AmmCl = restcl.NewRestClientWithTimeout(amm, 5*time.Minute)
		axopsutils.AxammCl = utils.AmmCl
	}

	if len(adc) != 0 {
		utils.AdcCl = restcl.NewRestClientWithTimeout(adc, 1*time.Minute)
		axopsutils.WorkflowAdcCl = utils.AdcCl
	}

	if len(axmon) != 0 {
		utils.AxmonCl = restcl.NewRestClientWithTimeout(axmon, 5*time.Minute)
		axopsutils.AxmonCl = utils.AxmonCl
	}

	if len(fixtureManager) != 0 {
		utils.FixMgrCl = restcl.NewRestClientWithTimeout(fixtureManager, 5*time.Minute)
		axopsutils.FixMgrCl = utils.FixMgrCl
	}

	if len(axops) != 0 {
		utils.AxopsCl = restcl.NewRestClientWithTimeout(axops, 5*time.Minute)
	}

	utils.AxNotifierCl = restcl.NewRestClientWithTimeout("http://axnotification.axsys:9889/v1", 5*time.Minute)
	axopsutils.AxNotifierCl = utils.AxNotifierCl

	AX_NAMESPACE = common.GetAxNameSpace()
	AX_VERSION = common.GetAxVersion()
	appName := common.GetApplicationName()
	if appName != "" && appName != "${APPLICATION_NAME}" {
		utils.APPLICATION_NAME = appName
	}

	if AX_NAMESPACE == "" {
		utils.ErrorLog.Printf("AX_NAMESAPCE is not available from environment variables. Abort.")
		os.Exit(1)
	}

	if AX_VERSION == "" {
		utils.ErrorLog.Printf("AX_VERSION is not available from environment variables. Abort.")
		os.Exit(1)
	}

	utils.InfoLog.Printf("AX-AMM AX_NAMESPACE:%s AX_VERSION:%s\n", AX_NAMESPACE, AX_VERSION)

	// Wait for axdb to be ready
	count := 0
	for {
		count++
		kvMap, dbErr := schema_internal.GetAppKeyValsByAppName(axdb.AXDBAppAMM, utils.DbCl)
		if dbErr != nil {
			utils.InfoLog.Printf("waiting for axdb to be ready ...... count %v error %v", count, dbErr)
		} else {
			AX_NAMESPACE_DB, _ := kvMap["AX_NAMESPACE"]
			AX_VERSION_DB, _ := kvMap["AX_VERSION"]
			utils.InfoLog.Printf("AX-AMM schema AX_NAMESPACE:%s AX_VERSION:%s\n", AX_NAMESPACE_DB, AX_VERSION_DB)
			if AX_NAMESPACE_DB != AX_NAMESPACE || AX_VERSION_DB != AX_VERSION {
				utils.InfoLog.Printf("AX-AMM schema version from system and db don't match\n")
			} else {
				utils.InfoLog.Printf("AX-AMM schema version from system and db match\n")
				break
			}
		}

		if count > 300 {
			// Give up, marathon would restart this container
			utils.ErrorLog.Printf("axdb is not available, exited")
			os.Exit(1)
		}

		time.Sleep(1 * time.Second)
	}
}

func InitTest(componet string,
	database string,
	amm string,
	axmon string,
	adc string,
	fixtureManager string,
	axops string) {

	utils.InitLoggers(componet)
	utils.InitAdcRedis()
	utils.InitSaaSRedis()

	if len(database) != 0 {
		utils.DbCl = axdbcl.NewAXDBClientWithTimeout(database, 5*time.Minute)
		axopsutils.Dbcl = utils.DbCl
	}

	if len(amm) != 0 {
		utils.AmmCl = restcl.NewRestClientWithTimeout(amm, 5*time.Minute)
		axopsutils.AxammCl = utils.AmmCl
	}

	if len(adc) != 0 {
		utils.AdcCl = restcl.NewRestClientWithTimeout(adc, 1*time.Minute)
		axopsutils.WorkflowAdcCl = utils.AdcCl
	}

	if len(axmon) != 0 {
		utils.AxmonCl = restcl.NewRestClientWithTimeout(axmon, 5*time.Minute)
		axopsutils.AxmonCl = utils.AxmonCl
	}

	if len(fixtureManager) != 0 {
		utils.FixMgrCl = restcl.NewRestClientWithTimeout(fixtureManager, 5*time.Minute)
		axopsutils.FixMgrCl = utils.FixMgrCl
	}

	if len(axops) != 0 {
		utils.AxopsCl = restcl.NewRestClientWithTimeout(axops, 5*time.Minute)
	}

	utils.AxNotifierCl = restcl.NewRestClientWithTimeout("http://axnotification.axsys:9889/v1", 5*time.Minute)
	axopsutils.AxNotifierCl = utils.AxNotifierCl
}

func RotateETag() {

	go func() {
		utils.DebugLog.Println("[ETAG]: start redis etag monitor.")
		for {
			values, axErr := utils.RedisSaaSCl.BRPopWithTTL(3*time.Minute, deployment.RedisDeployUpdate)
			if axErr != nil {
				utils.DebugLog.Printf("[ETAG]: error for BRPOP - %v.\n", axErr)
			} else {
				if len(values) != 0 && values[0] != "" {
					deployment.UpdateETag()
					utils.DebugLog.Println("[ETAG]: etag rotated for deployments.")
				}

				for i, _ := range values {
					if strings.HasPrefix(values[i], "{") {
						var event deployment.RedisDeploymentResult
						err := json.Unmarshal([]byte(values[i]), &event)
						if err != nil {
							utils.ErrorLog.Printf("[STREAM]: failed to unmarshal the redis event(%v): %v\n", values[i], err.Error())
						} else {
							deployment.PostStatusEvent(event.Id, event.Name, event.Status, event.StatusDetail)
						}
					}
				}
			}
		}
	}()

	secondsToHour := 60 - time.Now().Unix()%60
	utils.DebugLog.Printf("[ETAG]: sleep %v seconds to start etag rotation.\n", secondsToHour)
	time.Sleep(time.Duration(secondsToHour * int64(time.Second)))

	rotate := func() {
		application.UpdateETag()
		deployment.UpdateETag()
		utils.DebugLog.Println("[ETAG]: etag rotated.")
	}

	rotate()

	ticker := time.NewTicker(time.Minute * 1)
	go func() {
		for _ = range ticker.C {
			rotate()
		}
	}()
}
