package cron

import (
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	typed "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	"github.com/prometheus/common/log"
	"k8s.io/api/batch/v2alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"time"
)

type CronWorkflowWrapper struct {
	// CronWorkflow is the CronWorkflow to be run
	name        string
	cronWf      *v1alpha1.CronWorkflow
	wfClientset versioned.Interface
	wfClient    typed.WorkflowInterface
	cronWfIf    typed.CronWorkflowInterface
}

func NewCronWorkflowWrapper(cronWorkflow *v1alpha1.CronWorkflow, wfClientset versioned.Interface, cronWfIf typed.CronWorkflowInterface) (*CronWorkflowWrapper, error) {
	runtimeNamespace := cronWorkflow.Options.RuntimeNamespace
	return &CronWorkflowWrapper{
		name:        cronWorkflow.ObjectMeta.Name,
		cronWf:      cronWorkflow,
		wfClientset: wfClientset,
		wfClient:    wfClientset.ArgoprojV1alpha1().Workflows(runtimeNamespace),
		cronWfIf:    cronWfIf,
	}, nil
}

func (cronWfWrp *CronWorkflowWrapper) Run() {
	log.Infof("Running %s", cronWfWrp.name)

	ok, err := cronWfWrp.enforceRuntimePolicy()
	if err != nil {
		log.Errorf("Concurrency policy error: %s", err)
		return
	} else if !ok {
		return
	}

	wf, err := castToWorkflow(cronWfWrp.cronWf)
	if err != nil {
		log.Errorf("Unable to create Workflow for CronWorkflow %s", cronWfWrp.name)
		return
	}

	_, err = util.SubmitWorkflow(cronWfWrp.wfClient, cronWfWrp.wfClientset, cronWfWrp.cronWf.Options.RuntimeNamespace, wf, &util.SubmitOpts{})
	if err != nil {
		log.Errorf("Failed to run CronWorkflow: %v", err)
	}

	cronWfWrp.cronWf.Status.LastScheduledTime = &v1.Time{Time: time.Now().UTC()}
	err = cronWfWrp.persistUpdate()
	if err != nil {
		log.Error(err)
	}

	log.Infof("Created %s", cronWfWrp.cronWf.ObjectMeta.Name)
}

func (cronWfWrp *CronWorkflowWrapper) persistUpdate() error {
	_, err := cronWfWrp.cronWfIf.Update(cronWfWrp.cronWf)
	if err != nil {
		return fmt.Errorf("failed to update CronWorkflow: %w", err)
	}
	return nil
}

func (cronWfWrp *CronWorkflowWrapper) enforceRuntimePolicy() (bool, error) {
	if cronWfWrp.cronWf.Options.Suspend {
		log.Infof("%s is suspended, skipping execution", cronWfWrp.name)
		return false, nil
	}

	if cronWfWrp.cronWf.Options.ConcurrencyPolicy != "" {
		switch cronWfWrp.cronWf.Options.ConcurrencyPolicy {
		case v2alpha1.AllowConcurrent, "":
			// Do nothing
		case v2alpha1.ForbidConcurrent:
			runningWorkflows, err := cronWfWrp.getRunningWorkflows()
			if err != nil {
				return false, fmt.Errorf("error in running CronWorkflow %s: %w", cronWfWrp.name, err)
			}
			if len(runningWorkflows) > 0 {
				log.Infof("%s has 'ConcurrencyPolicy: Forbid' and has an active Workflow so it was not run", cronWfWrp.name)
				return false, nil
			}
		case v2alpha1.ReplaceConcurrent:
			runningWorkflows, err := cronWfWrp.getRunningWorkflows()
			if err != nil {
				return false, fmt.Errorf("error in running CronWorkflow %s: %w", cronWfWrp.name, err)
			}
			for _, wf := range runningWorkflows {
				log.Infof("%s has 'ConcurrencyPolicy: Replace' and has active Workflows. Stopping %s...", cronWfWrp.name, wf.Name)
				err := util.TerminateWorkflow(cronWfWrp.wfClient, wf.Name)
				if err != nil {
					return false, fmt.Errorf("error stopping workflow %s: %w", wf.Name, err)
				}
			}
		default:
			return false, fmt.Errorf("invalid ConcurrencyPolicy: %s", cronWfWrp.cronWf.Options.ConcurrencyPolicy)
		}
	}
	return true, nil
}

func (cronWfWrp *CronWorkflowWrapper) getRunningWorkflows() ([]v1alpha1.Workflow, error) {
	labelSelector := labels.NewSelector()
	req, err := labels.NewRequirement(common.LabelCronWorkflowParent, selection.Equals, []string{cronWfWrp.cronWf.Name})
	if err != nil {
		return nil, err
	}
	labelSelector = labelSelector.Add(*req)
	req, err = labels.NewRequirement(common.LabelKeyPhase, selection.Equals, []string{string(v1alpha1.NodeRunning)})
	if err != nil {
		return nil, err
	}
	labelSelector = labelSelector.Add(*req)
	wfList, err := cronWfWrp.wfClientset.ArgoprojV1alpha1().Workflows(cronWfWrp.cronWf.Options.RuntimeNamespace).List(v1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	return wfList.Items, nil
}

func castToWorkflow(cronWf *v1alpha1.CronWorkflow) (*v1alpha1.Workflow, error) {
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
