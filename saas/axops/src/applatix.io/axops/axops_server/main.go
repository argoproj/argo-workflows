// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @APIVersion 1.0
// @APITitle Applatix API
// @APIDescription This is the documentation for Applatix APIs.
// @Contact support@applatix.com
// @TermsOfServiceUrl www.applatix.com
// @BasePath /v1
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"applatix.io/axops"
	"applatix.io/axops/cluster"
	"applatix.io/axops/event"
	"applatix.io/axops/index"
	"applatix.io/axops/sandbox"
	"applatix.io/axops/secret"
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/monitor"
	"applatix.io/notification_center"
)

func main() {
	if len(os.Args) < 9 {
		fmt.Println("usage: axops_server http://axdb_ip:axdb_port/axdb_version http://devops_gateway_url http://axworkflowadc_url http://axmon_url http://axnotification_url http://fixture_manager_url  " +
			"http://scheduler_url http://artifact_manager_url <kafka_url>")
		os.Exit(1)
	}

	axops.Init(os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6], os.Args[7], os.Args[8])

	tool.IsSystemInitialized = false

	go monitor.MonitorSystem(60*time.Second, utils.DebugLog)
	go axops.ProfilerWorker()
	go index.SearchIndexWorker()
	go axops.GCLables()
	go axops.GCTemplatePolicyProjectFixture()
	go axops.MonitorRunningService()
	go axops.RotateETagHourly()
	go axops.RefreshScmToolScheduler()

	sandbox.InitSandbox()

	axops.PushScmTools()
	utils.InfoLog.Println("Loaded and pushed SCM tools.")

	axops.PushNotificationConfig()
	utils.InfoLog.Println("Loaded and pushed Notification configurations.")

	axops.ApplyAuthenticationConfig()
	utils.InfoLog.Println("Loaded and pushed Authentication configurations.")

	axops.LoadCustomCert()
	utils.InfoLog.Println("Loaded and applied Certificates.")

	secret.LoadRSAKey()
	utils.InfoLog.Println("Loaded RSA Key.")

	if axErr := axops.CreateDomainManagementTool(); axErr != nil {
		utils.ErrorLog.Println("Failed to load domain configuartions:", axErr)
		//os.Exit(1)
	} else {
		utils.InfoLog.Println("Loaded the domain configurations.")
	}

	if axErr := cluster.InitClusterSettings(); axErr != nil {
		utils.ErrorLog.Println("Failed to load cluster settings:", axErr)
		//os.Exit(1)
	} else {
		utils.InfoLog.Println("Loaded the cluster settings.")
	}

	// Start the event handler as the last step before API server
	if len(os.Args) >= 10 {
		event.Init(os.Args[9])
		notification_center.InitProducer(notification_center.FacilityAxops, common.DebugLog, os.Args[9])
	} else {
		event.Init()
		notification_center.InitProducer(notification_center.FacilityAxops, common.DebugLog, event.KafkaServiceName)
	}

	tool.IsSystemInitialized = true

	publicRouter := axops.GetRounter(false)
	publicRouter.LoadHTMLGlob("../public/internal/profile/*")

	swaggify(publicRouter)

	// Officially sanctioned HTTP router (applatix.axsys) for users to access the API from within cluster (no SSL)
	go publicRouter.Run(":8082")

	// HTTPs router for handling 443 outside
	go publicRouter.RunTLS(":8086", utils.GetPublicCertPath(), utils.GetPrivateKeyPath())

	// SCM Web hook router for handling 8443 outside
	webHookRounter := axops.GetScmWebHookRounter()
	go webHookRounter.RunTLS(":8087", utils.GetPublicCertPath(), utils.GetPrivateKeyPath())

	// Jira Web hook router
	jiraWebHookRounter := axops.GetJiraWebHookRounter()
	go jiraWebHookRounter.Run(":8088")

	// Redirect to HTTPs router for handling 80 to 443 outside
	go http.ListenAndServe(":8081", http.HandlerFunc(redirect))

	// Internal router for other internal components
	internalRouter := axops.GetRounter(true)
	internalRouter.Run(":8085")
}

func redirect(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req,
		"https://"+req.Host+req.URL.String(),
		http.StatusMovedPermanently)
}
