package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/resource"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	podcontroller "github.com/argoproj/argo-workflows/v4/workflow/controller/pod"
)

// setPodCompletionMetadata is also used by normal assessment. It contains only
// Pod-derived completion data, with no task-result, cache, metrics or queue effects.
func setPodCompletionMetadata(pod *apiv1.Pod, node *wfv1.NodeStatus) {
	node.FinishedAt = getLatestFinishedAt(pod)
	node.ResourcesDuration = resource.DurationForPod(pod)
}

// legacySuccessfulNode reconstructs one deliberately narrow class of results:
// a completed, single-container entrypoint with no external outputs. It never
// assesses a completed Workflow, replays memoization or copies an old result.
func legacySuccessfulNode(wf *wfv1.Workflow, pod *apiv1.Pod) (*wfv1.NodeStatus, error) {
	hold := podcontroller.StatusCaptureHold
	unsupported := podcontroller.CaptureLegacyUnsupported
	conflict := podcontroller.CaptureResultConflict
	owner := metav1.GetControllerOf(pod)
	if pod.UID == "" || wf.UID == "" || wf.ResourceVersion == "" || pod.Namespace != wf.Namespace || owner == nil || owner.APIVersion != workflow.APIVersion || owner.Kind != workflow.WorkflowKind || owner.Name != wf.Name || owner.UID != wf.UID || pod.Labels[common.LabelKeyWorkflow] != wf.Name || pod.Labels[common.LabelKeyControllerInstanceID] != wf.Labels[common.LabelKeyControllerInstanceID] || pod.Labels[common.LabelKeyComponent] != "" {
		return nil, hold(podcontroller.CaptureIdentityConflict, "legacy capture requires exact Workflow and Pod ownership")
	}
	if wf.DeletionTimestamp != nil || wf.Status.Phase != wfv1.WorkflowSucceeded || wf.Labels[common.LabelKeyCompleted] != "true" || len(wf.Status.Nodes) != 1 || pod.Status.Phase != apiv1.PodSucceeded {
		return nil, hold(unsupported, "legacy capture supports only a completed single-node successful Workflow")
	}
	node, ok := wf.Status.Nodes[wf.Name]
	if !ok || node.ID != wf.Name || node.Name != wf.Name || pod.Annotations[common.AnnotationKeyNodeID] != node.ID || pod.Annotations[common.AnnotationKeyNodeName] != node.Name || node.DisplayName != node.Name {
		return nil, hold(podcontroller.CaptureIdentityConflict, "legacy entrypoint node identity does not match Pod")
	}
	if node.CapturedPodUID != "" || node.Type != wfv1.NodeTypePod || node.Phase != wfv1.NodeSucceeded || node.MemoizationStatus != nil || node.Daemoned != nil || node.RestartingPodUID != "" || node.FailedPodRestarts != 0 || node.TemplateRef != nil || node.BoundaryID != "" || node.NodeFlag != nil || node.SynchronizationStatus != nil || node.Inputs != nil || len(node.Children) != 0 || len(node.OutboundNodes) != 0 || node.PodIP != "" {
		return nil, hold(unsupported, "legacy result includes an unsupported execution disposition")
	}
	spec := wf.GetExecSpec()
	if wf.Spec.Shutdown != "" || spec.Shutdown != "" || spec.OnExit != "" || len(spec.Hooks) != 0 || spec.RetryStrategy != nil || spec.ActiveDeadlineSeconds != nil || wf.Spec.ActiveDeadlineSeconds != nil || node.TemplateName != spec.Entrypoint {
		return nil, hold(unsupported, "legacy capture cannot replay shutdown, hooks or retries")
	}
	var tmpl *wfv1.Template
	for i := range spec.Templates {
		if spec.Templates[i].Name == node.TemplateName {
			if tmpl != nil {
				return nil, hold(unsupported, "legacy template name is ambiguous")
			}
			tmpl = &spec.Templates[i]
		}
	}
	if tmpl == nil || !simpleLegacyTemplate(tmpl) {
		return nil, hold(unsupported, "legacy capture requires a stored ordinary container definition without external results")
	}
	// ARGO_TEMPLATE is the immutable execution definition on the Pod, unlike a
	// current external WorkflowTemplate. Do not fetch a mutable external template.
	var executed *wfv1.Template
	for _, c := range append(append([]apiv1.Container{}, pod.Spec.InitContainers...), pod.Spec.Containers...) {
		for _, env := range c.Env {
			if env.Name != common.EnvVarTemplate {
				continue
			}
			var candidate wfv1.Template
			if env.ValueFrom != nil || json.Unmarshal([]byte(env.Value), &candidate) != nil {
				return nil, hold(conflict, "Pod execution template does not match the stored definition")
			}
			// The controller adds the artifact repository's archive location to the
			// executed template, so a stored ordinary container may still have saved
			// its logs as an artifact. That is a result outside the Pod, not a conflict.
			if !simpleLegacyTemplate(&candidate) {
				return nil, hold(unsupported, "Pod execution template has external results, such as archived logs")
			}
			if !sameLegacyTemplate(tmpl, &candidate) || executed != nil && !reflect.DeepEqual(executed, &candidate) {
				return nil, hold(conflict, "Pod execution template does not match the stored definition")
			}
			executed = &candidate
		}
	}
	if executed == nil {
		return nil, hold(unsupported, "Pod execution template is unavailable or offloaded")
	}
	if node.TaskResultSynced == nil {
		complete, exists := wf.Status.TaskResultsCompletionStatus[node.ID]
		if !exists || !complete {
			return nil, hold(unsupported, "legacy task-result completion is not explicit")
		}
	} else if !*node.TaskResultSynced {
		return nil, hold(podcontroller.CaptureResultPending, "legacy task-result synchronization is pending")
	}
	if wf.Status.IsTaskResultIncomplete(node.ID) {
		return nil, hold(podcontroller.CaptureResultPending, "legacy task-result synchronization is pending")
	}
	if !successfulContainerStatuses(pod.Spec.Containers, pod.Status.ContainerStatuses) || !successfulContainerStatuses(pod.Spec.InitContainers, pod.Status.InitContainerStatuses) || getExitCode(pod) == nil || pod.Spec.NodeName == "" {
		return nil, hold(conflict, "Pod does not contain complete successful container observations")
	}
	if node.StartedAt.IsZero() || wf.Status.StartedAt.IsZero() || wf.Status.FinishedAt.IsZero() || pod.CreationTimestamp.IsZero() || node.StartedAt.Before(&wf.Status.StartedAt) || node.StartedAt.After(pod.CreationTimestamp.Time) || node.EstimatedDuration < 0 || !legacyTemplateScopeMatches(wf, node.TemplateScope) {
		return nil, hold(conflict, "legacy execution metadata is inconsistent")
	}
	expected := &wfv1.NodeStatus{
		ID: wf.Name, Name: wf.Name, DisplayName: wf.Name, Type: wfv1.NodeTypePod,
		TemplateName: tmpl.Name, TemplateScope: node.TemplateScope,
		Phase: wfv1.NodeSucceeded, StartedAt: node.StartedAt,
		EstimatedDuration: node.EstimatedDuration, Progress: wfv1.ProgressDefault.Complete(),
		Outputs:      &wfv1.Outputs{ExitCode: new(fmt.Sprint(*getExitCode(pod)))},
		HostNodeName: pod.Spec.NodeName, TaskResultSynced: node.TaskResultSynced,
	}
	setPodCompletionMetadata(pod, expected)
	if expected.FinishedAt.Before(&node.StartedAt) || expected.FinishedAt.After(wf.Status.FinishedAt.Time) || !reflect.DeepEqual(expected, &node) {
		return nil, hold(conflict, "Pod-derived terminal result differs from the completed legacy node")
	}
	return expected, nil
}

