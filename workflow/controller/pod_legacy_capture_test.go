package controller

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	sqldbmocks "github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
)

func legacyCaptureFixture(t *testing.T) (*wfv1.Workflow, *apiv1.Pod) {
	t.Helper()
	wf, pod := captureWriterFixture()
	wf.Status.Phase = wfv1.WorkflowSucceeded
	wf.Status.FinishedAt = metav1.NewTime(time.Unix(1003, 0))
	wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
	pod.CreationTimestamp = metav1.NewTime(time.Unix(1000, 0))
	pod.Spec.NodeName = "worker"
	tmplJSON, err := json.Marshal(wf.Spec.Templates[0])
	require.NoError(t, err)
	pod.Spec.Containers = []apiv1.Container{{Name: common.MainContainerName}, {Name: common.WaitContainerName, Env: []apiv1.EnvVar{{Name: common.EnvVarTemplate, Value: string(tmplJSON)}}}}
	node := wf.Status.Nodes[wf.Name]
	node.Phase, node.DisplayName, node.TemplateScope = wfv1.NodeSucceeded, wf.Name, "local/"+wf.Name
	node.Outputs = &wfv1.Outputs{ExitCode: new("0")}
	node.HostNodeName, node.Progress = "worker", "1/1"
	setPodCompletionMetadata(pod, &node)
	wf.Status.Nodes[wf.Name] = node
	return wf, pod
}

func TestLegacyCapturePrototype(t *testing.T) {
	wf, pod := legacyCaptureFixture(t)
	before := wf.DeepCopy()
	result, err := legacySuccessfulNode(wf, pod)
	require.NoError(t, err)
	assert.Equal(t, wf.Status.Nodes[wf.Name], *result)
	assert.Equal(t, before, wf, "pure verifier must not mutate old result")
	// Compare to the existing assessor, followed by the same normal completion
	// progress step; assessNodeStatus itself is deliberately not used as verifier.
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	running := wf.Status.Nodes[wf.Name]
	running.Phase, running.Progress, running.FinishedAt, running.ResourcesDuration, running.Outputs = wfv1.NodeRunning, "0/1", metav1.Time{}, nil, nil
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assessed := woc.assessNodeStatus(ctx, pod, &running)
	require.NotNil(t, assessed)
	assessed.CapturedPodUID, assessed.Progress = "", assessed.Progress.Complete()
	assert.Equal(t, result, assessed, "normal and recovery interpretation agree")
}

