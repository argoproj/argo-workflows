// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"fmt"
	"time"

	"applatix.io/axdb"
	"applatix.io/axops/event"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/template"
	"gopkg.in/check.v1"
)

func (ss *S) TestServiceLifeCycle(t *check.C) {
	workflowTemplateStr, axErr := utils.ReadFromFile("testdata/template/workflow.json")
	t.Assert(axErr, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTasks, event.GetDevopsTaskHandler())

	tmpl, axErr := service.UnmarshalEmbeddedTemplate([]byte(workflowTemplateStr))
	if axErr != nil {
		fail(t)
	}
	workflowTemplate := tmpl.(service.EmbeddedWorkflowTemplate)

	data := &GeneralGetResult{}
	err := axopsClient.Get("commits", nil, data)
	t.Assert(err, check.IsNil)

	commit := "fake-commit"
	branch := "fake-branch"
	repo := "fake-repo"
	if len(data.Data) != 0 {
		m := data.Data[0]
		commit = m["revision"].(string)
		branch = m["branch"].(string)
		repo = m["repo"].(string)
	}

	// first post a service
	s := service.Service{}
	s.Template = workflowTemplate
	s.Arguments = make(template.Arguments)
	s.Arguments["session.commit"] = &commit
	s.Arguments["session.branch"] = &branch
	s.Arguments["session.repo"] = &repo
	resultMap, axErr := axopsClient.Post("services", s)
	checkError(t, axErr)

	// We should be able to find the service we just created
	s = service.Service{}
	axErr = axopsClient.Get("services/"+resultMap["id"].(string), nil, &s)
	checkError(t, axErr)

	checkoutId := workflowTemplate.Steps[0]["checkout"].Id
	buildId := workflowTemplate.Steps[1]["axdb_build"].Id

	checkServiceStatus := func(serviceid string, expectedStatus int) {
		params := map[string]interface{}{axdb.AXDBUUIDColumnName: serviceid}
		var tableName string
		if expectedStatus > 0 {
			tableName = service.RunningServiceTable
		} else {
			tableName = service.DoneServiceTable
		}
		serviceArray, axErr := service.GetServicesFromTable(tableName, false, params)
		if len(serviceArray) != 1 || axErr != nil {
			fail(t)
		} else {
			updatedService := serviceArray[0]
			if updatedService.Status != expectedStatus {
				t.Logf("Failing because service status %d doesn't match expected status %d", updatedService.Status, expectedStatus)
				fail(t)
			}
		}
	}
	eventMap := map[string]interface{}{"status": "WAITING", "service_id": checkoutId}
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusWaiting)

	eventMap["status"] = "RUNNING"
	eventMap["service_id"] = checkoutId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	eventMap["service_id"] = s.Id
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusRunning)
	checkServiceStatus(buildId, utils.ServiceStatusInitiating)
	checkServiceStatus(s.Id, utils.ServiceStatusRunning)

	eventMap["status"] = "COMPLETE"
	eventMap["result"] = "SUCCESS"
	eventMap["service_id"] = checkoutId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusSuccess)
	checkServiceStatus(buildId, utils.ServiceStatusInitiating)
	checkServiceStatus(s.Id, utils.ServiceStatusRunning)

	eventMap = map[string]interface{}{"status": "RUNNING"}
	eventMap["service_id"] = buildId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusSuccess)
	checkServiceStatus(buildId, utils.ServiceStatusRunning)
	checkServiceStatus(s.Id, utils.ServiceStatusRunning)

	eventMap["status"] = "COMPLETE"
	eventMap["result"] = "SUCCESS"
	eventMap["service_id"] = buildId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	eventMap["service_id"] = s.Id
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusSuccess)
	checkServiceStatus(buildId, utils.ServiceStatusSuccess)
	checkServiceStatus(s.Id, utils.ServiceStatusSuccess)

	// This time test service failure
	s = service.Service{}
	s.Template = workflowTemplate
	s.Arguments = make(template.Arguments)
	s.Arguments["session.commit"] = &commit
	s.Arguments["session.branch"] = &branch
	s.Arguments["session.repo"] = &repo
	resultMap, axErr = axopsClient.Post("services", s)
	checkError(t, axErr)

	// We should be able to find the service we just created
	s = service.Service{}
	axErr = axopsClient.Get("services/"+resultMap["id"].(string), nil, &s)
	checkError(t, axErr)

	checkoutId = workflowTemplate.Steps[0]["checkout"].Id
	buildId = workflowTemplate.Steps[1]["axdb_build"].Id

	eventMap = map[string]interface{}{"status": "WAITING"}
	eventMap["service_id"] = checkoutId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusWaiting)

	eventMap["status"] = "RUNNING"
	eventMap["service_id"] = checkoutId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	eventMap["service_id"] = s.Id
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusRunning)
	checkServiceStatus(buildId, utils.ServiceStatusInitiating)
	checkServiceStatus(s.Id, utils.ServiceStatusRunning)

	eventMap["status"] = "COMPLETE"
	eventMap["result"] = "SUCCESS"
	eventMap["service_id"] = checkoutId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusSuccess)
	checkServiceStatus(buildId, utils.ServiceStatusInitiating)
	checkServiceStatus(s.Id, utils.ServiceStatusRunning)

	eventMap = map[string]interface{}{"status": "RUNNING"}
	eventMap["service_id"] = buildId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusSuccess)
	checkServiceStatus(buildId, utils.ServiceStatusRunning)
	checkServiceStatus(s.Id, utils.ServiceStatusRunning)

	eventMap["status"] = "COMPLETE"
	eventMap["result"] = "FAILURE"
	eventMap["service_id"] = buildId
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	eventMap["service_id"] = s.Id
	PostOneEvent(t, event.TopicDevopsTasks, s.Id, "status", eventMap)
	time.Sleep(2 * time.Second)
	checkServiceStatus(checkoutId, utils.ServiceStatusSuccess)
	checkServiceStatus(buildId, utils.ServiceStatusFailed)
	checkServiceStatus(s.Id, utils.ServiceStatusFailed)

	// check failure_path
	s0, _ := service.GetServiceDetail(s.Id, nil)
	fmt.Printf("failure path is %v", s0.FailurePath)
	t.Assert(s0.FailurePath, check.HasLen, 1)
}

