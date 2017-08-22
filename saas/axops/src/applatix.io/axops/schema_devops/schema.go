package schema_devops

import "applatix.io/axdb"

const (
	WorkflowTable           = "workflow"
	WorkflowId              = "id"
	WorkflowStatus          = "status"
	WorkflowServiceTemplate = "service_template"
	WorkflowTimestamp       = "timestamp"
	WorkflowResource        = "resource"

	WorkflowLeafServiceTable     = "workflow_leaf_service"
	WorkflowLeafServiceLeafId    = "leaf_id"
	WorkflowLeafServiceRootId    = "root_id"
	WorkflowLeafServiceSN        = "sn"
	WorkflowLeafServiceResult    = "result"
	WorkflowLeafServiceDetail    = "detail"
	WorkflowLeafServiceTimestamp = "timestamp"

	WorkflowNodeEventTable        = "workflow_node_event"
	WorkflowNodeEventLeafId       = "leaf_id"
	WorkflowNodeEventRootId       = "root_id"
	WorkflowNodeEventResult       = "result"
	WorkflowNodeEventDetail       = "detail"
	WorkflowNodeEventStatusDetail = "status_detail"
	WorkflowNodeEventTimestamp    = "timestamp"

	WorkflowTimedEventsTable     = "workflow_timed_events"
	WorkflowTimedEventsRootId    = "root_id"
	WorkflowTimedEventsTimestamp = "timestamp"
	WorkflowTimedEventsEventType = "event_type"
	WorkflowTimedEventsDetail    = "detail"

	WorkflowKVTable     = "workflow_kv"
	WorkflowKVKey       = "key"
	WorkflowKVValue     = "value"
	WorkflowKVTimestamp = "timestamp"

	BranchTable  = "branch"
	BranchRepo   = "repo"
	BranchBranch = "branch"
	BranchHead   = "head"

	ApprovalTable          = "approval"
	ApprovalRootId         = "root_id"
	ApprovalLeafId         = "leaf_id"
	ApprovalRequiredList   = "required_list"
	ApprovalOptionalList   = "optional_list"
	ApprovalOptionalNumber = "optional_number"
	ApprovalTimeout        = "timeout"
	ApprovalFinalResult    = "result"
	ApprovalResultDetail   = "detail"

	ApprovalResultTable     = "approval_result"
	ApprovalResultRootId    = "root_id"
	ApprovalResultLeafId    = "leaf_id"
	ApprovalResultUser      = "user"
	ApprovalResultResult    = "result"
	ApprovalResultTimestamp = "timestamp"

	JunitTestCaseResultTable      = "junit_result"
	JunitTestCaseResultId         = "result_id"
	JunitTestCaseResultLeafId     = "leaf_id"
	JunitTestCaseResultName       = "name"
	JunitTestCaseResultStatus     = "status"
	JunitTestCaseResultMessage    = "message"
	JunitTestCaseResultClassname  = "classname"
	JunitTestCaseResultStderr     = "stderr"
	JunitTestCaseResultStdout     = "stdout"
	JunitTestCaseResultTestsuite  = "testsuite"
	JunitTestCaseResultTestsuites = "testsuites"
	JunitTestCaseResultDuration   = "duration"

	ResourceTable            = "resource"
	ResourceId               = "resource_id"
	ResourcePayload          = "resource"
	ResourceCategory         = "category"
	ResourceTTLSeconds       = "ttl"
	ResourceTimestampSeconds = "timestamp"
	ResourceDetail           = "detail"
)

var WorkflowSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    WorkflowTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		WorkflowId:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		WorkflowStatus:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		WorkflowServiceTemplate: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		WorkflowTimestamp:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		WorkflowResource:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
}

var WorkflowLeafServiceSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    WorkflowLeafServiceTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		WorkflowLeafServiceRootId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		WorkflowLeafServiceSN:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexClustering},
		WorkflowLeafServiceLeafId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		WorkflowLeafServiceResult:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		WorkflowLeafServiceDetail:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		WorkflowLeafServiceTimestamp: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}

var WorkflowTimedEventsSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    WorkflowTimedEventsTable,
	Type:    axdb.TableTypeTimedKeyValue,
	Columns: map[string]axdb.Column{
		WorkflowTimedEventsRootId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		WorkflowTimedEventsTimestamp: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		WorkflowTimedEventsEventType: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		WorkflowTimedEventsDetail:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
}

var WorkflowKVSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    WorkflowKVTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		WorkflowKVKey:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		WorkflowKVValue:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		WorkflowKVTimestamp: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}

var WorkflowNodeEventsSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    WorkflowNodeEventTable,
	Type:    axdb.TableTypeTimedKeyValue,
	Columns: map[string]axdb.Column{
		WorkflowNodeEventLeafId:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		WorkflowNodeEventRootId:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		WorkflowNodeEventResult:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		WorkflowNodeEventDetail:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		WorkflowNodeEventStatusDetail: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		WorkflowNodeEventTimestamp:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}

var BranchSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    BranchTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		BranchRepo:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		BranchBranch: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		BranchHead:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	},
}

var ApprovalSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    ApprovalTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ApprovalRootId:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ApprovalLeafId:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		ApprovalRequiredList:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApprovalOptionalList:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApprovalOptionalNumber: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ApprovalTimeout:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ApprovalFinalResult:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApprovalResultDetail:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
}

var ApprovalResultSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    ApprovalResultTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ApprovalResultLeafId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ApprovalResultUser:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		ApprovalResultRootId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApprovalResultResult:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ApprovalResultTimestamp: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}

var JunitTestCaseResultSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    JunitTestCaseResultTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		JunitTestCaseResultId:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		JunitTestCaseResultLeafId:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultName:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultStatus:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultMessage:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultClassname:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultStderr:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultStdout:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultTestsuite:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultTestsuites: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		JunitTestCaseResultDuration:   axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	},
}

var ResourceSchema = axdb.Table{
	AppName: axdb.AXDBAppAXDEVOPS,
	Name:    ResourceTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ResourceId:               axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ResourcePayload:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ResourceCategory:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ResourceTTLSeconds:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ResourceTimestampSeconds: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ResourceDetail:           axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
	},
}
