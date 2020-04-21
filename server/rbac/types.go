package rbac

import (
	"strings"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

func parseOp(fullMethod string) op {
	parts := strings.SplitN(fullMethod, "/", 3)
	return parts[2]
}

// an op is a act+obj (typically verb+noun, e.g. GetWorkflow)
type op = string
type act = string
type obj = string

//noinspection GoReservedWordUsedAsName
const (
	create act = "create"
	delete     = "delete"
	get        = "get"
	lint       = "lint"
	list       = "list"
	update     = "update"
	watch      = "watch"
)

func parseActObj(value op) (act, obj) {
	switch value {
	case "CreateClusterWorkflowTemplate":
		return create, workflow.ClusterWorkflowTemplatePlural
	case "CreateCronWorkflow":
		return create, workflow.CronWorkflowPlural
	case "CreateWorkflow":
		return create, workflow.WorkflowPlural
	case "CreateWorkflowTemplate":
		return create, workflow.WorkflowTemplatePlural
	case "DeleteArchivedWorkflow":
		return delete, workflow.WorkflowPlural
	case "DeleteClusterWorkflowTemplate":
		return delete, workflow.ClusterWorkflowTemplatePlural
	case "DeleteCronWorkflow":
		return delete, workflow.ClusterWorkflowTemplatePlural
	case "DeleteWorkflow":
		return delete, workflow.WorkflowPlural
	case "DeleteWorkflowTemplate":
		return delete, workflow.WorkflowTemplatePlural
	case "GetArchivedWorkflow":
		return get, workflow.WorkflowPlural
	case "GetClusterWorkflowTemplate":
		return get, workflow.ClusterWorkflowTemplatePlural
	case "GetCronWorkflow":
		return get, workflow.CronWorkflowPlural
	case "GetInfo":
		return get, workflow.WorkflowPlural
	case "GetWorkflow":
		return get, workflow.WorkflowPlural
	case "GetWorkflowTemplate":
		return get, workflow.WorkflowTemplatePlural
	case "LintClusterWorkflowTemplate":
		return lint, workflow.ClusterWorkflowTemplatePlural
	case "LintCronWorkflow":
		return lint, workflow.CronWorkflowPlural
	case "LintWorkflow":
		return lint, workflow.WorkflowPlural
	case "LintWorkflowTemplate":
		return list, workflow.WorkflowTemplatePlural
	case "ListArchivedWorkflows":
		return list, workflow.WorkflowPlural
	case "ListClusterWorkflowTemplates":
		return list, workflow.ClusterWorkflowTemplatePlural
	case "ListCronWorkflows":
		return list, workflow.CronWorkflowPlural
	case "ListWorkflowTemplates":
		return list, workflow.WorkflowTemplatePlural
	case "ListWorkflows":
		return list, workflow.WorkflowPlural
	case "PodLogs":
		return watch, workflow.WorkflowPlural
	case "ResubmitWorkflow":
		return create, workflow.WorkflowPlural
	case "ResumeWorkflow":
		return update, workflow.WorkflowPlural
	case "RetryWorkflow":
		return update, workflow.WorkflowPlural
	case "StopWorkflow":
		return update, workflow.WorkflowPlural
	case "SubmitFrom":
		return update, workflow.WorkflowPlural
	case "SuspendWorkflow":
		return update, workflow.WorkflowPlural
	case "TerminateWorkflow":
		return update, workflow.WorkflowPlural
	case "UpdateClusterWorkflowTemplate":
		return update, workflow.ClusterWorkflowTemplatePlural
	case "UpdateCronWorkflow":
		return update, workflow.CronWorkflowPlural
	case "UpdateWorkflowTemplate":
		return update, workflow.WorkflowTemplatePlural
	case "WatchWorkflows":
		return watch, workflow.WorkflowPlural
	}
	panic("cannot parse " + value)
}
