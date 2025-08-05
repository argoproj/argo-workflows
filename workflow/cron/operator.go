package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	argoerrs "github.com/argoproj/argo-workflows/v3/errors"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	typed "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v3/util/retry"
	"github.com/argoproj/argo-workflows/v3/util/template"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"

	"github.com/argoproj/argo-workflows/v3/workflow/controller/informer"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

const (
	variablePrefix string = `cronworkflow`
)

type cronWfOperationCtx struct {
	// CronWorkflow is the CronWorkflow to be run
	name            string
	cronWf          *v1alpha1.CronWorkflow
	wfClientset     versioned.Interface
	wfClient        typed.WorkflowInterface
	wfDefaults      *v1alpha1.Workflow
	cronWfIf        typed.CronWorkflowInterface
	wftmplInformer  wfextvv1alpha1.WorkflowTemplateInformer
	cwftmplInformer wfextvv1alpha1.ClusterWorkflowTemplateInformer
	log             *log.Entry
	metrics         *metrics.Metrics
	// scheduledTimeFunc returns the last scheduled time when it is called
	scheduledTimeFunc ScheduledTimeFunc
}

func newCronWfOperationCtx(cronWorkflow *v1alpha1.CronWorkflow, wfClientset versioned.Interface,
	metrics *metrics.Metrics, wftmplInformer wfextvv1alpha1.WorkflowTemplateInformer,
	cwftmplInformer wfextvv1alpha1.ClusterWorkflowTemplateInformer, wfDefaults *v1alpha1.Workflow,
) *cronWfOperationCtx {
	return &cronWfOperationCtx{
		name:            cronWorkflow.Name,
		cronWf:          cronWorkflow,
		wfClientset:     wfClientset,
		wfClient:        wfClientset.ArgoprojV1alpha1().Workflows(cronWorkflow.Namespace),
		wfDefaults:      wfDefaults,
		cronWfIf:        wfClientset.ArgoprojV1alpha1().CronWorkflows(cronWorkflow.Namespace),
		wftmplInformer:  wftmplInformer,
		cwftmplInformer: cwftmplInformer,
		log: log.WithFields(log.Fields{
			"workflow":  cronWorkflow.Name,
			"namespace": cronWorkflow.Namespace,
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

// Run handles the running of a cron workflow
// It fits the github.com/robfig/cron.Job interface
func (woc *cronWfOperationCtx) Run() {
	ctx := context.Background()
	woc.run(ctx, woc.scheduledTimeFunc())
}

func (woc *cronWfOperationCtx) run(ctx context.Context, scheduledRuntime time.Time) {
	defer woc.persistUpdate(ctx)

	woc.log.Infof("Running %s", woc.name)

	// If the cron workflow has a schedule that was just updated, update its annotation
	if woc.cronWf.IsUsingNewSchedule() {
		woc.cronWf.SetSchedule(woc.cronWf.Spec.GetScheduleWithTimezoneString())
	}

	err := woc.validateCronWorkflow(ctx)
	if err != nil {
		return
	}

	completed, err := woc.checkStopingCondition()
	if err != nil {
		woc.reportCronWorkflowError(ctx, v1alpha1.ConditionTypeSpecError, fmt.Sprintf("failed to check CronWorkflow '%s' stopping condition: %s", woc.cronWf.Name, err))
		return
	} else if completed {
		woc.setAsCompleted()
	}

	proceed, err := woc.enforceRuntimePolicy(ctx)
	if err != nil {
		woc.reportCronWorkflowError(ctx, v1alpha1.ConditionTypeSubmissionError, fmt.Sprintf("run policy error: %s", err))
		return
	} else if !proceed {
		return
	}

	woc.metrics.CronWfTrigger(ctx, woc.name, woc.cronWf.Namespace)

	wf := common.ConvertCronWorkflowToWorkflowWithProperties(woc.cronWf, getChildWorkflowName(woc.cronWf.Name, scheduledRuntime), scheduledRuntime)

	runWf, err := util.SubmitWorkflow(ctx, woc.wfClient, woc.wfClientset, woc.cronWf.Namespace, wf, woc.wfDefaults, &v1alpha1.SubmitOpts{})
	if err != nil {
		// If the workflow already exists (i.e. this is a duplicate submission), do not report an error
		if errors.IsAlreadyExists(err) {
			return
		}
		woc.reportCronWorkflowError(ctx, v1alpha1.ConditionTypeSubmissionError, fmt.Sprintf("Failed to submit Workflow: %s", err))
		return
	}

	woc.cronWf.Status.Active = append(woc.cronWf.Status.Active, getWorkflowObjectReference(wf, runWf))
	woc.cronWf.Status.Phase = v1alpha1.ActivePhase
	woc.cronWf.Status.LastScheduledTime = &v1.Time{Time: scheduledRuntime}
	woc.cronWf.Status.Conditions.RemoveCondition(v1alpha1.ConditionTypeSubmissionError)
}

func (woc *cronWfOperationCtx) validateCronWorkflow(ctx context.Context) error {
	wftmplGetter := informer.NewWorkflowTemplateFromInformerGetter(woc.wftmplInformer, woc.cronWf.Namespace)
	cwftmplGetter := informer.NewClusterWorkflowTemplateFromInformerGetter(woc.cwftmplInformer)
	err := validate.ValidateCronWorkflow(ctx, wftmplGetter, cwftmplGetter, woc.cronWf, woc.wfDefaults)
	if err != nil {
		woc.reportCronWorkflowError(ctx, v1alpha1.ConditionTypeSpecError, fmt.Sprint(err))
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

func (woc *cronWfOperationCtx) persistUpdate(ctx context.Context) {
	woc.patch(ctx, map[string]interface{}{"status": woc.cronWf.Status, "metadata": map[string]interface{}{"annotations": woc.cronWf.Annotations, "labels": woc.cronWf.Labels}})
}

func (woc *cronWfOperationCtx) persistCurrentWorkflowStatus(ctx context.Context) {
	woc.patch(ctx, map[string]interface{}{"status": map[string]interface{}{"active": woc.cronWf.Status.Active, "succeeded": woc.cronWf.Status.Succeeded, "failed": woc.cronWf.Status.Failed, "phase": woc.cronWf.Status.Phase}})
}

func (woc *cronWfOperationCtx) patch(ctx context.Context, patch map[string]interface{}) {
	data, err := json.Marshal(patch)
	if err != nil {
		woc.log.WithError(err).Error("failed to marshall cron workflow status.active data")
		return
	}
	err = waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		cronWf, err := woc.cronWfIf.Patch(ctx, woc.cronWf.Name, types.MergePatchType, data, v1.PatchOptions{})
		if err != nil {
			return !errorsutil.IsTransientErr(err), err
		}
		woc.cronWf = cronWf
		return true, nil
	})
	if err != nil {
		woc.log.WithError(err).Error("failed to update cron workflow")
		return
	}
}

// TODO: refactor shouldExecute in steps.go
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	expression, err := govaluate.NewEvaluableExpression(when)
	if err != nil {
		return false, err
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return false, err
	}

	boolRes, ok := result.(bool)
	if !ok {
		return false, argoerrs.Errorf(argoerrs.CodeBadRequest, "Expected boolean evaluation for '%s'. Got %v", when, result)
	}
	return boolRes, nil
}

func evalWhen(cron *v1alpha1.CronWorkflow) (bool, error) {
	if cron.Spec.When == "" {
		return true, nil
	}

	t, err := template.NewTemplate(string(cron.Spec.When))
	if err != nil {
		return false, err
	}
	env := make(map[string]interface{})
	addSetField := func(name string, value interface{}) {
		env[fmt.Sprintf("%s.%s", variablePrefix, name)] = value
	}
	err = expressionEnv(cron, addSetField)
	if err != nil {
		return false, err
	}
	newWhenStr, err := t.Replace(env, false)
	if err != nil {
		return false, err
	}
	newCron := cron.DeepCopy()
	newCron.Spec.When = newWhenStr

	return shouldExecute(newCron.Spec.When)
}

func (woc *cronWfOperationCtx) enforceRuntimePolicy(ctx context.Context) (bool, error) {
	if woc.cronWf.Spec.Suspend {
		woc.log.Infof("%s is suspended, skipping execution", woc.name)
		return false, nil
	}

	if woc.cronWf.Status.Phase == v1alpha1.StoppedPhase {
		woc.log.Infof("CronWorkflow %s is marked as stopped since it achieved the stopping condition", woc.cronWf.Name)
		return false, nil
	}

	canProceed, err := evalWhen(woc.cronWf)
	if err != nil || !canProceed {
		return canProceed, err
	}

	if woc.cronWf.Spec.ConcurrencyPolicy != "" {
		switch woc.cronWf.Spec.ConcurrencyPolicy {
		case v1alpha1.AllowConcurrent, "":
			// Do nothing
		case v1alpha1.ForbidConcurrent:
			if len(woc.cronWf.Status.Active) > 0 {
				woc.metrics.CronWfPolicy(ctx, woc.name, woc.cronWf.Namespace, v1alpha1.ForbidConcurrent)
				woc.log.Infof("%s has 'ConcurrencyPolicy: Forbid' and has an active Workflow so it was not run", woc.name)
				return false, nil
			}
		case v1alpha1.ReplaceConcurrent:
			if len(woc.cronWf.Status.Active) > 0 {
				woc.metrics.CronWfPolicy(ctx, woc.name, woc.cronWf.Namespace, v1alpha1.ReplaceConcurrent)
				woc.log.Infof("%s has 'ConcurrencyPolicy: Replace' and has active Workflows", woc.name)
				err := woc.terminateOutstandingWorkflows(ctx)
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

func (woc *cronWfOperationCtx) terminateOutstandingWorkflows(ctx context.Context) error {
	for _, wfObjectRef := range woc.cronWf.Status.Active {
		woc.log.Infof("stopping '%s'", wfObjectRef.Name)
		err := util.TerminateWorkflow(ctx, woc.wfClient, wfObjectRef.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				woc.log.Warnf("workflow %q not found when trying to terminate outstanding workflows", wfObjectRef.Name)
				continue
			}
			alreadyShutdownErr, ok := err.(util.AlreadyShutdownError)
			if ok {
				woc.log.Warn(alreadyShutdownErr.Error())
				continue
			}
			return fmt.Errorf("error stopping workflow %s: %e", wfObjectRef.Name, err)
		}
	}
	return nil
}

func (woc *cronWfOperationCtx) runOutstandingWorkflows(ctx context.Context) (bool, error) {
	missedExecutionTime, err := woc.shouldOutstandingWorkflowsBeRun(ctx)
	if err != nil {
		return false, err
	}
	if !missedExecutionTime.IsZero() {
		woc.run(ctx, missedExecutionTime)
		return true, nil
	}
	return false, nil
}

func (woc *cronWfOperationCtx) shouldOutstandingWorkflowsBeRun(ctx context.Context) (time.Time, error) {
	// If the CronWorkflow schedule was just updated, then do not run any outstanding workflows.
	if woc.cronWf.IsUsingNewSchedule() {
		return time.Time{}, nil
	}
	// If this CronWorkflow has been run before, check if we have missed any scheduled executions
	if woc.cronWf.Status.LastScheduledTime != nil {
		for _, schedule := range woc.cronWf.Spec.GetSchedulesWithTimezone(ctx) {
			var now time.Time
			var cronSchedule cron.Schedule
			now = time.Now()
			cronSchedule, err := cron.ParseStandard(schedule)
			if err != nil {
				return time.Time{}, err
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
				// if missedExecutionTime is within StartDeadlineSeconds, We are still within the deadline window, run the Workflow
				if woc.cronWf.Spec.StartingDeadlineSeconds != nil && now.Before(missedExecutionTime.Add(time.Duration(*woc.cronWf.Spec.StartingDeadlineSeconds)*time.Second)) {
					woc.log.Infof("%s missed an execution at %s and is within StartingDeadline", woc.cronWf.Name, missedExecutionTime.Format("Mon Jan _2 15:04:05 2006"))
					return missedExecutionTime, nil
				}
			}
		}
	}
	return time.Time{}, nil
}

type fulfilledWfsPhase struct {
	fulfilled bool
	phase     v1alpha1.WorkflowPhase
}

func (woc *cronWfOperationCtx) reconcileActiveWfs(ctx context.Context, workflows []v1alpha1.Workflow) error {
	updated := false
	currentWfsFulfilled := make(map[types.UID]fulfilledWfsPhase, len(workflows))
	for _, wf := range workflows {
		currentWfsFulfilled[wf.UID] = fulfilledWfsPhase{
			fulfilled: wf.Status.Fulfilled(),
			phase:     wf.Status.Phase,
		}
		if !woc.cronWf.Status.HasActiveUID(wf.UID) && !wf.Status.Fulfilled() {
			updated = true
			woc.cronWf.Status.Active = append(woc.cronWf.Status.Active, getWorkflowObjectReference(&wf, &wf))
		}
	}

	for _, objectRef := range woc.cronWf.Status.Active {
		if fulfilled, found := currentWfsFulfilled[objectRef.UID]; !found || fulfilled.fulfilled {
			updated = true
			woc.removeFromActiveList(objectRef.UID)
			if found && fulfilled.fulfilled {
				woc.updateWfPhaseCounter(fulfilled.phase)
				completed, err := woc.checkStopingCondition()
				if err != nil {
					return fmt.Errorf("failed to check CronWorkflow '%s' stopping condition: %s", woc.cronWf.Name, err)
				} else if completed {
					woc.setAsCompleted()
				}
			}
		}
	}

	if updated {
		woc.persistCurrentWorkflowStatus(ctx)
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

func (woc *cronWfOperationCtx) enforceHistoryLimit(ctx context.Context, workflows []v1alpha1.Workflow) error {
	woc.log.Debugf("Enforcing history limit for '%s'", woc.cronWf.Name)

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
	err := woc.deleteOldestWorkflows(ctx, successfulWorkflows, int(workflowsToKeep))
	if err != nil {
		return fmt.Errorf("unable to delete Successful Workflows of CronWorkflow '%s': %s", woc.cronWf.Name, err)
	}

	workflowsToKeep = int32(1)
	if woc.cronWf.Spec.FailedJobsHistoryLimit != nil && *woc.cronWf.Spec.FailedJobsHistoryLimit >= 0 {
		workflowsToKeep = *woc.cronWf.Spec.FailedJobsHistoryLimit
	}
	err = woc.deleteOldestWorkflows(ctx, failedWorkflows, int(workflowsToKeep))
	if err != nil {
		return fmt.Errorf("unable to delete Failed Workflows of CronWorkflow '%s': %s", woc.cronWf.Name, err)
	}
	return nil
}

func (woc *cronWfOperationCtx) deleteOldestWorkflows(ctx context.Context, jobList []v1alpha1.Workflow, workflowsToKeep int) error {
	if workflowsToKeep >= len(jobList) {
		return nil
	}

	sort.SliceStable(jobList, func(i, j int) bool {
		return jobList[i].Status.FinishedAt.After(jobList[j].Status.FinishedAt.Time)
	})

	for _, wf := range jobList[workflowsToKeep:] {
		err := woc.wfClient.Delete(ctx, wf.Name, v1.DeleteOptions{})
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

func (woc *cronWfOperationCtx) reportCronWorkflowError(ctx context.Context, conditionType v1alpha1.ConditionType, errString string) {
	woc.log.WithField("conditionType", conditionType).Error(errString)
	woc.cronWf.Status.Conditions.UpsertCondition(v1alpha1.Condition{
		Type:    conditionType,
		Message: errString,
		Status:  v1.ConditionTrue,
	})
	if conditionType == v1alpha1.ConditionTypeSpecError {
		woc.metrics.CronWorkflowSpecError(ctx)
	} else {
		if conditionType == v1alpha1.ConditionTypeSubmissionError {
			woc.cronWf.Status.Failed++
		}
		woc.metrics.CronWorkflowSubmissionError(ctx)
	}
}

func (woc *cronWfOperationCtx) updateWfPhaseCounter(phase v1alpha1.WorkflowPhase) {
	switch phase {
	case v1alpha1.WorkflowError, v1alpha1.WorkflowFailed:
		woc.cronWf.Status.Failed++
	case v1alpha1.WorkflowSucceeded:
		woc.cronWf.Status.Succeeded++
	}
}

func expressionEnv(cron *v1alpha1.CronWorkflow, addSetField func(name string, value interface{})) error {
	addSetField("name", cron.Name)
	addSetField("namespace", cron.Namespace)
	addSetField("labels", cron.Labels)
	addSetField("annotations", cron.Labels)
	addSetField("failed", cron.Status.Failed)
	addSetField("succeeded", cron.Status.Succeeded)

	labelsStr, err := json.Marshal(&cron.Labels)
	if err != nil {
		return err
	}

	annotationsStr, err := json.Marshal(&cron.Annotations)
	if err != nil {
		return err
	}

	addSetField("annotations.json", annotationsStr)
	addSetField("labels.json", labelsStr)

	var tm *time.Time
	tm = nil

	if cron.Status.LastScheduledTime != nil {
		tm = &cron.Status.LastScheduledTime.Time
	}

	addSetField("lastScheduledTime", tm)

	return nil
}

func (woc *cronWfOperationCtx) checkStopingCondition() (bool, error) {
	if woc.cronWf.Spec.StopStrategy == nil {
		return false, nil
	}
	prefixedEnv := make(map[string]interface{})
	addSetField := func(name string, value interface{}) {
		prefixedEnv[name] = value
	}
	env := make(map[string]interface{})
	env[variablePrefix] = prefixedEnv
	err := expressionEnv(woc.cronWf, addSetField)
	if err != nil {
		return false, err
	}

	suspend, err := argoexpr.EvalBool(woc.cronWf.Spec.StopStrategy.Expression, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate stop expression: %w", err)
	}
	return suspend, nil
}

func (woc *cronWfOperationCtx) setAsCompleted() {
	woc.cronWf.Status.Phase = v1alpha1.StoppedPhase
	if woc.cronWf.Labels == nil {
		woc.cronWf.Labels = map[string]string{}
	}
	woc.cronWf.Labels[common.LabelKeyCronWorkflowCompleted] = "true"
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
