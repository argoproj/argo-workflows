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
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
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
	return &cronWfOperationCtx{
		name:        cronWorkflow.ObjectMeta.Name,
		cronWf:      cronWorkflow,
		wfClientset: wfClientset,
		wfClient:    wfClientset.ArgoprojV1alpha1().Workflows(cronWorkflow.Namespace),
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

	runWf, err := util.SubmitWorkflow(woc.wfClient, woc.wfClientset, woc.cronWf.Namespace, wf, &util.SubmitOpts{})
	if err != nil {
		log.Errorf("Failed to run CronWorkflow: %v", err)
		return
	}

	woc.cronWf.Status.Active = append(woc.cronWf.Status.Active, getWorkflowObjectReference(wf, runWf))
	woc.cronWf.Status.LastScheduledTime = &v1.Time{Time: time.Now().UTC()}
	err = woc.persistUpdate()
	if err != nil {
		log.Error(err)
	}

	log.Infof("Created %s", woc.cronWf.ObjectMeta.Name)
}

func getWorkflowObjectReference(wf *v1alpha1.Workflow, runWf *v1alpha1.Workflow) v12.ObjectReference {
	// This is a bit of a hack. Ideally we'd use ref.GetReference, but for some reason the `runWf` object is coming back
	// without `Kind` and `APIVersion` set (even though it it set on `wf`). To fix this, we hard code those values.
	return v12.ObjectReference{
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
	if woc.cronWf.Spec.Suspend {
		log.Infof("%s is suspended, skipping execution", woc.name)
		return false, nil
	}

	if woc.cronWf.Spec.ConcurrencyPolicy != "" {
		switch woc.cronWf.Spec.ConcurrencyPolicy {
		case v1alpha1.AllowConcurrent, "":
			// Do nothing
		case v1alpha1.ForbidConcurrent:
			if len(woc.cronWf.Status.Active) > 0 {
				log.Infof("%s has 'ConcurrencyPolicy: Forbid' and has an active Workflow so it was not run", woc.name)
				return false, nil
			}
		case v1alpha1.ReplaceConcurrent:
			if len(woc.cronWf.Status.Active) > 0 {
				log.Infof("%s has 'ConcurrencyPolicy: Replace' and has active Workflows", woc.name)
				err := woc.terminateOutstandingWorkflows()
				if err != nil {
					return false, err
				}
			}
		default:
			return false, fmt.Errorf("invalid ConcurrencyPolicy: %s", woc.cronWf.Spec.ConcurrencyPolicy)
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
		cronSchedule, err := cron.ParseStandard(woc.cronWf.Spec.Schedule)
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
			if woc.cronWf.Spec.StartingDeadlineSeconds == nil || now.Before(missedExecutionTime.Add(time.Duration(*woc.cronWf.Spec.StartingDeadlineSeconds)*time.Second)) {
				log.Infof("%s missed an execution at %s and is within StartingDeadline", woc.cronWf.Name, missedExecutionTime.Format("Mon Jan _2 15:04:05 2006"))
				woc.Run()
			}
		}
	}
	return nil
}

func (woc *cronWfOperationCtx) removeActiveWf(wf *v1alpha1.Workflow) {
	if wf == nil || wf.ObjectMeta.UID == "" {
		return
	}
	for i, objectRef := range woc.cronWf.Status.Active {
		if objectRef.UID == wf.ObjectMeta.UID {
			woc.cronWf.Status.Active = append(woc.cronWf.Status.Active[:i], woc.cronWf.Status.Active[i+1:]...)
			err := woc.persistUpdate()
			if err != nil {
				log.Errorf("Unable to update CronWorkflow '%s': %s", woc.cronWf.Name, err)
			}
		}
	}
}

func (woc *cronWfOperationCtx) enforceHistoryLimit() {
	listOptions := &v1.ListOptions{}
	wfInformerListOptionsFunc(listOptions)
	wfList, err := woc.wfClient.List(*listOptions)
	if err != nil {
		log.Errorf("Unable to enforce history limit for CronWorkflow '%s': %s", woc.cronWf.Name, err)
		return
	}

	var successfulWorkflows []v1alpha1.Workflow
	var failedWorkflows []v1alpha1.Workflow

	for _, wf := range wfList.Items {
		if wf.Status.Completed() {
			if wf.Status.Successful() {
				successfulWorkflows = append(successfulWorkflows, wf)
			} else {
				failedWorkflows = append(failedWorkflows, wf)
			}
		}
	}

	workflowsToKeep := int32(3)
	if woc.cronWf.Spec.SuccessfulJobsHistoryLimit != nil && *woc.cronWf.Spec.SuccessfulJobsHistoryLimit >= 0 {
		workflowsToKeep = *woc.cronWf.Spec.SuccessfulJobsHistoryLimit
	}
	err = woc.deleteOldestWorkflows(successfulWorkflows, int(workflowsToKeep))
	if err != nil {
		log.Errorf("Unable to delete Successful Workflows of CronWorkflow '%s': %s", woc.cronWf.Name, err)
		return
	}

	workflowsToKeep = int32(1)
	if woc.cronWf.Spec.FailedJobsHistoryLimit != nil && *woc.cronWf.Spec.FailedJobsHistoryLimit >= 0 {
		workflowsToKeep = *woc.cronWf.Spec.SuccessfulJobsHistoryLimit
	}
	err = woc.deleteOldestWorkflows(successfulWorkflows, int(workflowsToKeep))
	if err != nil {
		log.Errorf("Unable to delete Failed Workflows of CronWorkflow '%s': %s", woc.cronWf.Name, err)
		return
	}

}

type newestFirst []v1alpha1.Workflow

func (n newestFirst) Len() int {
	return len(n)
}

func (n newestFirst) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n newestFirst) Less(i, j int) bool {
	return n[i].Status.FinishedAt.Time.Before(n[j].Status.FinishedAt.Time)
}

func (woc *cronWfOperationCtx) deleteOldestWorkflows(jobList []v1alpha1.Workflow, workflowsToKeep int) error {
	if workflowsToKeep >= len(jobList) {
		return nil
	}

	sort.Sort(newestFirst(jobList))
	for _, wf := range jobList[workflowsToKeep:] {
		err := woc.wfClient.Delete(wf.Name, &v1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("error deleting workflow '%s': %e", wf.Name, err)
		}
	}
	return nil
}
