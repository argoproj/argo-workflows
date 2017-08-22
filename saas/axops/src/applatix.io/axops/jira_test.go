package axops_test

import (
	"applatix.io/axdb"
	"applatix.io/axops/jira"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/template"
	"gopkg.in/check.v1"
)

func (ss *S) TestJiraCRUD(t *check.C) {
	workflowTemplateStr, axErr := utils.ReadFromFile("testdata/template/workflow.json")
	t.Assert(axErr, check.IsNil)
	workflowTemplate, axErr := service.UnmarshalEmbeddedTemplate([]byte(workflowTemplateStr))
	if axErr != nil {
		fail(t)
	}

	jiraBody := map[string]interface{}{
		"project":     "fake_project",
		"id":          "fp-001",
		"summary":     "test",
		"description": "jira_test",
		"status":      "To Do",
	}

	_, axErr = axdbClient.Post(axdb.AXDBAppAXOPS, jira.JiraBodyTable, jiraBody)
	checkError(t, axErr)

	// query the jira created just now
	params := map[string]interface{}{
		jira.JiraId: "fp-001",
	}
	resultArray := []map[string]interface{}{}

	axErr = axdbClient.Get(axdb.AXDBAppAXOPS, jira.JiraBodyTable, params, &resultArray)
	checkError(t, axErr)
	t.Assert(len(resultArray), check.Equals, 1)

	// test update to jira summary
	jiraBody["summary"] = "test1"
	_, axErr = axopsClient.Put("jira/issues", jiraBody)
	checkError(t, axErr)
	axErr = axdbClient.Get(axdb.AXDBAppAXOPS, jira.JiraBodyTable, params, &resultArray)
	checkError(t, axErr)
	t.Assert(len(resultArray), check.Equals, 1)
	t.Assert(resultArray[0]["summary"].(string), check.Equals, "test1")

	// test update to jira status
	jiraBody["status"] = "In Progress"
	_, axErr = axopsClient.Put("jira/issues", jiraBody)
	checkError(t, axErr)
	axErr = axdbClient.Get(axdb.AXDBAppAXOPS, jira.JiraBodyTable, params, &resultArray)
	checkError(t, axErr)
	t.Assert(len(resultArray), check.Equals, 1)
	t.Assert(resultArray[0]["status"].(string), check.Equals, "In Progress")

	// post a service
	s := service.Service{}
	s.Template = workflowTemplate
	s.Arguments = make(template.Arguments)
	s.Arguments["session.commit"] = utils.NewString("jira_commit")
	s.Arguments["session.repo"] = utils.NewString("jira_repo")
	resultMap, axErr := axopsClient.Post("services", s)
	checkError(t, axErr)

	//sid := resultMap[axdb.AXDBUUIDColumnName]
	sid := resultMap["id"]
	//associate jira with service
	_, axErr = axopsClient.Put("jira/issues/fp-001/service/"+sid.(string), nil)
	checkError(t, axErr)

	// jira should have this service id
	axErr = axdbClient.Get(axdb.AXDBAppAXOPS, jira.JiraBodyTable, params, &resultArray)
	checkError(t, axErr)
	t.Assert(len(resultArray), check.Equals, 1)
	jobList := resultArray[0][jira.JobList].([]interface{})
	var exist bool = false
	for _, jobIdObj := range jobList {
		jobId := jobIdObj.(string)
		if jobId == sid {
			exist = true
			break
		}
	}
	t.Assert(exist, check.Equals, true)

	// the service should contains this jira
	params = map[string]interface{}{
		axdb.AXDBUUIDColumnName: sid.(string),
	}
	srvMap, axErr := service.GetServiceMapByID(sid.(string))
	checkError(t, axErr)
	jiraList := srvMap[service.ServiceJiraIssues].([]interface{})

	exist = false
	for _, jiraObj := range jiraList {
		jiraId := jiraObj.(string)
		if jiraId == "fp-001" {
			exist = true
			break
		}
	}
	t.Assert(exist, check.Equals, true)
}