func TestLegacyCaptureUnsupportedAndMismatch(t *testing.T) {
	for name, mutate := range map[string]func(*wfv1.Workflow, *apiv1.Pod){
		"memoization transformed Error": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.Phase = wfv1.NodeError
			n.Message = "cache Save failed"
			n.MemoizationStatus = &wfv1.MemoizationStatus{CacheName: "restored-cache"}
			w.Status.Nodes[w.Name] = n
			w.Status.Phase = wfv1.WorkflowError
		},
		"template lookup unavailable": func(w *wfv1.Workflow, _ *apiv1.Pod) { w.Spec.Templates = nil },
		"foreign owner":               func(_ *wfv1.Workflow, p *apiv1.Pod) { p.OwnerReferences[0].UID = "foreign" },
		"identity":                    func(_ *wfv1.Workflow, p *apiv1.Pod) { p.Annotations[common.AnnotationKeyNodeID] = "foreign" },
		"message": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.Message = "previous result"
			w.Status.Nodes[w.Name] = n
		},
		"output": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.Outputs.Result = new("external")
			w.Status.Nodes[w.Name] = n
		},
		"task sync absent": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.TaskResultSynced = nil
			w.Status.Nodes[w.Name] = n
		},
		"task sync false": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			w.Status.TaskResultsCompletionStatus = map[string]bool{w.Name: false}
		},
		"finished mismatch": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.FinishedAt = w.Status.FinishedAt
			w.Status.Nodes[w.Name] = n
		},
		"resources mismatch": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.ResourcesDuration = nil
			w.Status.Nodes[w.Name] = n
		},
		"host mismatch": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.HostNodeName = "other"
			w.Status.Nodes[w.Name] = n
		},
		"progress mismatch": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.Progress = "2/2"
			w.Status.Nodes[w.Name] = n
		},
		"missing status": func(_ *wfv1.Workflow, p *apiv1.Pod) { p.Status.ContainerStatuses = p.Status.ContainerStatuses[:1] },
		"invalid simultaneous state": func(_ *wfv1.Workflow, p *apiv1.Pod) {
			p.Status.ContainerStatuses[1].State.Running = &apiv1.ContainerStateRunning{}
		},
		"invalid stored scope": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			w.Status.StoredWorkflowSpec = w.Spec.DeepCopy()
			n := w.Status.Nodes[w.Name]
			n.TemplateScope = "cluster/unrelated"
			w.Status.Nodes[w.Name] = n
		},
		"workflow deadline": func(w *wfv1.Workflow, _ *apiv1.Pod) { w.Spec.ActiveDeadlineSeconds = new(int64(100)) },
		"nonzero wait":      func(_ *wfv1.Workflow, p *apiv1.Pod) { p.Status.ContainerStatuses[1].State.Terminated.ExitCode = 1 },
		"template mutated":  func(w *wfv1.Workflow, _ *apiv1.Pod) { w.Spec.Templates[0].Container.Image = "new-definition" },
		"template absent":   func(_ *wfv1.Workflow, p *apiv1.Pod) { p.Spec.Containers[1].Env = nil },
		"stop":              func(w *wfv1.Workflow, _ *apiv1.Pod) { w.Spec.Shutdown = wfv1.ShutdownStrategyStop },
		"restart": func(w *wfv1.Workflow, _ *apiv1.Pod) {
			n := w.Status.Nodes[w.Name]
			n.FailedPodRestarts = 1
			w.Status.Nodes[w.Name] = n
		},
	} {
		t.Run(name, func(t *testing.T) {
			wf, pod := legacyCaptureFixture(t)
			mutate(wf, pod)
			before := wf.DeepCopy()
			_, err := legacySuccessfulNode(wf, pod)
			require.Error(t, err)
			assert.Equal(t, before, wf)
		})
	}
	t.Run("legacy explicit map", func(t *testing.T) {
		wf, pod := legacyCaptureFixture(t)
		n := wf.Status.Nodes[wf.Name]
		n.TaskResultSynced = nil
		wf.Status.Nodes[wf.Name] = n
		wf.Status.TaskResultsCompletionStatus = map[string]bool{wf.Name: true}
		_, err := legacySuccessfulNode(wf, pod)
		require.NoError(t, err)
	})
	// The controller adds the artifact repository's archive location to the
	// executed template, so the stored template stays ordinary either way.
	t.Run("archive location without log archiving", func(t *testing.T) {
		wf, pod := legacyCaptureFixture(t)
		executed := wf.Spec.Templates[0].DeepCopy()
		executed.ArchiveLocation = &wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{Key: "archive"}}
		tmplJSON, err := json.Marshal(executed)
		require.NoError(t, err)
		pod.Spec.Containers[1].Env[0].Value = string(tmplJSON)
		_, err = legacySuccessfulNode(wf, pod)
		require.NoError(t, err)
	})
	t.Run("archived logs are unsupported, not a conflict", func(t *testing.T) {
		wf, pod := legacyCaptureFixture(t)
		executed := wf.Spec.Templates[0].DeepCopy()
		executed.ArchiveLocation = &wfv1.ArtifactLocation{ArchiveLogs: new(true)}
		tmplJSON, err := json.Marshal(executed)
		require.NoError(t, err)
		pod.Spec.Containers[1].Env[0].Value = string(tmplJSON)
		_, err = legacySuccessfulNode(wf, pod)
		require.EqualError(t, err, "Pod execution template has external results, such as archived logs")
	})
}

