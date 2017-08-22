// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package fixture

import "applatix.io/axdb"

const (
	TemplateTableName   = "fixture_templates"
	TemplateID          = "id"
	TemplateName        = "name"
	TemplateDescription = "description"
	TemplateAttributes  = "attributes"
	TemplateActions     = "actions"
	TemplateRepo        = "repo"
	TemplateBranch      = "branch"
	TemplateRevision    = "revision"
	TemplateRepoBranch  = "repo_branch"
)

var FixtureTemplateSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    TemplateTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		TemplateID:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		TemplateName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		TemplateDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateAttributes:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateRepo:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		TemplateBranch:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		TemplateRevision:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateActions:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		TemplateRepoBranch:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	},
	UseSearch: true,
}

const (
	ClassTableName       = "fixture_classes"
	ClassID              = "id"
	ClassName            = "name"
	ClassDescription     = "description"
	ClassAttributes      = "attributes"
	ClassActions         = "actions"
	ClassRepo            = "repo"
	ClassBranch          = "branch"
	ClassRevision        = "revision"
	ClassActionTemplates = "action_templates"
	ClassStatus          = "status"
	ClassStatusDetail    = "status_detail"
)

var FixtureClassSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    ClassTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ClassID:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ClassName:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		ClassDescription:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassAttributes:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassActions:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassRepo:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassBranch:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassRevision:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassActionTemplates: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassStatus:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClassStatusDetail:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}

const (
	InstanceTableName     = "fixture_instances"
	InstanceID            = "id"
	InstanceName          = "name"
	InstanceDescription   = "description"
	InstanceClassName     = "class_name"
	InstanceClassID       = "class_id"
	InstanceEnabled       = "enabled"
	InstanceDisableReason = "disable_reason"
	InstanceCreator       = "creator"
	InstanceOwner         = "owner"
	InstanceStatus        = "status"
	InstanceStatusDetail  = "status_detail"
	InstanceConcurrency   = "concurrency"
	InstanceReferrers     = "referrers"
	InstanceOperation     = "operation"
	InstanceAttributes    = "attributes"
	InstanceCtime         = "ctime"
	InstanceMtime         = "mtime"
	InstanceAtime         = "atime"
)

var FixtureInstanceSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    InstanceTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		InstanceID:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		InstanceName:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		InstanceDescription:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceClassName:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceClassID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceEnabled:       axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
		InstanceDisableReason: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceCreator:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceOwner:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceStatus:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceStatusDetail:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceConcurrency:   axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		InstanceReferrers:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceOperation:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceAttributes:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InstanceCtime:         axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		InstanceMtime:         axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		InstanceAtime:         axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}

const (
	RequestTableName             = "fixture_requests"
	RequestServiceID             = "service_id"
	RequestAssigned              = "assigned"
	RequestRequester             = "requester"
	RequestUser                  = "user"
	RequestRootWorkflowID        = "root_workflow_id"
	RequestApplicationID         = "application_id"
	RequestApplicationName       = "application_name"
	RequestApplicationGeneration = "application_generation"
	RequestDeploymentName        = "deployment_name"
	RequestTime                  = "request_time"
	RequestRequirements          = "requirements"
	RequestVolRequirements       = "vol_requirements"
	RequestAssignmentTime        = "assignment_time"
	RequestAssignment            = "assignment"
	RequestVolAssignment         = "vol_assignment"
)

var FixtureRequestSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    RequestTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		RequestServiceID:             axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		RequestAssigned:              axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
		RequestRequester:             axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestUser:                  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestRootWorkflowID:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestApplicationID:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestApplicationName:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestApplicationGeneration: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestDeploymentName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestTime:                  axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		RequestRequirements:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestVolRequirements:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestAssignmentTime:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		RequestAssignment:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RequestVolAssignment:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}
