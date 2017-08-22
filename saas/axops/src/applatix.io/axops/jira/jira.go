package jira

import (
	"applatix.io/axamm/application"
	"applatix.io/axamm/deployment"
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"fmt"
	"strings"
)

type JiraIssue struct {
	Project     string `json:"project,omitempty"`
	JiraId      string `json:"id,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Issuetype   string `json:"issuetype,omitempty"`
	Reporter    string `json:"reporter,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	OldJiraId   string `json:"old_id,omitempty"`
}

type JiraIssueBodyInDB struct {
	JiraIssue
	JobList    []string `json:"job_list"`
	AppList    []string `json:"app_list"`
	DeployList []string `json:"deploy_list"`
}

func DeleteJiraIssue(id string) *axerror.AXError {
	jira, axErr := GetJiraBodyByID(id)
	if axErr != nil {
		utils.ErrorLog.Printf("Failed to get Jira Body from DB (Err: %v)", axErr)
		return axErr
	}

	// if the jira to be delete doesn't exist, just ignore it.
	if jira == nil {
		utils.InfoLog.Printf("The Jira to be deleted doesn't exist in DB, just ignore it.")
		return nil
	}

	utils.InfoLog.Printf("The Jira to be deleted is %v", jira)
	axErr = updateJiraAssociation(jira, id)
	if axErr != nil {
		return axErr
	}
	//finally remove the jira from jira metadata table
	return DeleteJiraFromDBByID(id)
}

func UpdateJiraIssue(jiraBody JiraIssue) *axerror.AXError {
	jiraMap, axErr := GetJiraBodyByID(jiraBody.JiraId)
	if axErr != nil {
		utils.ErrorLog.Printf("Failed to get Jira Body from DB (Err: %v)", axErr)
		return axErr
	}

	if jiraMap == nil {
		utils.ErrorLog.Printf("The Jira %s to be updated doesn't exist in DB.", jiraBody.JiraId)
		return nil
	}

	if jiraMap[JiraProjectName].(string) != jiraBody.Project || jiraMap[JiraId].(string) != jiraBody.JiraId {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("The jira from DB (%s_%s) isn't the same as the specifed one (%s_%s).", jiraMap[JiraProjectName].(string), jiraMap[JiraId].(string), jiraBody.Project, jiraBody.JiraId)
	}
	jiraMap[JiraSummary] = jiraBody.Summary
	jiraMap[JiraDescription] = jiraBody.Description
	jiraMap[JiraStatus] = jiraBody.Status

	return updateJiraBodyInDB(jiraMap)
}

func MoveJiraIssue(jiraBody JiraIssue) *axerror.AXError {
	oldJiraId := jiraBody.OldJiraId
	oldJiraMap, axErr := GetJiraBodyByID(oldJiraId)
	if axErr != nil {
		utils.ErrorLog.Printf("Failed to get Jira Body from DB (Err: %v)", axErr)
		return axErr
	}

	// if the old jira wasn't created from our UI, just ignore it
	if oldJiraMap == nil {
		utils.ErrorLog.Printf("The old Jira to be moved doesn't exist in DB, just ignore it.")
		return nil
	}

	// change the association
	axErr = updateJiraAssociation(oldJiraMap, oldJiraId)
	if axErr != nil {
		return axErr
	}
	// insert the new jira to DB
	payload := map[string]interface{}{
		JiraId:          jiraBody.JiraId,
		JiraProjectName: jiraBody.Project,
		JiraDescription: jiraBody.Description,
		JiraStatus:      jiraBody.Status,
		JiraSummary:     jiraBody.Summary,
		JobList:         oldJiraMap[JobList],
		ApplicationList: oldJiraMap[ApplicationList],
		DeploymentList:  oldJiraMap[DeploymentList],
	}
	axErr = insertJiraBodyInDB(payload)
	if axErr != nil {
		utils.ErrorLog.Printf("Failed to insert new Jira (%s) to DB.", jiraBody.JiraId)
		return axErr
	}

	// get the list of services associated with old jira
	// we need to attach the new jira to these services
	srvList := oldJiraMap[JobList].([]interface{})
	for _, srvIdObj := range srvList {
		srvId := srvIdObj.(string)
		axErr = AttachJiraToService(srvId, jiraBody.JiraId)
		if axErr != nil {
			return axErr
		}
	}
	// finally remove the old jira from DB
	return DeleteJiraFromDBByID(oldJiraId)
}

