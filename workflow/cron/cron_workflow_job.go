package cron

import (
	"fmt"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/util"
	"github.com/prometheus/common/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CronWorkflowJob struct {
	// Name is the namespace where the Workflow will be run
	RuntimeNamespace string

	// CronWorkflow is the CronWorkflow to be run
	cronWf *v1alpha1.Workflow

	wfClientset versioned.Interface
}

func NewCronWorkflowJob(templateName string, cronWorkflow *v1alpha1.CronWorkflow, wfClientset versioned.Interface) (*CronWorkflowJob, error) {
	wf, err := castToWorkflow(cronWorkflow)
	if err != nil {
		return nil, errors.InternalWrapError(err, "Unable to create CronWorkflowJob")
	}
	return &CronWorkflowJob{
		RuntimeNamespace: cronWorkflow.Options.RuntimeNamespace,
		cronWf:           wf,
		wfClientset:      wfClientset,
	}, nil
}

func (job *CronWorkflowJob) Run() {
	log.Infof("Running %s", job.cronWf.ObjectMeta.Name)
	runtimeNamespace := job.RuntimeNamespace
	wfClient := job.wfClientset.ArgoprojV1alpha1().Workflows(runtimeNamespace)
	// TODO: Is this the best way to submit Workflows?
	// TODO: SubmitOpts is currently always nil
	_, err := util.SubmitWorkflow(wfClient, job.wfClientset, runtimeNamespace, job.cronWf, &util.SubmitOpts{})
	if err != nil {
		log.Fatalf("Failed to run CronWorkflow: %v", err)
	}
	log.Infof("Created %s", job.cronWf.ObjectMeta.Name)
}

func castToWorkflow(cronWf *v1alpha1.CronWorkflow) (*v1alpha1.Workflow, error) {
	// TODO: Overall, is this the best way to create the actual Workflow object?
	newTypeMeta := v1.TypeMeta{
		Kind:       workflow.WorkflowKind,
		APIVersion: cronWf.TypeMeta.APIVersion,
	}
	newObjectMeta := v1.ObjectMeta{}
	if cronWf.Options.RuntimeGenerateName != "" {
		newObjectMeta.GenerateName = cronWf.Options.RuntimeGenerateName
	} else {
		return nil, fmt.Errorf("CronWorkflow should have runtimeGenerateName defined")
	}
	return &v1alpha1.Workflow{
		TypeMeta:   newTypeMeta,
		ObjectMeta: newObjectMeta,
		Spec:       cronWf.Spec,
	}, nil
}
