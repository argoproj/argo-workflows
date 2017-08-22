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

	"applatix.io/axamm"

	"os"

	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axops/index"
	"applatix.io/notification_center"
)

func main() {
	if len(os.Args) != 8 {
		fmt.Println("usage: axam_server http://axdb_url:port/v1 http://amm_url http://axmon_url http://adc_url http://fixture_manager_url http://axops_url kafka-zk.axsys:9092")
		os.Exit(1)
	}

	axamm.Init("AM", os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])

	deployment.Init(utils.APPLICATION_NAME)

	utils.InfoLog.Println("Initializing notification center.")
	notification_center.InitProducer(notification_center.FacilityAxam, utils.DebugLog, os.Args[7])

	// Monitor the Application HeartBeats
	deployment.ScheduleDeploymentMonitor()
	deployment.ScheduleSendingHeartbeatToAMM()
	deployment.ScheduleDeploymentResourceExtender()
	go axamm.RotateETag()
	go index.SearchIndexWorker()

	router := GetRouterAM()
	router.Run(":8968")
}