func GetJiraBodyByID(jid string) (map[string]interface{}, *axerror.AXError) {
	params := map[string]interface{}{
		JiraId: jid,
	}

	resultArray := []map[string]interface{}{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, JiraBodyTable, params, &resultArray)

	if axErr != nil {
		return nil, axErr
	}

	if len(resultArray) > 1 {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("There should be at most one Jira for jira id %s, but we got %d.", jid, len(resultArray)))
	}

	if len(resultArray) == 0 {
		return nil, nil
	} else {
		return resultArray[0], nil
	}
}

func DeleteJiraFromDBByID(jid string) *axerror.AXError {
	params := []map[string]interface{}{{
		JiraId: jid,
	}}

	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, JiraBodyTable, params)
	if axErr != nil {
		utils.ErrorLog.Printf("Got error when deleting Jira (err %v)", axErr)
	}

	return axErr
}

func CreateJiraIssue(jiraBody JiraIssue) *axerror.AXError {
	jiraMap := createJiraMapFromJiraObject(jiraBody)
	_, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, JiraBodyTable, jiraMap)
	if axErr != nil {
		utils.ErrorLog.Printf("Got error when creating Jira (err %v)", axErr)
	}
	return axErr
}

func AttachServiceToJira(sid string, jiraMap map[string]interface{}) *axerror.AXError {
	serviceIds := jiraMap[JobList].([]interface{})
	for _, srvIdObj := range serviceIds {
		srvId := srvIdObj.(string)
		if srvId == sid {
			utils.InfoLog.Printf("The jira has already been associated with job (%s)", sid)
			return nil
		}
	}
	serviceIds = append(serviceIds, sid)
	jiraMap[JobList] = serviceIds
	return updateJiraBodyInDB(jiraMap)
}

func AttachApplicationToJira(aid string, jiraMap map[string]interface{}) *axerror.AXError {
	appIds := jiraMap[ApplicationList].([]interface{})
	for _, appIdObj := range appIds {
		appId := appIdObj.(string)
		if appId == aid {
			utils.InfoLog.Printf("The jira has already been associated with application (%s)", aid)
			return nil
		}
	}
	appIds = append(appIds, aid)
	jiraMap[ApplicationList] = appIds
	return updateJiraBodyInDB(jiraMap)
}

func AttachDeploymentToJira(did string, jiraMap map[string]interface{}) *axerror.AXError {
	deployIds := jiraMap[DeploymentList].([]interface{})
	for _, deployIdObj := range deployIds {
		deployId := deployIdObj.(string)
		if deployId == did {
			utils.InfoLog.Printf("The jira has already been associated with deployment (%s)", did)
			return nil
		}
	}
	deployIds = append(deployIds, did)
	jiraMap[DeploymentList] = deployIds
	return updateJiraBodyInDB(jiraMap)
}

