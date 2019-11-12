package cron

import (
	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/util"
	"github.com/prometheus/common/log"
)

type CronWorkflowJob struct {
	// Name is the name of the CronWorkflow template to be run
	Name string

	// CronWorkflow is the CronWorkflow to be run
	// TODO: Maybe do the casting during submission and store the actual Workflow type here?
	cronWf *v1alpha1.CronWorkflow

	wfClientset versioned.Interface
}

func NewCronWorkflowJob(templateName string, cronWorkflow *v1alpha1.CronWorkflow, wfClientset versioned.Interface) *CronWorkflowJob {
	return &CronWorkflowJob{
		Name:   templateName,
		cronWf: cronWorkflow,
		wfClientset: wfClientset,
	}
}

func (job *CronWorkflowJob) Run() {
	log.Infof("Running %s", job.Name)
	wf := job.castToWorkflow()
	runtimeNamespace := job.cronWf.Options.RuntimeNamespace
	wfClient := job.wfClientset.ArgoprojV1alpha1().Workflows(runtimeNamespace)
	// TODO: Is this the best way to submit Workflows?
	// TODO: SubmitOpts is currently always nil
	_, err := util.SubmitWorkflow(wfClient, job.wfClientset, runtimeNamespace, wf, &util.SubmitOpts{})
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}
	log.Infof("Created %s", job.Name)
}

func (job *CronWorkflowJob) castToWorkflow() *v1alpha1.Workflow {
	// TODO: Overall, is this the best way to create the actual Workflow object?
	newTypeMeta := job.cronWf.TypeMeta
	newTypeMeta.Kind = workflow.WorkflowKind
	return &v1alpha1.Workflow{
		TypeMeta:   newTypeMeta,
		ObjectMeta: job.cronWf.ObjectMeta,
		Spec:       job.cronWf.Spec,
	}
}