func TestLegacyCaptureCommit(t *testing.T) {
	for _, encoding := range []string{"raw", "compressed", "offload"} {
		t.Run(encoding, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := legacyCaptureFixture(t)
			original := wf.DeepCopy()
			repo := &sqldbmocks.OffloadNodeStatusRepo{}
			if encoding == "compressed" {
				nodesJSON, err := json.Marshal(wf.Status.Nodes)
				require.NoError(t, err)
				wf.Status.CompressedNodes = file.CompressEncodeString(ctx, string(nodesJSON))
				wf.Status.Nodes = nil
			}
			if encoding == "offload" {
				t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "true")
				wf.Status.Nodes = nil
				wf.Status.OffloadNodeStatusVersion = "old-version"
				repo.On("Get", string(wf.UID), "old-version").Return(original.Status.Nodes.DeepCopy(), nil).Once()
			}
			cancel, c := newController(ctx, wf)
			defer cancel()
			_, err := c.kubeclientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
			c.hydrator = hydrator.New(repo)
			if encoding == "offload" {
				var saved wfv1.Nodes
				repo.On("Save", string(wf.UID), wf.Namespace, mock.Anything).Run(func(args mock.Arguments) { saved = args.Get(2).(wfv1.Nodes).DeepCopy() }).Return("new-version", nil).Once()
				repo.On("Get", string(wf.UID), "new-version").Return(func(string, string) wfv1.Nodes { return saved.DeepCopy() }, nil).Once()
			}
			got, err := c.recaptureLegacyPod(ctx, pod)
			require.NoError(t, err)
			node := got.Status.Nodes[wf.Name]
			assert.Equal(t, string(pod.UID), node.CapturedPodUID)
			node.CapturedPodUID = ""
			got.Status.Nodes[wf.Name] = node
			assert.Equal(t, original.Status, got.Status)
			assert.Equal(t, original.Spec, got.Spec)
			repo.AssertExpectations(t)
		})
	}
}

func TestLegacyCaptureCommitFailures(t *testing.T) {
	for _, failure := range []string{"conflict", "unknown-before-write", "lost-reply-after-write", "pruned-receipt", "readback-error", "concurrent-result", "concurrent-label", "concurrent-deletion", "replacement"} {
		t.Run(failure, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := legacyCaptureFixture(t)
			cancel, c := newController(ctx, wf)
			defer cancel()
			_, err := c.kubeclientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
			client := c.wfclientset.(*fakewfclientset.Clientset)
			updates := 0
			reads := 0
			client.PrependReactor("get", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
				reads++
				if failure == "readback-error" && reads > 1 {
					return true, nil, fmt.Errorf("API unavailable")
				}
				return false, nil, nil
			})
			client.PrependReactor("update", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updates++
				next := action.(k8stesting.UpdateAction).GetObject().(*wfv1.Workflow)
				require.Equal(t, wf.UID, next.UID)
				require.Equal(t, wf.ResourceVersion, next.ResourceVersion)
				switch failure {
				case "conflict":
					return true, nil, apierr.NewConflict(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("concurrent write"))
				case "unknown-before-write":
					return true, nil, fmt.Errorf("request timeout")
				case "lost-reply-after-write":
					require.NoError(t, client.Tracker().Update(wfv1.SchemeGroupVersion.WithResource("workflows"), next, wf.Namespace))
					return true, nil, fmt.Errorf("response lost")
				case "pruned-receipt":
					n := next.Status.Nodes[wf.Name]
					n.CapturedPodUID = ""
					next.Status.Nodes[wf.Name] = n
				case "concurrent-label":
					next.Labels[common.LabelKeyCompleted] = "false"
				case "concurrent-deletion":
					next.DeletionTimestamp = &metav1.Time{Time: time.Unix(1004, 0)}
				case "concurrent-result":
					n := next.Status.Nodes[wf.Name]
					n.Message = "concurrent result"
					next.Status.Nodes[wf.Name] = n
				}
				return false, nil, nil
			})
			if failure == "replacement" {
				pod = pod.DeepCopy()
				pod.UID = "stale"
			}
			_, err = c.recaptureLegacyPod(ctx, pod)
			require.Error(t, err)
			if failure == "replacement" {
				assert.Zero(t, updates)
			} else {
				assert.Equal(t, 1, updates)
			}
			if failure == "lost-reply-after-write" {
				persisted, err := client.Tracker().Get(wfv1.SchemeGroupVersion.WithResource("workflows"), wf.Namespace, wf.Name)
				require.NoError(t, err)
				assert.Equal(t, string(pod.UID), persisted.(*wfv1.Workflow).Status.Nodes[wf.Name].CapturedPodUID, "a later reader can observe the committed receipt, but this attempt returned an error")
			}
			for _, a := range c.kubeclientset.(*fake.Clientset).Actions() {
				assert.NotEqual(t, "patch", a.GetVerb())
				assert.NotEqual(t, "delete", a.GetVerb())
			}
		})
	}
}