func simpleLegacyTemplate(tmpl *wfv1.Template) bool {
	return tmpl.Container != nil && tmpl.GetType() == wfv1.TemplateTypeContainer && tmpl.ContainerSet == nil && tmpl.Script == nil && tmpl.Resource == nil && tmpl.DAG == nil && len(tmpl.Steps) == 0 && tmpl.Suspend == nil && tmpl.Data == nil && tmpl.HTTP == nil && tmpl.Plugin == nil && tmpl.Daemon == nil && tmpl.Memoize == nil && tmpl.RetryStrategy == nil && tmpl.ActiveDeadlineSeconds == nil && !tmpl.Inputs.HasInputs() && !tmpl.Outputs.HasOutputs() && tmpl.Outputs.ExitCode == nil && len(tmpl.Sidecars) == 0 && len(tmpl.InitContainers) == 0 && tmpl.PodSpecPatch == "" && !tmpl.SaveLogsAsArtifact()
}

func legacyTemplateScopeMatches(wf *wfv1.Workflow, scope string) bool {
	if scope == "local/"+wf.Name {
		return true
	}
	ref := wf.Spec.WorkflowTemplateRef
	if wf.Status.StoredWorkflowSpec == nil || ref == nil {
		return false
	}
	expected := "namespaced/" + ref.Name
	if ref.ClusterScope {
		expected = "cluster/" + ref.Name
	}
	return scope == expected
}