func AttachJiraToService(sid string, jid string) *axerror.AXError {
	params := map[string]interface{}{axdb.AXDBUUIDColumnName: sid}
	resultArray, axErr := service.GetServiceMapsFromDB(params)
	if axErr != nil {
		return axErr
	} else if len(resultArray) < 1 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The service (%s) doesn't exist.", sid))
	}
	srv := resultArray[0]
	var jiraIds []interface{}
	if srv[service.ServiceJiraIssues] != nil {
		jiraIds = srv[service.ServiceJiraIssues].([]interface{})
	}
	for _, jiraIdObj := range jiraIds {
		jiraId := jiraIdObj.(string)
		if jid == jiraId {
			utils.InfoLog.Printf("The jira (%s) has already been associated with job (%s)", jiraId, sid)
			return nil
		}
	}

	jiraIds = append(jiraIds, jid)

	srv[service.ServiceJiraIssues] = jiraIds
	srv[axdb.AXDBConditionalUpdateExist] = ""
	delete(srv, axdb.AXDBWeekColumnName)
	service.ConvertFloatToInt64(srv, service.ServiceAverageRunTime)
	service.ConvertFloatToInt64(srv, service.ServiceRunTime)
	service.ConvertFloatToInt64(srv, service.ServiceAverageWaitTime)
	service.ConvertFloatToInt64(srv, service.ServiceWaitTime)
	service.ConvertFloatToInt64(srv, service.ServiceLaunchTime)
	service.ConvertFloatToInt64(srv, service.ServiceEndTime)
	service.ConvertFloatToInt64(srv, service.ServiceAverageInitTime)
	service.ConvertFloatToInt64(srv, axdb.AXDBTimeColumnName)
	service.ConvertFloatToInt(srv, service.ServiceStatus)

	_, axErr = utils.Dbcl.Put(axdb.AXDBAppAXOPS, service.RunningServiceTable, srv)
	if axErr != nil {
		if strings.Contains(axErr.Message, "Conditional Update failed, the row doesn't exist.") {
			//need to update to done table
			_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, service.DoneServiceTable, srv)
			if axErr != nil {
				utils.ErrorLog.Printf("Failed to update the jira issues for service (%v)", srv[service.ServiceTaskId])
				return axErr
			}
		} else {
			return axErr
		}

	}
	return nil
}

func AttachJiraToApplication(aid string, jid string) *axerror.AXError {
	params := map[string]interface{}{axdb.AXDBUUIDColumnName: aid}
	resultArray := []map[string]interface{}{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, application.ApplicationLatestTable, params, &resultArray)
	if axErr != nil {
		utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", application.ApplicationLatestTable, axErr)
		return axErr
	} else if len(resultArray) > 1 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
	} else if len(resultArray) == 0 {
		params = map[string]interface{}{axdb.AXDBUUIDColumnName: aid}
		axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, application.ApplicationHistoryTable, params, &resultArray)
		if axErr != nil {
			utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", application.ApplicationHistoryTable, axErr)
			return axErr
		} else if len(resultArray) > 1 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
		}
	}

	appMap := resultArray[0]
	var jiraIds []interface{}
	if appMap[application.ApplicationJiraIssues] != nil {
		jiraIds = appMap[application.ApplicationJiraIssues].([]interface{})
	}
	for _, jiraIdObj := range jiraIds {
		jiraId := jiraIdObj.(string)
		if jid == jiraId {
			utils.InfoLog.Printf("The jira (%s) has already been associated with application (%s)", jiraId, aid)
			return nil
		}
	}

	jiraIds = append(jiraIds, jid)

	appMap[application.ApplicationJiraIssues] = jiraIds
	appMap[axdb.AXDBConditionalUpdateExist] = ""
	delete(appMap, axdb.AXDBWeekColumnName)
	delete(appMap, axdb.AXDBTimeColumnName)
	service.ConvertFloatToInt64(appMap, application.ApplicationMtime)
	service.ConvertFloatToInt64(appMap, application.DeploymentsInit)
	service.ConvertFloatToInt64(appMap, application.DeploymentsWaiting)
	service.ConvertFloatToInt64(appMap, application.DeploymentsError)
	service.ConvertFloatToInt64(appMap, application.DeploymentsActive)
	service.ConvertFloatToInt64(appMap, application.DeploymentsTerminating)
	service.ConvertFloatToInt64(appMap, application.DeploymentsTerminated)
	service.ConvertFloatToInt64(appMap, application.DeploymentsStopping)
	service.ConvertFloatToInt(appMap, application.DeploymentsStopped)

	_, axErr = utils.Dbcl.Put(axdb.AXDBAppAMM, application.ApplicationLatestTable, appMap)
	if axErr != nil {
		if strings.Contains(axErr.Message, "Conditional Update failed, the row doesn't exist.") {
			//need to update to history application table
			_, axErr := utils.Dbcl.Put(axdb.AXDBAppAMM, application.ApplicationHistoryTable, appMap)
			if axErr != nil {
				utils.ErrorLog.Printf("Failed to update the jira issues for application (%v)", appMap[axdb.AXDBUUIDColumnName])
				return axErr
			}
		} else {
			return axErr
		}

	}
	return nil
}

