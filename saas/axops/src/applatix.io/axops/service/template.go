package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/index"
	"applatix.io/axops/utils"
	"applatix.io/template"
)

type ServiceMap map[string]*Service

// Returns the template's parameter name that has the default value matching the value passed in
func FindParameterNameWithDefault(inputs *template.Inputs, value string) *string {
	if inputs != nil {
		for name, param := range inputs.Parameters {
			if param != nil && param.Default != nil && *param.Default == value {
				argName := fmt.Sprintf("parameters.%s", name)
				return &argName
			}
		}
	}
	return nil
}

func GetTemplates(params map[string]interface{}) ([]EmbeddedTemplateIf, *axerror.AXError) {

	if params != nil && params[axdb.AXDBSelectColumns] != nil {
		fields := params[axdb.AXDBSelectColumns].([]string)
		fields = append(fields, TemplateType)
		fields = utils.DedupStringList(fields)
		params[axdb.AXDBSelectColumns] = fields
	}

	resultArray := []map[string]interface{}{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, TemplateTable, params, &resultArray)
	if axErr != nil {
		return nil, axErr
	}

	tempArray := make([]EmbeddedTemplateIf, 0)
	for _, m := range resultArray {
		t, axErr := MapToTemplate(m)
		if axErr == nil && t != nil {
			tempArray = append(tempArray, t)
		} else {
			utils.ErrorLog.Printf("Failed to deserialize template %s: %v", m[TemplateId], axErr)
		}
	}
	return tempArray, nil
}

func GetTemplateById(id string) (EmbeddedTemplateIf, *axerror.AXError) {
	params := map[string]interface{}{TemplateId: id}
	templateArray, axErr := GetTemplates(params)
	if axErr != nil {
		return nil, axErr
	}
	if len(templateArray) > 0 {
		return templateArray[0], nil
	}
	return nil, nil
}

func DeleteTemplateById(id string) *axerror.AXError {
	params := []map[string]interface{}{
		map[string]interface{}{TemplateId: id},
	}
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, TemplateTable, params)
	return axErr
}

var templateIndexKeyList []string = []string{
	TemplateName,
	TemplateDescription,
	TemplateRepo,
	TemplateBranch,
	TemplateType,
}

// convert to format suitable to AXDB
func TemplateToMap(tmpl EmbeddedTemplateIf) map[string]interface{} {
	tempMap := make(map[string]interface{})
	tempMap[TemplateId] = tmpl.GetID()
	tempMap[TemplateName] = tmpl.GetName()
	tempMap[TemplateDescription] = tmpl.GetDescription()
	tempMap[TemplateType] = tmpl.GetType()
	tempMap[TemplateVersion] = tmpl.GetVersion()
	tempMap[TemplateRepo] = tmpl.GetRepo()
	tempMap[TemplateBranch] = tmpl.GetBranch()
	tempMap[TemplateRevision] = tmpl.GetRevision()

	if tempMap[TemplateId] == "" {
		tempMap[TemplateId] = utils.GenerateUUIDv5(fmt.Sprintf("%s:%s:%s", tmpl.GetRepo(), tmpl.GetBranch(), tmpl.GetName()))
	}

	if tmpl.GetRepo() != "" && tmpl.GetBranch() != "" {
		tempMap[TemplateRepoBranch] = tmpl.GetRepo() + "_" + tmpl.GetBranch()
	} else {
		tempMap[TemplateRepoBranch] = ""
	}

	templateBody, err := json.Marshal(tmpl)
	if err != nil {
		panic(err)
	}
	tempMap[TemplateBody] = string(templateBody)
	labels := tmpl.GetLabels()
	if labels != nil {
		tempMap[TemplateLabels] = labels
	} else {
		tempMap[TemplateLabels] = map[string]string{}
	}

	// TODO: revisit. do templates use annotations? -Jesse
	// if template.Annotations != nil {
	// 	tempMap[TemplateAnnotations] = template.Annotations
	// } else {
	// 	tempMap[TemplateAnnotations] = map[string]string{}
	// }
	tempMap[TemplateAnnotations] = map[string]string{}

	// TODO: revisit -Jesse
	// if u != nil {
	// 	tempMap[TemplateUserName] = u.Username
	// 	tempMap[TemplateUserId] = u.ID
	// }

	// This builds the env_params column, which allows us to query on templates that want a certain type of input
	// The primary use case is for querying templates which accept a %%session.commit%%, %%session.repo%%.
	inputs := tmpl.GetInputs()
	argNames := []string{}
	if inputs != nil {
		for _, input := range inputs.Parameters {
			if input != nil && input.Default != nil && template.IsParam(*input.Default) {
				parts := strings.Split(strings.Trim(*input.Default, "%"), ".")
				if len(parts) >= 2 {
					argNames = append(argNames, parts[1])
				}
			}
		}
	}
	tempMap[TemplateEnvParameters] = argNames

	//this is a workaround for cassandra bug in case where env_params column is null.
	if tempMap[TemplateEnvParameters] == nil || len(tempMap[TemplateEnvParameters].([]string)) == 0 {
		tempMap[TemplateEnvParameters] = []string{"none"}
	}

	for _, key := range templateIndexKeyList {
		if _, ok := tempMap[key]; ok {
			index.SendToSearchIndexChan("templates", key, tempMap[key].(string))
		}
	}

	return tempMap
}

