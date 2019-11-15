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
	"github.com/robfig/cron"
	"k8s.io/api/batch/v2alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"time"
)

type cronWfOperationCtx struct {
	// CronWorkflow is the CronWorkflow to be run
	name        string
	cronWf      *v1alpha1.CronWorkflow
	wfClientset versioned.Interface
	wfClient    typed.WorkflowInterface
	cronWfIf    typed.CronWorkflowInterface
}

func newCronWfOperationCtx(cronWorkflow *v1alpha1.CronWorkflow, wfClientset versioned.Interface, cronWfIf typed.CronWorkflowInterface) (*cronWfOperationCtx, error) {
	runtimeNamespace := cronWorkflow.Options.RuntimeNamespace
	return &cronWfOperationCtx{
		name:        cronWorkflow.ObjectMeta.Name,
		cronWf:      cronWorkflow,
		wfClientset: wfClientset,
		wfClient:    wfClientset.ArgoprojV1alpha1().Workflows(runtimeNamespace),
		cronWfIf:    cronWfIf,
	}, nil
}

func (woc *cronWfOperationCtx) Run() {
	log.Infof("Running %s", woc.name)

	ok, err := woc.enforceRuntimePolicy()
	if err != nil {
		log.Errorf("Concurrency policy error: %s", err)
		return
	} else if !ok {
		return
	}

	wf, err := common.CastToWorkflow(woc.cronWf)
	if err != nil {
		log.Errorf("Unable to create Workflow for CronWorkflow %s", woc.name)
		return
	}

	_, err = util.SubmitWorkflow(woc.wfClient, woc.wfClientset, woc.cronWf.Options.RuntimeNamespace, wf, &util.SubmitOpts{})
	if err != nil {
		log.Errorf("Failed to run CronWorkflow: %v", err)
	}

	woc.cronWf.Status.LastScheduledTime = &v1.Time{Time: time.Now().UTC()}
	err = woc.persistUpdate()
	if err != nil {
		log.Error(err)
	}

	log.Infof("Created %s", woc.cronWf.ObjectMeta.Name)
}

func (woc *cronWfOperationCtx) persistUpdate() error {
	_, err := woc.cronWfIf.Update(woc.cronWf)
	if err != nil {
		return fmt.Errorf("failed to update CronWorkflow: %w", err)
	}
	return nil
}

func (woc *cronWfOperationCtx) enforceRuntimePolicy() (bool, error) {
	if woc.cronWf.Options.Suspend {
		log.Infof("%s is suspended, skipping execution", woc.name)
		return false, nil
	}

	if woc.cronWf.Options.ConcurrencyPolicy != "" {
		switch woc.cronWf.Options.ConcurrencyPolicy {
		case v2alpha1.AllowConcurrent, "":
			// Do nothing
		case v2alpha1.ForbidConcurrent:
			runningWorkflows, err := woc.getRunningWorkflows()
			if err != nil {
				return false, fmt.Errorf("error in running CronWorkflow %s: %w", woc.name, err)
			}
			if len(runningWorkflows) > 0 {
				log.Infof("%s has 'ConcurrencyPolicy: Forbid' and has an active Workflow so it was not run", woc.name)
				return false, nil
			}
		case v2alpha1.ReplaceConcurrent:
			runningWorkflows, err := woc.getRunningWorkflows()
			if err != nil {
				return false, fmt.Errorf("error in running CronWorkflow %s: %w", woc.name, err)
			}
			for _, wf := range runningWorkflows {
				log.Infof("%s has 'ConcurrencyPolicy: Replace' and has active Workflows. Stopping %s...", woc.name, wf.Name)
				err := util.TerminateWorkflow(woc.wfClient, wf.Name)
				if err != nil {
					return false, fmt.Errorf("error stopping workflow %s: %w", wf.Name, err)
				}
			}
		default:
			return false, fmt.Errorf("invalid ConcurrencyPolicy: %s", woc.cronWf.Options.ConcurrencyPolicy)
		}
	}
	return true, nil
}

func (woc *cronWfOperationCtx) runOutstandingWorkflows() error {

	// If this CronWorkflow has been run before, check if we have missed any scheduled executions
	if woc.cronWf.Status.LastScheduledTime != nil {
		now := time.Now()
		var missedExecutionTime time.Time
		cronSchedule, err := cron.ParseStandard(woc.cronWf.Options.Schedule)
		if err != nil {
			return err
		}

		nextScheduledRunTime := cronSchedule.Next(woc.cronWf.Status.LastScheduledTime.Time)
		// Workflow should have ran
		for nextScheduledRunTime.Before(now) {
			missedExecutionTime = nextScheduledRunTime
			nextScheduledRunTime = cronSchedule.Next(missedExecutionTime)
		}

		// We missed the latest execution time
		if !missedExecutionTime.IsZero() {
			// If StartingDeadlineSeconds is not set, or we are still within the deadline window, run the Workflow
			if woc.cronWf.Options.StartingDeadlineSeconds == nil || now.Before(missedExecutionTime.Add(time.Duration(*woc.cronWf.Options.StartingDeadlineSeconds)*time.Second)) {
				log.Infof("%s missed an execution at %s and is within StartingDeadline", woc.cronWf.Name, missedExecutionTime.Format("Mon Jan _2 15:04:05 2006"))
				woc.Run()
			}
		}
	}
	return nil
}

func (woc *cronWfOperationCtx) getRunningWorkflows() ([]v1alpha1.Workflow, error) {
	labelSelector := labels.NewSelector()
	req, err := labels.NewRequirement(common.LabelCronWorkflowParent, selection.Equals, []string{woc.cronWf.Name})
	if err != nil {
		return nil, err
	}
	labelSelector = labelSelector.Add(*req)
	req, err = labels.NewRequirement(common.LabelKeyPhase, selection.Equals, []string{string(v1alpha1.NodeRunning)})
	if err != nil {
		return nil, err
	}
	labelSelector = labelSelector.Add(*req)
	wfList, err := woc.wfClientset.ArgoprojV1alpha1().Workflows(woc.cronWf.Options.RuntimeNamespace).List(v1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	return wfList.Items, nil
}
