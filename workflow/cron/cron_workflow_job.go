package cron

import (
	"fmt"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	"github.com/prometheus/common/log"
	"k8s.io/api/batch/v2alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type CronWorkflowJob struct {
	// CronWorkflow is the CronWorkflow to be run
	jobName     string
	cronWf      *v1alpha1.Workflow
	options     v1alpha1.CronWorkflowOptions
	wfClientset versioned.Interface
}

func NewCronWorkflowJob(cronWorkflow *v1alpha1.CronWorkflow, wfClientset versioned.Interface) (*CronWorkflowJob, error) {
	wf, err := castToWorkflow(cronWorkflow)
	if err != nil {
		return nil, errors.InternalWrapError(err, "Unable to create CronWorkflowJob")
	}
	return &CronWorkflowJob{
		jobName:     cronWorkflow.ObjectMeta.Name,
		cronWf:      wf,
		options:     cronWorkflow.Options,
		wfClientset: wfClientset,
	}, nil
}

func (job *CronWorkflowJob) Run() {
	log.Infof("Running %s", job.jobName)
	runtimeNamespace := job.options.RuntimeNamespace

	wfClient := job.wfClientset.ArgoprojV1alpha1().Workflows(runtimeNamespace)

	if job.options.ConcurrencyPolicy != "" {
		switch job.options.ConcurrencyPolicy {
		case v2alpha1.ForbidConcurrent:
			labelSelector := labels.NewSelector()
			req, _ := labels.NewRequirement(common.LabelCronWorkflowParent, selection.Equals, []string{job.cronWf.Name})
			labelSelector.Add(*req)
			wfList, _ := job.wfClientset.ArgoprojV1alpha1().Workflows(runtimeNamespace).List(v1.ListOptions{
				LabelSelector: labelSelector.String(),
			})
			for _, wf := range wfList.Items {
				if wf.Status.Phase == v1alpha1.NodeRunning {
					log.Infof("%s has ConcurrencyPolicy: Forbid and has an active Workflow so it was not run", job.jobName)
					return
				}
			}
		case v2alpha1.ReplaceConcurrent:
			labelSelector := labels.NewSelector()
			req, _ := labels.NewRequirement(common.LabelCronWorkflowParent, selection.Equals, []string{job.cronWf.Name})
			labelSelector.Add(*req)
			wfList, _ := job.wfClientset.ArgoprojV1alpha1().Workflows(runtimeNamespace).List(v1.ListOptions{
				LabelSelector: labelSelector.String(),
			})
			for _, wf := range wfList.Items {
				if wf.Status.Phase == v1alpha1.NodeRunning {
					log.Infof("%s has ConcurrencyPolicy: Replace and has an active Workflow %s", job.jobName, wf.Name)
					log.Infof("Stopping %s...", wf.Name)
					err := util.TerminateWorkflow(wfClient, wf.Name)
					if err != nil {
						log.Errorf("Error in stopping workflow %s: %s", wf.Name, err)
						return
					}
				}
			}
		}
	}

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

	newObjectMeta.Labels = make(map[string]string)
	newObjectMeta.Labels[common.LabelCronWorkflowParent] = cronWf.Name

	return &v1alpha1.Workflow{
		TypeMeta:   newTypeMeta,
		ObjectMeta: newObjectMeta,
		Spec:       cronWf.Spec,
	}, nil
}
