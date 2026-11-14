package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type workflowCleanupHandlerRecorder struct {
	cache.SharedIndexInformer
	handlers []cache.ResourceEventHandler
}

func (recorder *workflowCleanupHandlerRecorder) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	recorder.handlers = append(recorder.handlers, handler)
	return nil, nil
}

func TestWorkflowCallbackPodFinalizerCleanup(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
	for _, event := range []string{"completed", "deleted"} {
		for _, instanceID := range []string{"", "instance-a"} {
			for _, scenario := range []string{"matching", "other namespace", "other instance", "empty instance label", "unowned", "other workflow UID", "other owner name", "other owner kind", "other owner API", "other workflow label", "other workflow instance", "list error"} {
				t.Run(event+"/"+instanceID+"/"+scenario, func(t *testing.T) {
					ctx := logging.TestContext(t.Context())
					cancel, controller := newController(ctx, func(controller *WorkflowController) { controller.Config.InstanceID = instanceID })
					defer cancel()
					workflow := &unstructured.Unstructured{}
					workflow.SetAPIVersion(wfv1.SchemeGroupVersion.String())
					workflow.SetKind("Workflow")
					workflow.SetNamespace("workflow-namespace")
					workflow.SetName("same-name")
					workflow.SetUID("workflow-uid")
					workflow.SetResourceVersion("1")
					workflowLabels := map[string]string{}
					podLabels := map[string]string{common.LabelKeyWorkflow: workflow.GetName(), common.LabelKeyCompleted: "true"}
					if instanceID != "" {
						workflowLabels[common.LabelKeyControllerInstanceID] = instanceID
						podLabels[common.LabelKeyControllerInstanceID] = instanceID
					}
					workflow.SetLabels(workflowLabels)
					target := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{
						Namespace: workflow.GetNamespace(), Name: "completed-pod", UID: "pod-uid", ResourceVersion: "123",
						Labels: podLabels, Finalizers: []string{common.FinalizerPodStatus, "example.com/retain"},
						OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(workflow, wfv1.SchemeGroupVersion.WithKind("Workflow"))},
					}}
					switch scenario {
					case "other namespace":
						target.Namespace = "other-namespace"
					case "other instance":
						target.Labels[common.LabelKeyControllerInstanceID] = "instance-b"
					case "empty instance label":
						target.Labels[common.LabelKeyControllerInstanceID] = ""
					case "unowned":
						target.OwnerReferences = nil
					case "other workflow UID":
						target.OwnerReferences[0].UID = "replacement-workflow-uid"
					case "other owner name":
						target.OwnerReferences[0].Name = "different-name"
					case "other owner kind":
						target.OwnerReferences[0].Kind = "Job"
					case "other owner API":
						target.OwnerReferences[0].APIVersion = "example.com/v1alpha1"
					case "other workflow label":
						target.Labels[common.LabelKeyWorkflow] = "different-name"
					case "other workflow instance":
						workflowLabels[common.LabelKeyControllerInstanceID] = "instance-b"
						workflow.SetLabels(workflowLabels)
					}
					client := controller.kubeclientset.(*fake.Clientset)
					require.NoError(t, client.Tracker().Create(apiv1.SchemeGroupVersion.WithResource("pods"), target, target.Namespace))
					listCount := 0
					client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
						listCount++
						assert.Equal(t, workflow.GetNamespace(), action.GetNamespace())
						listAction := action.(k8stesting.ListAction)
						selector := listAction.GetListRestrictions().Labels
						expectedLabels := labels.Set{common.LabelKeyWorkflow: workflow.GetName()}
						if instanceID != "" {
							expectedLabels[common.LabelKeyControllerInstanceID] = instanceID
						}
						assert.True(t, selector.Matches(expectedLabels))
						otherInstance := labels.Set{common.LabelKeyWorkflow: workflow.GetName(), common.LabelKeyControllerInstanceID: "instance-b"}
						assert.False(t, selector.Matches(otherInstance))
						assert.Positive(t, action.(k8stesting.ListActionImpl).GetListOptions().Limit)
						if scenario == "list error" {
							return true, nil, assert.AnError
						}
						return true, &apiv1.PodList{Items: []apiv1.Pod{*target.DeepCopy()}}, nil
					})
					recorder := &workflowCleanupHandlerRecorder{SharedIndexInformer: controller.wfInformer}
					controller.wfInformer = recorder
					require.NoError(t, controller.addWorkflowInformerHandlers(ctx))
					completed := workflow.DeepCopy()
					completedLabels := completed.GetLabels()
					completedLabels[common.LabelKeyCompleted] = "true"
					completed.SetLabels(completedLabels)
					completed.SetResourceVersion("2")
					if event == "completed" {
						recorder.handlers[0].OnUpdate(workflow, completed)
					} else {
						recorder.handlers[0].OnDelete(workflow)
					}
					key := target.Namespace + "/" + target.Name + "/removeFinalizer/" + string(target.UID)
					if scenario == "matching" {
						require.Equal(t, 1, controller.PodController.TestingQueueNumRequeues(key))
						require.True(t, controller.PodController.TestingProcessNextItem(ctx))
					} else {
						assert.Zero(t, controller.PodController.TestingQueueNumRequeues(key))
					}
					if scenario == "other workflow instance" {
						assert.Zero(t, listCount)
					} else {
						assert.Equal(t, 1, listCount)
					}
					updated, err := client.CoreV1().Pods(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
					require.NoError(t, err)
					if scenario == "matching" {
						assert.Equal(t, []string{"example.com/retain"}, updated.Finalizers)
					} else {
						assert.Equal(t, target.Finalizers, updated.Finalizers)
					}
				})
			}
		}
	}
}