// convert the map from AXDB to template.
func MapToTemplate(tempMap map[string]interface{}) (EmbeddedTemplateIf, *axerror.AXError) {
	tStats := TemplateStats{}
	if tempMap[TemplateCost] != nil {
		cost := tempMap[TemplateCost].(float64)
		tStats.Cost = &cost
	}
	if tempMap[TemplateJobsFail] != nil {
		failed := int64(tempMap[TemplateJobsFail].(float64))
		tStats.JobsFail = &failed
	}
	if tempMap[TemplateJobsSuccess] != nil {
		success := int64(tempMap[TemplateJobsSuccess].(float64))
		tStats.JobsSuccess = &success
	}

	if tempMap[TemplateBody] != nil {
		// If template body was returned as part of the query, we simply need to unmarshal it
		tmpl, axErr := UnmarshalEmbeddedTemplate(([]byte)(tempMap[TemplateBody].(string)))
		if axErr != nil {
			return nil, axErr
		}
		switch tmpl.GetType() {
		case template.TemplateTypeContainer:
			ct := tmpl.(*EmbeddedContainerTemplate)
			ct.TemplateStats = tStats
		case template.TemplateTypeWorkflow:
			wt := tmpl.(*EmbeddedWorkflowTemplate)
			wt.TemplateStats = tStats
		case template.TemplateTypeDeployment:
			dt := tmpl.(*EmbeddedDeploymentTemplate)
			dt.TemplateStats = tStats
		}
		return tmpl, nil
	}

	// The following supports the case where query was made against a subset of columns
	base := template.BaseTemplate{}
	if tempMap[TemplateType] != nil {
		base.Type = tempMap[TemplateType].(string)
	}
	if tempMap[TemplateId] != nil {
		base.ID = tempMap[TemplateId].(string)
	}
	if tempMap[TemplateName] != nil {
		base.Name = tempMap[TemplateName].(string)
	}
	if tempMap[TemplateDescription] != nil {
		base.Description = tempMap[TemplateDescription].(string)
	}
	if tempMap[TemplateRepo] != nil {
		base.Repo = tempMap[TemplateRepo].(string)
	}
	if tempMap[TemplateBranch] != nil {
		base.Branch = tempMap[TemplateBranch].(string)
	}
	if tempMap[TemplateRevision] != nil {
		base.Revision = tempMap[TemplateRevision].(string)
	}
	switch base.Type {
	case template.TemplateTypeContainer:
		ct := EmbeddedContainerTemplate{
			ContainerTemplate: &template.ContainerTemplate{
				BaseTemplate: base,
			},
			TemplateStats: tStats,
		}
		return &ct, nil
	case template.TemplateTypeWorkflow:
		wt := EmbeddedWorkflowTemplate{
			WorkflowTemplate: &template.WorkflowTemplate{
				BaseTemplate: base,
			},
			TemplateStats: tStats,
		}
		return &wt, nil
	case template.TemplateTypeDeployment:
		dt := EmbeddedDeploymentTemplate{
			DeploymentTemplate: &template.DeploymentTemplate{
				BaseTemplate: base,
			},
			TemplateStats: tStats,
		}
		return &dt, nil
	default:
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Unknown template type: %s", base.Type)
	}
}