func AttachJiraToDeployment(did string, jid string) *axerror.AXError {
	params := map[string]interface{}{axdb.AXDBUUIDColumnName: did}
	resultArray := []map[string]interface{}{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, deployment.DeploymentLatestTable, params, &resultArray)
	if axErr != nil {
		utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", deployment.DeploymentLatestTable, axErr)
		return axErr
	} else if len(resultArray) > 1 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
	} else if len(resultArray) == 0 {
		params = map[string]interface{}{axdb.AXDBUUIDColumnName: did}
		axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, deployment.DeploymentHistoryTable, params, &resultArray)
		if axErr != nil {
			utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", deployment.DeploymentHistoryTable, axErr)
			return axErr
		} else if len(resultArray) > 1 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
		}
	}

	deployMap := resultArray[0]
	var jiraIds []interface{}
	if deployMap[deployment.ServiceJiraIssues] != nil {
		jiraIds = deployMap[deployment.ServiceJiraIssues].([]interface{})
	}
	for _, jiraIdObj := range jiraIds {
		jiraId := jiraIdObj.(string)
		if jid == jiraId {
			utils.InfoLog.Printf("The jira (%s) has already been associated with deployment (%s)", jiraId, did)
			return nil
		}
	}

	jiraIds = append(jiraIds, jid)

	deployMap[deployment.ServiceJiraIssues] = jiraIds
	deployMap[axdb.AXDBConditionalUpdateExist] = ""
	delete(deployMap, axdb.AXDBWeekColumnName)
	delete(deployMap, axdb.AXDBTimeColumnName)
	service.ConvertFloatToInt64(deployMap, deployment.ServiceLaunchTime)
	service.ConvertFloatToInt64(deployMap, deployment.ServiceEndTime)
	service.ConvertFloatToInt64(deployMap, deployment.ServiceWaitTime)
	service.ConvertFloatToInt64(deployMap, deployment.ServiceAverageWaitTime)
	service.ConvertFloatToInt64(deployMap, deployment.ServiceRunTime)

	_, axErr = utils.Dbcl.Put(axdb.AXDBAppAMM, deployment.DeploymentLatestTable, deployMap)
	if axErr != nil {
		if strings.Contains(axErr.Message, "Conditional Update failed, the row doesn't exist.") {
			//need to update to history application table
			_, axErr := utils.Dbcl.Put(axdb.AXDBAppAMM, deployment.DeploymentHistoryTable, deployMap)
			if axErr != nil {
				utils.ErrorLog.Printf("Failed to update the jira issues for deployment (%v)", deployMap[axdb.AXDBUUIDColumnName])
				return axErr
			}
		} else {
			return axErr
		}

	}
	return nil
}

func updateJiraBodyInDB(jiraMap map[string]interface{}) *axerror.AXError {
	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, JiraBodyTable, jiraMap)
	return axErr
}

func insertJiraBodyInDB(jiraMap map[string]interface{}) *axerror.AXError {
	_, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, JiraBodyTable, jiraMap)
	return axErr
}

func createJiraMapFromJiraObject(jiraBody JiraIssue) map[string]interface{} {
	jiraMap := make(map[string]interface{})

	jiraMap[JiraId] = jiraBody.JiraId
	jiraMap[JiraProjectName] = jiraBody.Project
	jiraMap[JiraDescription] = jiraBody.Description
	jiraMap[JiraReport] = jiraBody.Reporter
	jiraMap[JiraSummary] = jiraBody.Summary
	jiraMap[JiraStatus] = jiraBody.Status

	return jiraMap
}

