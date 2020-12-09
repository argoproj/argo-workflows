package cron

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	typed "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	argoerr "github.com/argoproj/argo/util/errors"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

type cronWfOperationCtx struct {
	// CronWorkflow is the CronWorkflow to be run
	name        string
	cronWf      *v1alpha1.CronWorkflow
	wfClientset versioned.Interface
	wfClient    typed.WorkflowInterface
	cronWfIf    typed.CronWorkflowInterface
	log         *log.Entry
	metrics     *metrics.Metrics
	// scheduledTimeFunc returns the last scheduled time when it is called
	scheduledTimeFunc ScheduledTimeFunc
}

func newCronWfOperationCtx(cronWorkflow *v1alpha1.CronWorkflow, wfClientset versioned.Interface, metrics *metrics.Metrics) *cronWfOperationCtx {
	return &cronWfOperationCtx{
		name:        cronWorkflow.ObjectMeta.Name,
		cronWf:      cronWorkflow,
		wfClientset: wfClientset,
		wfClient:    wfClientset.ArgoprojV1alpha1().Workflows(cronWorkflow.Namespace),
		cronWfIf:    wfClientset.ArgoprojV1alpha1().CronWorkflows(cronWorkflow.Namespace),
		log: log.WithFields(log.Fields{
			"workflow":  cronWorkflow.ObjectMeta.Name,
			"namespace": cronWorkflow.ObjectMeta.Namespace,
		}),
		metrics: metrics,
		// inferScheduledTime returns an inferred scheduled time based on the current time and only works if it is called
		// within 59 seconds of the scheduled time. Here it acts as a placeholder until it is replaced by a similar
		// function that returns the last scheduled time deterministically from the cron engine. Since we are only able
		// to generate the latter function after the job is scheduled, there is a tiny chance that the job is run before
		// the deterministic function is supplanted. If that happens, we use the infer function as the next-best thing
		scheduledTimeFunc: inferScheduledTime,
	}
}

func (woc *cronWfOperationCtx) Run() {
	woc.run(woc.scheduledTimeFunc())
}

func (woc *cronWfOperationCtx) run(scheduledRuntime time.Time) {
	defer woc.persistUpdate()

	woc.log.Infof("Running %s", woc.name)

	err := woc.validateCronWorkflow()
	if err != nil {
		return
	}

	proceed, err := woc.enforceRuntimePolicy()
	if err != nil {
		woc.reportCronWorkflowError(v1alpha1.ConditionTypeSubmissionError, fmt.Sprintf("Concurrency policy error: %s", err))
		return
	} else if !proceed {
		return
	}

	wf := common.ConvertCronWorkflowToWorkflowWithName(woc.cronWf, getChildWorkflowName(woc.cronWf.Name, scheduledRuntime))

	runWf, err := util.SubmitWorkflow(woc.wfClient, woc.wfClientset, woc.cronWf.Namespace, wf, &v1alpha1.SubmitOpts{})
	if err != nil {
		woc.reportCronWorkflowError(v1alpha1.ConditionTypeSubmissionError, fmt.Sprintf("Failed to submit Workflow: %s", err))
		return
	}

	woc.cronWf.Status.Active = append(woc.cronWf.Status.Active, getWorkflowObjectReference(wf, runWf))
	woc.cronWf.Status.LastScheduledTime = &v1.Time{Time: scheduledRuntime}
	woc.cronWf.Status.Conditions.RemoveCondition(v1alpha1.ConditionTypeSubmissionError)
}