func UpdateTemplateCost(templateId string, cost float64) {
	utils.DebugLog.Printf("UpdateTemplate %s cost to %f", TemplateId, cost)
	// we use a simple algorithm to get an approximate. We weight the past history at 90% and the current
	// data point at 10%.
	params := map[string]interface{}{
		TemplateId:             templateId,
		axdb.AXDBSelectColumns: []string{TemplateId, TemplateCost, TemplateName, TemplateRepo, TemplateBranch},
	}
	resultArray := []map[string]interface{}{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, TemplateTable, params, &resultArray)
	if axErr != nil || len(resultArray) == 0 {
		// This can happen if the template is deleted while a service is still running
		return
	}

	temp := resultArray[0]
	if c, ok := temp[TemplateCost].(float64); ok && c > 0 {
		cost = 0.9*c + 0.1*cost
	}
	temp[TemplateCost] = cost
	utils.DebugLog.Printf("updating template %v", temp)
	utils.Dbcl.Put(axdb.AXDBAppAXOPS, TemplateTable, temp)

	UpdateTemplateETag()
}

func UpdateTemplateJobCounts(templateId string, oldStatus, newStatus int) *axerror.AXError {

	params := map[string]interface{}{
		TemplateId: templateId,
		//axdb.AXDBSelectColumns: []string{TemplateId, TemplateName, TemplateRepo, TemplateBranch, TemplateJobsInit, TemplateJobsWait, TemplateJobsRun, TemplateJobsFail, TemplateJobsSuccess},
		axdb.AXDBSelectColumns: []string{TemplateId, TemplateName, TemplateRepo, TemplateBranch, TemplateJobsFail, TemplateJobsSuccess},
	}
	resultArray := []map[string]interface{}{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, TemplateTable, params, &resultArray)
	if axErr != nil || len(resultArray) == 0 {
		// This can happen if the template is deleted while a service is still running
		return nil
	}

	tempMap := resultArray[0]

	//convertFloatToInt64(tempMap, TemplateJobsInit)
	//convertFloatToInt64(tempMap, TemplateJobsWait)
	//convertFloatToInt64(tempMap, TemplateJobsRun)
	ConvertFloatToInt64(tempMap, TemplateJobsFail)
	ConvertFloatToInt64(tempMap, TemplateJobsSuccess)

	//decVal := func(key string) {
	//	var val int64 = 0
	//	if tempMap[key] != nil {
	//		val = tempMap[key].(int64)
	//	}
	//
	//	if val > 0 {
	//		val = val - 1
	//	}
	//
	//	tempMap[key] = val
	//}

	incVal := func(key string) {
		var val int64 = 0
		if tempMap[key] != nil {
			val = tempMap[key].(int64)
		}

		tempMap[key] = val + 1
	}

	//switch oldStatus {
	//case utils.ServiceStatusInitiating:
	//	decVal(TemplateJobsInit)
	//case utils.ServiceStatusWaiting:
	//	decVal(TemplateJobsWait)
	//case utils.ServiceStatusRunning, utils.ServiceStatusCanceling:
	//	decVal(TemplateJobsRun)
	//case utils.ServiceStatusCancelled, utils.ServiceStatusFailed:
	//	decVal(TemplateJobsFail)
	//case utils.ServiceStatusSuccess:
	//	decVal(TemplateJobsSuccess)
	//}

	switch newStatus {
	//case utils.ServiceStatusInitiating:
	//	incVal(TemplateJobsInit)
	//case utils.ServiceStatusWaiting:
	//	incVal(TemplateJobsWait)
	//case utils.ServiceStatusRunning, utils.ServiceStatusCanceling:
	//	incVal(TemplateJobsRun)
	case utils.ServiceStatusCancelled, utils.ServiceStatusFailed:
		incVal(TemplateJobsFail)
	case utils.ServiceStatusSuccess:
		incVal(TemplateJobsSuccess)
	}

	utils.DebugLog.Printf("updating template %v", tempMap)
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, TemplateTable, tempMap); axErr != nil {
		return axErr
	}

	UpdateTemplateETag()

	return nil
}