func TestWorkflowCallbackPodFinalizerCleanupPagination(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
	for _, event := range []string{"completed", "deleted"} {
		for _, instanceID := range []string{"", "instance-a"} {
			t.Run(event+"/"+instanceID, func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				cancel, controller := newController(ctx, func(controller *WorkflowController) { controller.Config.InstanceID = instanceID })
				defer cancel()
				workflow := &unstructured.Unstructured{}
				workflow.SetNamespace("workflow-namespace")
				workflow.SetName("large-workflow")
				workflow.SetUID("workflow-uid")
				workflow.SetResourceVersion("1")
				workflowLabels := map[string]string{}
				podLabels := map[string]string{common.LabelKeyWorkflow: workflow.GetName(), common.LabelKeyCompleted: "true"}
				if instanceID != "" {
					workflowLabels[common.LabelKeyControllerInstanceID] = instanceID
					podLabels[common.LabelKeyControllerInstanceID] = instanceID
				}
				workflow.SetLabels(workflowLabels)
				client := controller.kubeclientset.(*fake.Clientset)
				targets := make([]apiv1.Pod, 501)
				for podIndex := range targets {
					name := fmt.Sprintf("completed-pod-%03d", podIndex)
					targets[podIndex] = apiv1.Pod{ObjectMeta: metav1.ObjectMeta{
						Namespace: workflow.GetNamespace(), Name: name, UID: types.UID(name + "-uid"), ResourceVersion: "123",
						Labels: podLabels, Finalizers: []string{common.FinalizerPodStatus, "example.com/retain"},
						OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(workflow, wfv1.SchemeGroupVersion.WithKind("Workflow"))},
					}}
					require.NoError(t, client.Tracker().Create(apiv1.SchemeGroupVersion.WithResource("pods"), &targets[podIndex], workflow.GetNamespace()))
				}
				unrelated := targets[500].DeepCopy()
				unrelated.Name = "unrelated-pod"
				unrelated.UID = "unrelated-pod-uid"
				unrelated.OwnerReferences[0].UID = "other-workflow-uid"
				require.NoError(t, client.Tracker().Create(apiv1.SchemeGroupVersion.WithResource("pods"), unrelated, workflow.GetNamespace()))
				var continueTokens []string
				var labelSelector string
				client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
					require.Equal(t, workflow.GetNamespace(), action.GetNamespace())
					options := action.(k8stesting.ListActionImpl).GetListOptions()
					require.Equal(t, int64(500), options.Limit)
					continueTokens = append(continueTokens, options.Continue)
					if labelSelector == "" {
						labelSelector = options.LabelSelector
					}
					require.Equal(t, labelSelector, options.LabelSelector)
					selector := action.(k8stesting.ListAction).GetListRestrictions().Labels
					require.True(t, selector.Matches(labels.Set(podLabels)))
					require.False(t, selector.Matches(labels.Set{common.LabelKeyWorkflow: workflow.GetName(), common.LabelKeyControllerInstanceID: "instance-b"}))
					require.False(t, selector.Matches(labels.Set{common.LabelKeyWorkflow: "other-workflow", common.LabelKeyControllerInstanceID: instanceID}))
					switch options.Continue {
					case "":
						return true, &apiv1.PodList{ListMeta: metav1.ListMeta{Continue: "second-page"}, Items: targets[:500]}, nil
					case "second-page":
						return true, &apiv1.PodList{Items: []apiv1.Pod{targets[500], *unrelated}}, nil
					default:
						t.Fatalf("unexpected continuation token: %q", options.Continue)
						return true, nil, assert.AnError
					}
				})
				recorder := &workflowCleanupHandlerRecorder{SharedIndexInformer: controller.wfInformer}
				controller.wfInformer = recorder
				require.NoError(t, controller.addWorkflowInformerHandlers(ctx))
				if event == "completed" {
					completed := workflow.DeepCopy()
					completedLabels := completed.GetLabels()
					completedLabels[common.LabelKeyCompleted] = "true"
					completed.SetLabels(completedLabels)
					completed.SetResourceVersion("2")
					recorder.handlers[0].OnUpdate(workflow, completed)
				} else {
					recorder.handlers[0].OnDelete(workflow)
				}
				require.Equal(t, []string{"", "second-page"}, continueTokens)
				for _, target := range targets {
					key := target.Namespace + "/" + target.Name + "/removeFinalizer/" + string(target.UID)
					require.Equal(t, 1, controller.PodController.TestingQueueNumRequeues(key))
				}
				assert.Zero(t, controller.PodController.TestingQueueNumRequeues(unrelated.Namespace+"/"+unrelated.Name+"/removeFinalizer/"+string(unrelated.UID)))
				updated, err := client.CoreV1().Pods(unrelated.Namespace).Get(ctx, unrelated.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, unrelated.Finalizers, updated.Finalizers)
			})
		}
	}
}