func (woc *cronWfOperationCtx) validateCronWorkflow() error {
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(woc.wfClientset.ArgoprojV1alpha1().WorkflowTemplates(woc.cronWf.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(woc.wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	err := validate.ValidateCronWorkflow(wftmplGetter, cwftmplGetter, woc.cronWf)
	if err != nil {
		woc.reportCronWorkflowError(v1alpha1.ConditionTypeSpecError, fmt.Sprint(err))
	} else {
		woc.cronWf.Status.Conditions.RemoveCondition(v1alpha1.ConditionTypeSpecError)
	}
	return err
}

func getWorkflowObjectReference(wf *v1alpha1.Workflow, runWf *v1alpha1.Workflow) corev1.ObjectReference {
	// This is a bit of a hack. Ideally we'd use ref.GetReference, but for some reason the `runWf` object is coming back
	// without `Kind` and `APIVersion` set (even though it it set on `wf`). To fix this, we hard code those values.
	return corev1.ObjectReference{
		Kind:            wf.Kind,
		APIVersion:      wf.APIVersion,
		Name:            runWf.GetName(),
		Namespace:       runWf.GetNamespace(),
		UID:             runWf.GetUID(),
		ResourceVersion: runWf.GetResourceVersion(),
	}
}

func (woc *cronWfOperationCtx) persistUpdate() {
	woc.patch(map[string]interface{}{"status": woc.cronWf.Status})
}

func (woc *cronWfOperationCtx) persistUpdateActiveWorkflows() {
	woc.patch(map[string]interface{}{"status": map[string]interface{}{"active": woc.cronWf.Status.Active}})
}

func (woc *cronWfOperationCtx) patch(patch map[string]interface{}) {
	data, err := json.Marshal(patch)
	if err != nil {
		woc.log.WithError(err).Error("failed to marshall cron workflow status.active data")
		return
	}
	err = wait.ExponentialBackoff(retry.DefaultBackoff, func() (bool, error) {
		cronWf, err := woc.cronWfIf.Patch(woc.cronWf.Name, types.MergePatchType, data)
		if err != nil {
			if argoerr.IsTransientErr(err) {
				return false, nil
			}
			return false, err
		}
		woc.cronWf = cronWf
		return true, nil
	})
	if err != nil {
		woc.log.WithError(err).Error("failed to data cron workflow")
		return
	}
}

func (woc *cronWfOperationCtx) enforceRuntimePolicy() (bool, error) {
	if woc.cronWf.Spec.Suspend {
		woc.log.Infof("%s is suspended, skipping execution", woc.name)
		return false, nil
	}

	if woc.cronWf.Spec.ConcurrencyPolicy != "" {
		switch woc.cronWf.Spec.ConcurrencyPolicy {
		case v1alpha1.AllowConcurrent, "":
			// Do nothing
		case v1alpha1.ForbidConcurrent:
			if len(woc.cronWf.Status.Active) > 0 {
				woc.log.Infof("%s has 'ConcurrencyPolicy: Forbid' and has an active Workflow so it was not run", woc.name)
				return false, nil
			}
		case v1alpha1.ReplaceConcurrent:
			if len(woc.cronWf.Status.Active) > 0 {
				woc.log.Infof("%s has 'ConcurrencyPolicy: Replace' and has active Workflows", woc.name)
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
		woc.log.Infof("stopping '%s'", wfObjectRef.Name)
		err := util.TerminateWorkflow(woc.wfClient, wfObjectRef.Name)
		if err != nil {
			return fmt.Errorf("error stopping workflow %s: %e", wfObjectRef.Name, err)
		}
	}
	return nil
}

func (woc *cronWfOperationCtx) runOutstandingWorkflows() (bool, error) {
	missedExecutionTime, err := woc.shouldOutstandingWorkflowsBeRun()
	if err != nil {
		return false, err
	}
	if !missedExecutionTime.IsZero() {
		woc.run(missedExecutionTime)
		return true, nil
	}
	return false, nil
}

func (woc *cronWfOperationCtx) shouldOutstandingWorkflowsBeRun() (time.Time, error) {
	// If this CronWorkflow has been run before, check if we have missed any scheduled executions
	if woc.cronWf.Status.LastScheduledTime != nil {
		var now time.Time
		var cronSchedule cron.Schedule
		if woc.cronWf.Spec.Timezone != "" {
			loc, err := time.LoadLocation(woc.cronWf.Spec.Timezone)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid timezone '%s': %s", woc.cronWf.Spec.Timezone, err)
			}
			now = time.Now().In(loc)

			cronScheduleString := "CRON_TZ=" + woc.cronWf.Spec.Timezone + " " + woc.cronWf.Spec.Schedule
			cronSchedule, err = cron.ParseStandard(cronScheduleString)
			if err != nil {
				return time.Time{}, fmt.Errorf("unable to form timezone schedule '%s': %s", cronScheduleString, err)
			}
		} else {
			var err error
			now = time.Now()
			cronSchedule, err = cron.ParseStandard(woc.cronWf.Spec.Schedule)
			if err != nil {
				return time.Time{}, err
			}
		}

		var missedExecutionTime time.Time
		nextScheduledRunTime := cronSchedule.Next(woc.cronWf.Status.LastScheduledTime.Time)
		// Workflow should have ran
		for nextScheduledRunTime.Before(now) {
			missedExecutionTime = nextScheduledRunTime
			nextScheduledRunTime = cronSchedule.Next(missedExecutionTime)
		}

		// We missed the latest execution time
		if !missedExecutionTime.IsZero() {
			// If StartingDeadlineSeconds is not set, or we are still within the deadline window, run the Workflow
			if woc.cronWf.Spec.StartingDeadlineSeconds == nil || *woc.cronWf.Spec.StartingDeadlineSeconds == 0 || now.Before(missedExecutionTime.Add(time.Duration(*woc.cronWf.Spec.StartingDeadlineSeconds)*time.Second)) {
				woc.log.Infof("%s missed an execution at %s and is within StartingDeadline", woc.cronWf.Name, missedExecutionTime.Format("Mon Jan _2 15:04:05 2006"))
				return missedExecutionTime, nil
			}
		}
	}
	return time.Time{}, nil
}

func (woc *cronWfOperationCtx) reconcileActiveWfs(workflows []v1alpha1.Workflow) error {
	updated := false
	currentWfsFulfilled := make(map[types.UID]bool)
	for _, wf := range workflows {
		currentWfsFulfilled[wf.UID] = wf.Status.Fulfilled()

		if !woc.cronWf.Status.HasActiveUID(wf.UID) && !wf.Status.Fulfilled() {
			updated = true
			woc.cronWf.Status.Active = append(woc.cronWf.Status.Active, getWorkflowObjectReference(&wf, &wf))
		}
	}

	for _, objectRef := range woc.cronWf.Status.Active {
		if fulfilled, found := currentWfsFulfilled[objectRef.UID]; !found || fulfilled {
			updated = true
			woc.removeFromActiveList(objectRef.UID)
		}
	}

	if updated {
		woc.persistUpdateActiveWorkflows()
	}

	return nil
}

func (woc *cronWfOperationCtx) removeFromActiveList(uid types.UID) {
	var newActive []corev1.ObjectReference
	for _, ref := range woc.cronWf.Status.Active {
		if ref.UID != uid {
			newActive = append(newActive, ref)
		}
	}
	woc.cronWf.Status.Active = newActive
}

func (woc *cronWfOperationCtx) enforceHistoryLimit(workflows []v1alpha1.Workflow) error {
	woc.log.Infof("Enforcing history limit for '%s'", woc.cronWf.Name)

	var successfulWorkflows []v1alpha1.Workflow
	var failedWorkflows []v1alpha1.Workflow
	for _, wf := range workflows {
		if wf.Labels[common.LabelKeyCronWorkflow] != woc.cronWf.Name {
			continue
		}
		if wf.Status.Fulfilled() {
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
	err := woc.deleteOldestWorkflows(successfulWorkflows, int(workflowsToKeep))
	if err != nil {
		return fmt.Errorf("unable to delete Successful Workflows of CronWorkflow '%s': %s", woc.cronWf.Name, err)
	}

	workflowsToKeep = int32(1)
	if woc.cronWf.Spec.FailedJobsHistoryLimit != nil && *woc.cronWf.Spec.FailedJobsHistoryLimit >= 0 {
		workflowsToKeep = *woc.cronWf.Spec.FailedJobsHistoryLimit
	}
	err = woc.deleteOldestWorkflows(failedWorkflows, int(workflowsToKeep))
	if err != nil {
		return fmt.Errorf("unable to delete Failed Workflows of CronWorkflow '%s': %s", woc.cronWf.Name, err)
	}
	return nil
}

func (woc *cronWfOperationCtx) deleteOldestWorkflows(jobList []v1alpha1.Workflow, workflowsToKeep int) error {
	if workflowsToKeep >= len(jobList) {
		return nil
	}

	sort.SliceStable(jobList, func(i, j int) bool {
		return jobList[i].Status.FinishedAt.Time.After(jobList[j].Status.FinishedAt.Time)
	})

	for _, wf := range jobList[workflowsToKeep:] {
		err := woc.wfClient.Delete(wf.Name, &v1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				woc.log.Infof("Workflow '%s' was already deleted", wf.Name)
				continue
			}
			return fmt.Errorf("error deleting workflow '%s': %e", wf.Name, err)
		}
		woc.log.Infof("Deleted Workflow '%s' due to CronWorkflow '%s' history limit", wf.Name, woc.cronWf.Name)
	}
	return nil
}

func (woc *cronWfOperationCtx) reportCronWorkflowError(conditionType v1alpha1.ConditionType, errString string) {
	woc.log.WithField("conditionType", conditionType).Error(errString)
	woc.cronWf.Status.Conditions.UpsertCondition(v1alpha1.Condition{
		Type:    conditionType,
		Message: errString,
		Status:  v1.ConditionTrue,
	})
	woc.metrics.CronWorkflowSubmissionError()
}

func inferScheduledTime() time.Time {
	// Infer scheduled runtime by getting current time and zeroing out current seconds and nanoseconds
	// This works because the finest possible scheduled runtime is a minute. It is unlikely to ever be used, since this
	// function is quickly supplanted by a deterministic function from the cron engine.
	now := time.Now().UTC()
	scheduledTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())

	log.Infof("inferred scheduled time: %s", scheduledTime)
	return scheduledTime
}

func getChildWorkflowName(cronWorkflowName string, scheduledRuntime time.Time) string {
	return fmt.Sprintf("%s-%d", cronWorkflowName, scheduledRuntime.Unix())
}
