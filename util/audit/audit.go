package audit

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
)

type WorkflowAudit string

const (
	WorkflowAuditCreate    WorkflowAudit = "Create"
	WorkflowAuditDelete    WorkflowAudit = "Delete"
	WorkflowAuditRetry     WorkflowAudit = "Retry"
	WorkflowAuditResubmit  WorkflowAudit = "Resubmit"
	WorkflowAuditResume    WorkflowAudit = "Resume"
	WorkflowAuditSuspend   WorkflowAudit = "Suspend"
	WorkflowAuditTerminate WorkflowAudit = "Terminate"
	WorkflowAuditStop      WorkflowAudit = "Stop"
	WorkflowAuditSet       WorkflowAudit = "Set"
	WorkflowAuditSubmit    WorkflowAudit = "Submit"
)

func LogWorkflowAudit(ctx context.Context, wf *wfv1.Workflow, wa WorkflowAudit) {
	fields := log.Fields{
		"name":             wf.Name,
		"namespace":        wf.Namespace,
		"phase":            wf.Labels[common.LabelKeyPhase],
		"workflowTemplate": wf.Labels[common.LabelKeyWorkflowTemplate],
		"cronWorkflow":     wf.Labels[common.LabelKeyCronWorkflow],
	}
	creator.LogFields(ctx, fields)
	log.WithFields(fields).Info(fmt.Sprintf("Workflow %s", wa))
}

type WorkflowTemplateAudit string

const (
	WorkflowTemplateAuditCreate WorkflowTemplateAudit = "Create"
	WorkflowTemplateAuditDelete WorkflowTemplateAudit = "Delete"
	WorkflowTemplateAuditUpdate WorkflowTemplateAudit = "Update"
)

func LogWorkflowTemplateAudit(ctx context.Context, wfTmpl *wfv1.WorkflowTemplate, wta WorkflowTemplateAudit) {
	fields := log.Fields{
		"name":      wfTmpl.Name,
		"namespace": wfTmpl.Namespace,
	}
	creator.LogFields(ctx, fields)
	log.WithFields(fields).Info(fmt.Sprintf("WorkflowTemplate %s", wta))
}

type CronWorkflowAudit string

const (
	CronWorkflowAuditCreate  CronWorkflowAudit = "Create"
	CronWorkflowAuditDelete  CronWorkflowAudit = "Delete"
	CronWorkflowAuditUpdate  CronWorkflowAudit = "Update"
	CronWorkflowAuditResume  CronWorkflowAudit = "Resume"
	CronWorkflowAuditSuspend CronWorkflowAudit = "Suspend"
)

func LogCronWorkflowAudit(ctx context.Context, cwf *wfv1.CronWorkflow, cwa CronWorkflowAudit) {
	fields := log.Fields{
		"name":      cwf.Name,
		"namespace": cwf.Namespace,
	}
	creator.LogFields(ctx, fields)
	log.WithFields(fields).Info(fmt.Sprintf("CronWorkflow %s", cwa))
}