func sameLegacyTemplate(stored, executed *wfv1.Template) bool {
	a, b := stored.DeepCopy(), executed.DeepCopy()
	// A default archive location without log capture is not a result channel.
	a.ArchiveLocation, b.ArchiveLocation = nil, nil
	return reflect.DeepEqual(a, b)
}

func successfulContainerStatuses(containers []apiv1.Container, statuses []apiv1.ContainerStatus) bool {
	if len(containers) != len(statuses) {
		return false
	}
	seen := make(map[string]bool, len(statuses))
	for _, c := range containers {
		found := false
		for _, status := range statuses {
			if status.Name != c.Name {
				continue
			}
			term := status.State.Terminated
			if seen[c.Name] || status.RestartCount != 0 || status.State.Running != nil || status.State.Waiting != nil || term == nil || term.ExitCode != 0 || term.StartedAt.IsZero() || term.FinishedAt.IsZero() || term.FinishedAt.Before(&term.StartedAt) {
				return false
			}
			seen[c.Name], found = true, true
		}
		if !found {
			return false
		}
	}
	return true
}

// recaptureLegacyPod is a commit-only transaction. Update carries the exact
// Workflow UID/resourceVersion; no merge or ordinary operation side effects are
// permitted. An uncertain response retains the barrier until a fresh read.
func (wfc *WorkflowController) recaptureLegacyPod(ctx context.Context, observed *apiv1.Pod) (*wfv1.Workflow, error) {
	pod, err := wfc.kubeclientset.CoreV1().Pods(observed.Namespace).Get(ctx, observed.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if pod.UID != observed.UID || !reflect.DeepEqual(pod.OwnerReferences, observed.OwnerReferences) {
		return nil, podcontroller.StatusCaptureHold(podcontroller.CaptureIdentityConflict, "Pod changed before legacy capture")
	}
	owner := metav1.GetControllerOf(pod)
	if owner == nil {
		return nil, podcontroller.StatusCaptureHold(podcontroller.CaptureIdentityConflict, "legacy Pod owner is unavailable")
	}
	wf, err := wfc.lookupWorkflowForPodCleanup(ctx, pod.Namespace, owner.Name, true)
	if err != nil {
		return nil, err
	}
	node, err := legacySuccessfulNode(wf, pod)
	if err != nil {
		return nil, err
	}
	original := wf.DeepCopy()
	node.CapturedPodUID = string(pod.UID)
	wf.Status.Nodes[node.ID] = *node
	if saveErr := wfc.hydrator.Dehydrate(ctx, wf); saveErr != nil {
		return nil, saveErr
	}
	if _, updateErr := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, wf, metav1.UpdateOptions{}); updateErr != nil {
		return nil, updateErr
	}
	readback, err := wfc.lookupWorkflowForPodCleanup(ctx, wf.Namespace, wf.Name, true)
	if err != nil {
		return nil, err
	}
	got, exists := readback.Status.Nodes[node.ID]
	if readback.UID != original.UID || !exists || got.CapturedPodUID != string(pod.UID) {
		return nil, podcontroller.StatusCaptureHold(podcontroller.CaptureReceiptNotRetained, "legacy capture receipt is absent from authoritative readback; verify the installed CRD")
	}
	// Encoding/reference may change, but every hydrated result field must survive.
	verify := readback.DeepCopy()
	got.CapturedPodUID = ""
	verify.Status.Nodes[node.ID] = got
	// The API may advance bookkeeping fields for this write. User-controlled
	// metadata, deletion state and completion labels must otherwise remain exact.
	verify.ResourceVersion = original.ResourceVersion
	verify.Generation = original.Generation
	verify.ManagedFields = original.ManagedFields
	if !reflect.DeepEqual(original.Status, verify.Status) || !reflect.DeepEqual(original.Spec, verify.Spec) || !reflect.DeepEqual(original.ObjectMeta, verify.ObjectMeta) {
		return nil, podcontroller.StatusCaptureHold(podcontroller.CaptureResultConflict, "legacy capture readback changed the persisted result")
	}
	return readback, nil
}