func updateJiraAssociation(jiraMap map[string]interface{}, jiraId string) *axerror.AXError {
	//delete the jira-number from service/application/deployment table
	if jiraMap[JobList] != nil {
		//TODO: this part of code should be refactoraized as an individual function
		serviceIds := jiraMap[JobList].([]interface{})
		for _, sidObj := range serviceIds {
			sid := sidObj.(string)
			srv, axErr := service.GetServiceMapByID(sid)
			if axErr != nil {
				utils.ErrorLog.Printf("Failed to get Service instance from DB (Err: %v)", axErr)
				return axErr
			}
			utils.InfoLog.Printf("update jira on service %v", srv)
			srvJiras := []string{}

			for _, jiraNumObj := range srv[service.ServiceJiraIssues].([]interface{}) {
				jiraNum := jiraNumObj.(string)
				if jiraNum != jiraId {
					srvJiras = append(srvJiras, jiraNum)
				}
			}

			//var index int = 0
			//for i, jiraNumObj := range srvJiras {
			//	jiraNum := jiraNumObj.(string)
			//	if jiraNum == jiraId {
			//		index = i
			//		break
			//	}
			//}
			//srvJiras = append(srvJiras[:index], srvJiras[index+1:]...)

			srv[service.ServiceJiraIssues] = srvJiras
			srv[axdb.AXDBConditionalUpdateExist] = ""
			delete(srv, axdb.AXDBWeekColumnName)
			service.ConvertFloatToInt64(srv, service.ServiceAverageRunTime)
			service.ConvertFloatToInt64(srv, service.ServiceRunTime)
			service.ConvertFloatToInt64(srv, service.ServiceAverageWaitTime)
			service.ConvertFloatToInt64(srv, service.ServiceWaitTime)
			service.ConvertFloatToInt64(srv, service.ServiceLaunchTime)
			service.ConvertFloatToInt64(srv, service.ServiceEndTime)
			service.ConvertFloatToInt64(srv, service.ServiceAverageInitTime)
			service.ConvertFloatToInt64(srv, axdb.AXDBTimeColumnName)
			service.ConvertFloatToInt(srv, service.ServiceStatus)

			_, axErr = utils.Dbcl.Put(axdb.AXDBAppAXOPS, service.RunningServiceTable, srv)
			if axErr != nil {
				if strings.Contains(axErr.Message, "Conditional Update failed, the row doesn't exist.") {
					//need to update to done table
					_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, service.DoneServiceTable, srv)
					if axErr != nil {
						utils.ErrorLog.Printf("Failed to update the jira issues for service (%v)", srv[service.ServiceTaskId])
						return axErr
					}
				} else {
					return axErr
				}

			}
		}
	}

	if jiraMap[ApplicationList] != nil {
		appIds := jiraMap[ApplicationList].([]interface{})
		for _, appIdObj := range appIds {
			//TODO: this part of code should be refactoraized as an individual function
			appId := appIdObj.(string)
			params := map[string]interface{}{axdb.AXDBUUIDColumnName: appId}
			resultArray := []map[string]interface{}{}
			axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, application.ApplicationLatestTable, params, &resultArray)
			if axErr != nil {
				utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", application.ApplicationLatestTable, axErr)
				return axErr
			} else if len(resultArray) > 1 {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
			} else if len(resultArray) == 0 {
				axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, application.ApplicationHistoryTable, params, &resultArray)
				if axErr != nil {
					utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", application.ApplicationHistoryTable, axErr)
					return axErr
				} else if len(resultArray) > 1 {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
				}
			}

			appMap := resultArray[0]

			utils.InfoLog.Printf("update jira on application %v", appMap)
			appJiras := []string{}
			for _, jiraNumObj := range appMap[application.ApplicationJiraIssues].([]interface{}) {
				jiraNum := jiraNumObj.(string)
				if jiraNum != jiraId {
					appJiras = append(appJiras, jiraNum)
				}
			}
			//var index int = 0
			//for i, jiraNumObj := range appJiras {
			//	jiraNum := jiraNumObj.(string)
			//	if jiraNum == jiraId {
			//		index = i
			//		break
			//	}
			//}
			//appJiras = append(appJiras[:index], appJiras[index+1:]...)

			appMap[application.ApplicationJiraIssues] = appJiras
			appMap[axdb.AXDBConditionalUpdateExist] = ""
			delete(appMap, axdb.AXDBWeekColumnName)
			delete(appMap, axdb.AXDBTimeColumnName)
			service.ConvertFloatToInt64(appMap, application.ApplicationMtime)
			service.ConvertFloatToInt64(appMap, application.DeploymentsInit)
			service.ConvertFloatToInt64(appMap, application.DeploymentsWaiting)
			service.ConvertFloatToInt64(appMap, application.DeploymentsError)
			service.ConvertFloatToInt64(appMap, application.DeploymentsActive)
			service.ConvertFloatToInt64(appMap, application.DeploymentsTerminating)
			service.ConvertFloatToInt64(appMap, application.DeploymentsTerminated)
			service.ConvertFloatToInt64(appMap, application.DeploymentsStopping)
			service.ConvertFloatToInt(appMap, application.DeploymentsStopped)

			_, axErr = utils.Dbcl.Put(axdb.AXDBAppAMM, application.ApplicationLatestTable, appMap)
			if axErr != nil {
				if strings.Contains(axErr.Message, "Conditional Update failed, the row doesn't exist.") {
					//need to update to history application table
					_, axErr := utils.Dbcl.Put(axdb.AXDBAppAMM, application.ApplicationHistoryTable, appMap)
					if axErr != nil {
						utils.ErrorLog.Printf("Failed to update the jira issues for application (%v)", appMap[axdb.AXDBUUIDColumnName])
						return axErr
					}
				} else {
					return axErr
				}

			}

		}
	}

	if jiraMap[DeploymentList] != nil {
		deployIds := jiraMap[DeploymentList].([]interface{})
		for _, deployIdObj := range deployIds {
			deployId := deployIdObj.(string)
			//TODO: this part of code should be refactoraized as an individual function
			params := map[string]interface{}{axdb.AXDBUUIDColumnName: deployId}
			resultArray := []map[string]interface{}{}
			axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, deployment.DeploymentLatestTable, params, &resultArray)
			if axErr != nil {
				utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", deployment.DeploymentLatestTable, axErr)
				return axErr
			} else if len(resultArray) > 1 {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
			} else if len(resultArray) == 0 {
				axErr := utils.Dbcl.Get(axdb.AXDBAppAMM, deployment.DeploymentHistoryTable, params, &resultArray)
				if axErr != nil {
					utils.ErrorLog.Printf("Got error when retrieving table (%s), (err: %v)", deployment.DeploymentHistoryTable, axErr)
					return axErr
				} else if len(resultArray) > 1 {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Expect at most one record, but got %d.", len(resultArray)))
				}
			}

			deployMap := resultArray[0]

			utils.InfoLog.Printf("update jira on deployment %v", deployMap)
			deployJiras := []string{}
			for _, jiraNumObj := range deployMap[deployment.ServiceJiraIssues].([]interface{}) {
				jiraNum := jiraNumObj.(string)
				if jiraNum != jiraId {
					deployJiras = append(deployJiras, jiraNum)
				}
			}
			//var index int = 0
			//for i, jiraNumObj := range deployJiras {
			//	jiraNum := jiraNumObj.(string)
			//	if jiraNum == jiraId {
			//		index = i
			//		break
			//	}
			//}
			//deployJiras = append(deployJiras[:index], deployJiras[index+1:]...)

			deployMap[deployment.ServiceJiraIssues] = deployJiras
			deployMap[axdb.AXDBConditionalUpdateExist] = ""
			delete(deployMap, axdb.AXDBWeekColumnName)
			delete(deployMap, axdb.AXDBTimeColumnName)
			service.ConvertFloatToInt64(deployMap, deployment.ServiceLaunchTime)
			service.ConvertFloatToInt64(deployMap, deployment.ServiceEndTime)
			service.ConvertFloatToInt64(deployMap, deployment.ServiceWaitTime)
			service.ConvertFloatToInt64(deployMap, deployment.ServiceAverageWaitTime)
			service.ConvertFloatToInt64(deployMap, deployment.ServiceRunTime)

			_, axErr = utils.Dbcl.Put(axdb.AXDBAppAMM, deployment.DeploymentLatestTable, deployMap)
			if axErr != nil {
				if strings.Contains(axErr.Message, "Conditional Update failed, the row doesn't exist.") {
					//need to update to history application table
					_, axErr := utils.Dbcl.Put(axdb.AXDBAppAMM, deployment.DeploymentHistoryTable, deployMap)
					if axErr != nil {
						utils.ErrorLog.Printf("Failed to update the jira issues for deployment (%v)", deployMap[axdb.AXDBUUIDColumnName])
						return axErr
					}
				} else {
					return axErr
				}

			}
		}
	}

	return nil
}
