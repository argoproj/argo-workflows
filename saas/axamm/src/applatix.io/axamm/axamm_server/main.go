// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @APIVersion 1.0
// @APITitle Applatix API
// @APIDescription This is the documentation for Applatix APIs.
// @Contact support@applatix.com
// @TermsOfServiceUrl www.applatix.com
// @BasePath /v1
package main

import (
	"fmt"
	"os"

	"applatix.io/axamm"
	"applatix.io/axamm/application"
	"applatix.io/axamm/utils"
	"applatix.io/axops/index"
	"applatix.io/common"
	"applatix.io/notification_center"
)

func main() {
	if len(os.Args) != 8 {
		fmt.Println("usage: axamm_server http://axdb_url:port/v1 http://amm_url http://axmon_url http://adc_url http://fixture_manager_url http://axops_url kafka-zk:9092")
		os.Exit(1)
	}

	axamm.Init("AMM", os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])

	utils.InfoLog.Println("Initializing notification center.")
	notification_center.InitProducer(notification_center.FacilityAxamm, common.DebugLog, os.Args[7])

	axErr := application.Init()
	if axErr != nil {
		utils.ErrorLog.Printf("Failed to init the applications: %v.\n", axErr)
		os.Exit(1)
	}

	// Monitor the Application HeartBeats
	application.ScheduleApplicationMonitor()
	application.ScheduleApplicationResourceExtender()
	go axamm.RotateETag()
	go index.SearchIndexWorker()

	router := GetRouterAMM()
	router.Run(":8966")
}