func TestLegacyCaptureOffloadCannotPublishOnFailure(t *testing.T) {
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "true")
	for _, failure := range []string{"save", "reference-conflict"} {
		t.Run(failure, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := legacyCaptureFixture(t)
			originalNodes := wf.Status.Nodes.DeepCopy()
			wf.Status.Nodes = nil
			wf.Status.OffloadNodeStatusVersion = "old-version"
			cancel, c := newController(ctx, wf)
			defer cancel()
			_, err := c.kubeclientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
			repo := &sqldbmocks.OffloadNodeStatusRepo{}
			repo.On("Get", string(wf.UID), "old-version").Return(originalNodes, nil).Once()
			var saveError error
			if failure == "save" {
				saveError = fmt.Errorf("SQL write unavailable")
			}
			repo.On("Save", string(wf.UID), wf.Namespace, mock.Anything).Return("unpublished-version", saveError).Once()
			c.hydrator = hydrator.New(repo)
			client := c.wfclientset.(*fakewfclientset.Clientset)
			updates := 0
			client.PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
				updates++
				return true, nil, apierr.NewConflict(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("concurrent update"))
			})
			_, err = c.recaptureLegacyPod(ctx, pod)
			require.Error(t, err)
			if failure == "save" {
				assert.Zero(t, updates)
			} else {
				assert.Equal(t, 1, updates)
			}
			persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, "old-version", persisted.Status.OffloadNodeStatusVersion)
			assert.Empty(t, persisted.Status.Nodes)
			repo.AssertExpectations(t)
		})
	}
}

func TestLegacyCaptureMemoizationErrorDoesNotReplay(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := legacyCaptureFixture(t)
	n := wf.Status.Nodes[wf.Name]
	n.Phase = wfv1.NodeError
	n.Message = "failed to save memoization cache"
	n.MemoizationStatus = &wfv1.MemoizationStatus{CacheName: "restored-cache"}
	wf.Status.Nodes[wf.Name] = n
	wf.Status.Phase = wfv1.WorkflowError
	cancel, c := newController(ctx, wf)
	defer cancel()
	kube := c.kubeclientset.(*fake.Clientset)
	_, err := kube.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = kube.CoreV1().ConfigMaps(pod.Namespace).Create(ctx, &apiv1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "restored-cache"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	kube.ClearActions()
	client := c.wfclientset.(*fakewfclientset.Clientset)
	client.ClearActions()
	_, err = c.recaptureLegacyPod(ctx, pod)
	require.Error(t, err)
	for _, a := range kube.Actions() {
		assert.Equal(t, "get", a.GetVerb())
		assert.Equal(t, "pods", a.GetResource().Resource, "no cache read or Save")
	}
	for _, a := range client.Actions() {
		assert.NotEqual(t, "update", a.GetVerb())
	}
	persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wf.Status, persisted.Status)
}

func TestLegacyCaptureInitlessObservation(t *testing.T) {
	for _, incompleteSupervisor := range []bool{false, true} {
		t.Run(fmt.Sprintf("incompleteSupervisor=%v", incompleteSupervisor), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := legacyCaptureFixture(t)
			// The init-less layout carries the execution template on supervisor,
			// which performs the same result collection as the legacy wait sidecar.
			pod.Spec.Containers[1].Name = common.SupervisorContainerName
			pod.Status.ContainerStatuses[1].Name = common.SupervisorContainerName
			require.Empty(t, pod.Spec.InitContainers)
			require.Empty(t, pod.Status.InitContainerStatuses)
			before := wf.DeepCopy()
			if incompleteSupervisor {
				pod.Status.ContainerStatuses[1].State = apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.NewTime(time.Unix(1000, 0))}}
				_, err := legacySuccessfulNode(wf, pod)
				require.Error(t, err)
				assert.Equal(t, before, wf)
				return
			}
			result, err := legacySuccessfulNode(wf, pod)
			require.NoError(t, err)
			assert.Equal(t, wf.Status.Nodes[wf.Name], *result)
			assert.Equal(t, before, wf)
			cancel, controller := newController(ctx, wf)
			defer cancel()
			running := wf.Status.Nodes[wf.Name]
			running.Phase, running.Progress, running.FinishedAt, running.ResourcesDuration, running.Outputs = wfv1.NodeRunning, "0/1", metav1.Time{}, nil, nil
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			assessed := woc.assessNodeStatus(ctx, pod, &running)
			require.NotNil(t, assessed)
			assessed.CapturedPodUID, assessed.Progress = "", assessed.Progress.Complete()
			assert.Equal(t, result, assessed, "init-less normal and recovery interpretation agree")
		})
	}
}