func TestWorkflowPodFinalizerCleanupPaginationError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	workflow := &unstructured.Unstructured{}
	workflow.SetNamespace("workflow-namespace")
	workflow.SetName("workflow")
	workflow.SetUID("workflow-uid")
	client := controller.kubeclientset.(*fake.Clientset)
	var continueTokens []string
	client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		options := action.(k8stesting.ListActionImpl).GetListOptions()
		continueTokens = append(continueTokens, options.Continue)
		if options.Continue == "" {
			return true, &apiv1.PodList{ListMeta: metav1.ListMeta{Continue: "second-page"}}, nil
		}
		return true, nil, assert.AnError
	})
	require.ErrorIs(t, controller.removeWorkflowPodFinalizers(ctx, workflow), assert.AnError)
	assert.Equal(t, []string{"", "second-page"}, continueTokens)
}

func TestWorkflowCallbackDoesNotScanCompletedWorkflowAtStartup(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	client := controller.kubeclientset.(*fake.Clientset)
	listCount := 0
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		listCount++
		return true, &apiv1.PodList{}, nil
	})
	recorder := &workflowCleanupHandlerRecorder{SharedIndexInformer: controller.wfInformer}
	controller.wfInformer = recorder
	require.NoError(t, controller.addWorkflowInformerHandlers(ctx))
	workflow := &unstructured.Unstructured{}
	workflow.SetNamespace("workflow-namespace")
	workflow.SetName("historical-workflow")
	workflow.SetUID("historical-workflow-uid")
	workflow.SetLabels(map[string]string{common.LabelKeyCompleted: "true"})
	recorder.handlers[0].OnAdd(workflow, true)
	recorder.handlers[0].OnDelete(workflow)
	assert.Zero(t, listCount)
}
