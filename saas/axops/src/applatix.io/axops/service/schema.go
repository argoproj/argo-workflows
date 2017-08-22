package service

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/index"
	"applatix.io/axops/utils"
)

const (
	TemplateTable         = "template"
	TemplateId            = "id"
	TemplateName          = "name"
	TemplateDescription   = "description"
	TemplateType          = "type"
	TemplateVersion       = "version"
	TemplateRepo          = "repo"
	TemplateBranch        = "branch"
	TemplateRevision      = "revision"
	TemplateBody          = "body"
	TemplateUserName      = "username"
	TemplateUserId        = "user_id"
	TemplateEnvParameters = "env_params"
	TemplateCost          = "cost"
	TemplateLabels        = "labels"
	TemplateAnnotations   = "annotations"
	TemplateRepoBranch    = "repo_branch"

	//TemplateJobsInit    = "jobs_init"
	//TemplateJobsWait    = "jobs_wait"
	//TemplateJobsRun     = "jobs_run"
	TemplateJobsFail    = "jobs_fail"
	TemplateJobsSuccess = "jobs_success"
)

// Note: we store the parameters that the template takes in the templateEnvParameters column. We will need to search against
// this to find out if the template is compatible with certain UI operations
var TemplateSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    TemplateTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		TemplateId:            axdb.Column{Type: axdb.ColumnTypeUUID, Index: axdb.ColumnIndexPartition},
		TemplateName:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		TemplateDescription:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateType:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateVersion:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateRepo:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateBranch:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		TemplateRevision:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateCost:          axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		TemplateBody:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateUserName:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateUserId:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		TemplateEnvParameters: axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong},
		TemplateLabels:        axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		TemplateAnnotations:   axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		TemplateRepoBranch:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},

		//TemplateJobsInit:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		//TemplateJobsWait:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		//TemplateJobsRun:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		TemplateJobsFail:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		TemplateJobsSuccess: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
	ExcludedIndexColumns: map[string]bool{
		TemplateBody: true,
	},
}

// table schema
const (
	RunningServiceTable = "live_service"
	DoneServiceTable    = "done_service"

	ServiceTemplateName      = "template_name"
	ServiceTemplateId        = "template_id"
	ServiceTemplateStr       = "template"
	ServiceArguments         = "arguments"
	ServiceFlags             = "flags"
	ServiceStatus            = "status"
	ServiceName              = "name"
	ServiceDescription       = "description"
	ServiceMem               = "mem"
	ServiceCPU               = "cpu"
	ServiceCostId            = "cost_id"
	ServiceUserName          = "username"
	ServiceUserId            = "user_id"
	ServiceHostName          = "hostname"
	ServiceHostId            = "host_id"
	ServiceContainerId       = "container_id"
	ServiceContainerName     = "container_name"
	ServiceCost              = "cost"
	ServiceTaskId            = "task_id"
	ServiceIsTask            = "is_task"
	ServiceParentId          = "parent_id"
	ServiceLaunchTime        = "launch_time"
	ServiceAverageInitTime   = "average_init_time"
	ServiceEndTime           = "end_time"
	ServiceWaitTime          = "wait_time"
	ServiceAverageWaitTime   = "avg_wait_time"
	ServiceRunTime           = "run_time"
	ServiceAverageRunTime    = "avg_run_time"
	ServiceNotifications     = "notifications"
	ServiceCommit            = "commit"
	ServicePolicyId          = "policy_id"
	ServiceLogLive           = "url_run"
	ServiceLogDone           = "url_done"
	ServiceLabels            = "labels"
	ServiceAnnotations       = "annotations"
	ServiceStatusDetail      = "status_detail"
	ServiceEndpoint          = "endpoint"
	ServiceRepo              = "repo"
	ServiceBranch            = "branch"
	ServiceRevision          = "revision"
	ServiceRepoBranch        = "repo_branch"
	ServiceFailurePath       = "failure_path"
	ServiceArtifactTags      = "tags"
	ServiceTerminationPolicy = "termination_policy"
	ServiceIsSubmitted       = "is_submitted"
	ServiceStatusString      = "status_string"
	ServiceJiraIssues        = "jira_issues"
	ServiceFixtures          = "fixtures"
)

var ServiceSchema = axdb.Table{AppName: axdb.AXDBAppAXOPS, Name: "", Type: axdb.TableTypeTimeSeries, Columns: map[string]axdb.Column{
	ServiceTemplateName:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
	ServiceTemplateId:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	ServiceTemplateStr:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceArguments:         axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong},
	ServiceFlags:             axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	ServiceStatus:            axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceName:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceDescription:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceMem:               axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	ServiceCPU:               axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	ServiceCostId:            axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	ServiceUserName:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceUserId:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	ServiceContainerId:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceContainerName:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceHostName:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceHostId:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	ServiceCost:              axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	ServiceTaskId:            axdb.Column{Type: axdb.ColumnTypeUUID, Index: axdb.ColumnIndexStrong},
	ServiceIsTask:            axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
	ServiceParentId:          axdb.Column{Type: axdb.ColumnTypeUUID, Index: axdb.ColumnIndexStrong},
	ServiceLaunchTime:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceAverageInitTime:   axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceEndTime:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceWaitTime:          axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceAverageWaitTime:   axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceRunTime:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceAverageRunTime:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	ServiceNotifications:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceCommit:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServicePolicyId:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceLogLive:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceLogDone:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceLabels:            axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	ServiceAnnotations:       axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	ServiceStatusDetail:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceEndpoint:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceRepo:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceBranch:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceRepoBranch:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	ServiceRevision:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	ServiceFailurePath:       axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
	ServiceArtifactTags:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceTerminationPolicy: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceIsSubmitted:       axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
	ServiceStatusString:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	ServiceJiraIssues:        axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
	ServiceFixtures:          axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexStrong, IndexFlagForMapColumn: axdb.ColumnIndexMapKeys},
},

	Stats: map[string]int{
		ServiceWaitTime: axdb.ColumnStatPercent,
		ServiceRunTime:  axdb.ColumnStatSum,
	},

	UseSearch: true,
	ExcludedIndexColumns: map[string]bool{
		ServiceTemplateStr:       true,
		ServiceTerminationPolicy: true,
		ServiceNotifications:     true,
	},
}

func GetDoneServiceSchema() axdb.Table {
	table := ServiceSchema
	table.Name = DoneServiceTable
	return table
}

func GetRunningServiceSchema() axdb.Table {
	table := ServiceSchema
	table.Name = RunningServiceTable
	return table
}

var serviceIndexKeyList []string = []string{
	ServiceName,
	ServiceDescription,
	ServiceStatusString,
	ServiceRepo,
	ServiceBranch,
	ServiceUserName,
}

func updateServiceDB(currentTable string, serviceMap map[string]interface{}) (map[string]interface{}, *axerror.AXError) {
	if serviceMap[ServiceIsTask] != nil && serviceMap[ServiceIsTask].(bool) {
		for _, key := range serviceIndexKeyList {
			if _, ok := serviceMap[key]; ok {
				index.SendToSearchIndexChan("services", key, serviceMap[key].(string))
			}
		}
	}
	return utils.Dbcl.Put(axdb.AXDBAppAXOPS, currentTable, serviceMap)
}
