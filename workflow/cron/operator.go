package cron

import (
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	typed "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	"github.com/prometheus/common/log"
	"github.com/robfig/cron"
	"k8s.io/api/batch/v2alpha1"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	runWf, err := util.SubmitWorkflow(woc.wfClient, woc.wfClientset, woc.cronWf.Options.RuntimeNamespace, wf, &util.SubmitOpts{})
	if err != nil {
		log.Errorf("Failed to run CronWorkflow: %v", err)
		return
	}

	woc.cronWf.Status.Active = append(woc.cronWf.Status.Active, *getWorkflowObjectReference(wf, runWf))
	woc.cronWf.Status.LastScheduledTime = &v1.Time{Time: time.Now().UTC()}
	err = woc.persistUpdate()
	if err != nil {
		log.Error(err)
	}

	log.Infof("Created %s", woc.cronWf.ObjectMeta.Name)
}

func getWorkflowObjectReference(wf *v1alpha1.Workflow, runWf *v1alpha1.Workflow) *v12.ObjectReference {
	// This is a bit of a hack. Ideally we'd use ref.GetReference, but for some reason the `runWf` object is coming back
	// without `Kind` and `APIVersion` set (even though it it set on `wf`). To fix this, we hard code those values.
	return &v12.ObjectReference{
		Kind:            wf.Kind,
		APIVersion:      wf.APIVersion,
		Name:            runWf.GetName(),
		Namespace:       runWf.GetNamespace(),
		UID:             runWf.GetUID(),
		ResourceVersion: runWf.GetResourceVersion(),
	}
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
			if len(woc.cronWf.Status.Active) > 0 {
				log.Infof("%s has 'ConcurrencyPolicy: Forbid' and has an active Workflow so it was not run", woc.name)
				return false, nil
			}
		case v2alpha1.ReplaceConcurrent:
			if len(woc.cronWf.Status.Active) > 0 {
				log.Infof("%s has 'ConcurrencyPolicy: Replace' and has active Workflows", woc.name)
				err := woc.terminateOutstandingWorkflows()
				if err != nil {
					return false, err
				}
			}
		default:
			return false, fmt.Errorf("invalid ConcurrencyPolicy: %s", woc.cronWf.Options.ConcurrencyPolicy)
		}
	}
	return true, nil
}

func (woc *cronWfOperationCtx) terminateOutstandingWorkflows() error {
	for _, wfObjectRef := range woc.cronWf.Status.Active {
		log.Infof("stopping '%s'", wfObjectRef.Name)
		err := util.TerminateWorkflow(woc.wfClient, wfObjectRef.Name)
		if err != nil {
			return fmt.Errorf("error stopping workflow %s: %e", wfObjectRef.Name, err)
		}
	}
	return nil
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
