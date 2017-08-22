package jira

import "applatix.io/axdb"

// table schema
const (
	JiraBodyTable   = "jira_body"
	JiraProjectName = "project"
	JiraId          = "id"
	JiraSummary     = "summary"
	JiraDescription = "description"
	JiraStatus      = "status"
	JiraReport      = "reporter"
	JobList         = "job_list"
	ApplicationList = "app_list"
	DeploymentList  = "deploy_list"
)

var JiraSchema = axdb.Table{AppName: axdb.AXDBAppAXOPS, Name: JiraBodyTable, Type: axdb.TableTypeKeyValue, Columns: map[string]axdb.Column{
	JiraId:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
	JiraProjectName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	JiraSummary:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	JiraDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	JiraStatus:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	JiraReport:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	JobList:         axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
	ApplicationList: axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
	DeploymentList:  axdb.Column{Type: axdb.ColumnTypeArray, Index: axdb.ColumnIndexNone},
},

	UseSearch: true,
}