func (ss *S) TestServiceExpandParameters(t *check.C) {

	workflowTemplateStr, axErr := utils.ReadFromFile("testdata/template/workflow_expand.json")
	t.Assert(axErr, check.IsNil)

	event.RegisterEventHandler(event.TopicDevopsTasks, event.GetDevopsTaskHandler())

	workflowTemplate, axErr := service.UnmarshalEmbeddedTemplate([]byte(workflowTemplateStr))
	if axErr != nil {
		fail(t)
	}

	command := "$$[echo 1,echo 2,echo 3]$$"
	branch := "fake-branch"
	repo := "fake-repo"
	commit := "fake-commit"

	// first post a service
	s := service.Service{}
	s.Template = workflowTemplate
	s.Arguments = make(template.Arguments)
	s.Arguments["session.commit"] = &commit
	s.Arguments["session.branch"] = &branch
	s.Arguments["session.repo"] = &repo
	s.Arguments["command"] = &command
	resultMap, axErr := axopsClient.Post("services", s)
	checkError(t, axErr)

	// We should be able to find the service we just created
	s = service.Service{}
	axErr = axopsClient.Get("services/"+resultMap["id"].(string), nil, &s)
	checkError(t, axErr)

	// Verify that the build step has 3 services.
	fmt.Printf("children len %d", len(s.Children))
	fmt.Printf("expanded service: \n%v", s)
	t.Assert(len(s.Children), check.Equals, 4)
}
