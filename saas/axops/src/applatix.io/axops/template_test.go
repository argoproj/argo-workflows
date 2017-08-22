// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

/*
import (
	"applatix.io/axops"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"encoding/json"
	"gopkg.in/check.v1"
)

type TemplateListResult struct {
	Data []service.ServiceTemplate `json:"data,omitempty"`
}

func (s *S) TestTemplate(t *check.C) {
	workflowTemplateStr, axErr := utils.ReadFromFile("testdata/template/workflow.json")
	t.Assert(axErr, check.IsNil)

	var workflowTemplate service.ServiceTemplate
	err := json.Unmarshal([]byte(workflowTemplateStr), &workflowTemplate)
	if err != nil {
		fail(t)
	}

	// First, delete the templates if they exist. Otherwise can't post new ones with the same name.
	resultMap := map[string]interface{}{}
	axErr = axopsClient.Get("templates", nil, &resultMap)
	checkError(t, axErr)

	templateArray := resultMap[axops.RestData].([]interface{})
	for _, temp := range templateArray {
		axopsClient.Delete("templates/"+temp.(map[string]interface{})["id"].(string), nil)
	}

	// post all the templates the workflow depends on
	checkoutMap, axErr := axopsClient.Post("templates", workflowTemplate.Steps[0]["checkout"].Template)
	checkError(t, axErr)
	workflowTemplate.Steps[0]["checkout"].Template.Id = checkoutMap["id"].(string)

	buildMap, axErr := axopsClient.Post("templates", workflowTemplate.Steps[1]["axdb_build"].Template)
	checkError(t, axErr)
	workflowTemplate.Steps[1]["axdb_build"].Template.Id = buildMap["id"].(string)

	workflowMap, axErr := axopsClient.Post("templates", workflowTemplate)
	checkError(t, axErr)
	workflowTemplate.Id = workflowMap["id"].(string)

	resultMap = map[string]interface{}{}
	axErr = axopsClient.Get("templates", nil, &resultMap)
	checkError(t, axErr)

	templateArray = resultMap[axops.RestData].([]interface{})
	count := len(templateArray)
	if count < 3 {
		t.Logf("Too few templates found %d", count)
		fail(t)
	}

	// look for templates that are compatible with commits, we should find workflow, checkout, but not axdb_build
	var tempResult TemplateListResult
	axErr = axopsClient.Get("templates?commit=aaa", nil, &tempResult)
	checkError(t, axErr)

	foundWorkflow := false
	foundCheckout := false
	foundBuild := false
	for _, temp := range tempResult.Data {
		if temp.Id == checkoutMap["id"].(string) {
			foundCheckout = true
		} else if temp.Id == buildMap["id"].(string) {
			foundBuild = true
		} else if temp.Id == workflowTemplate.Id {
			foundWorkflow = true
		}
	}
	if !foundWorkflow || !foundCheckout || foundBuild {
		fail(t)
	}

	// Now modify the build template to accept commit parameter
	var buildTemp *service.ServiceTemplate
	axErr = axopsClient.Get("templates/"+buildMap["id"].(string), nil, &buildTemp)
	checkError(t, axErr)
	if buildTemp == nil {
		fail(t)
	}
	buildTemp.Inputs.Parameters["c"] = &service.TemplateParameterInput{Default: "%%session.commit%%"}

	_, axErr = axopsClient.Put("templates/"+buildMap["id"].(string), buildTemp)
	checkError(t, axErr)

	axErr = axopsClient.Get("templates?commit=aaa", nil, &tempResult)
	checkError(t, axErr)

	foundWorkflow = false
	foundCheckout = false
	foundBuild = false
	t.Logf("Looking for workflow %s checkout %s build %s", workflowTemplate.Id, checkoutMap["id"].(string), buildMap["id"].(string))
	for _, temp := range tempResult.Data {
		t.Logf("got %s %s ", temp.Id, temp.Name)
		if temp.Id == checkoutMap["id"].(string) {
			foundCheckout = true
		} else if temp.Id == buildMap["id"].(string) {
			foundBuild = true
		} else if temp.Id == workflowTemplate.Id {
			foundWorkflow = true
		}
	}
	if !foundWorkflow || !foundCheckout || !foundBuild {
		fail(t)
	}

	for _, entry := range templateArray {
		temp := entry.(map[string]interface{})
		temp["description"] = "test desc"
		id := temp["id"].(string)
		_, axErr = axopsClient.Put("templates/"+id, temp)
		checkError(t, axErr)

		var getTemp service.ServiceTemplate
		axErr = axopsClient.Get("templates/"+id, nil, &getTemp)
		checkError(t, axErr)

		if getTemp.Description != "test desc" {
			fail(t)
		}

		_, axErr = axopsClient.Delete("templates/"+id, nil)
		checkError(t, axErr)
	}

	axErr = axopsClient.Get("templates", nil, &resultMap)
	checkError(t, axErr)
	templateArray = resultMap[axops.RestData].([]interface{})
	count = len(templateArray)
	if count != 0 {
		t.Logf("Too many templates found %d", count)
		fail(t)
	}

	t.Log("TestTemplate done")
}
*/
