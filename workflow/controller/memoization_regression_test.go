package controller

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

const memoizedInlineWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: inline-memo
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: cacheAge
      value: 1h
  templates:
  - name: main
    steps:
    - - name: a
        inline:
          memoize:
            key: k
            maxAge: "{{workflow.parameters.cacheAge}}"
            cache:
              configMap:
                name: inline-cache
          container:
            image: alpine
            command: [echo, hello]
`

func TestMemoizedInlineTemplateSucceeds(t *testing.T) {
	for _, variant := range []string{"configmap-inline-steps", "sql-inline-steps", "configmap-inline-dag", "sql-inline-dag", "configmap-named-steps", "sql-named-steps"} {
		t.Run(variant, func(t *testing.T) {
			backend, _, _ := strings.Cut(variant, "-")
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx)
			defer cancel()
			saved := false
			if backend == "sql" {
				controller.cacheFactory.SetQueries(ctx, &recordingMemoizationDB{
					load: func(string, string, string) (*memodb.CacheRecord, error) { return nil, nil },
					save: func(_, _, _, _ string, _ *wfv1.Outputs, age string) error {
						saved = true
						assert.Equal(t, "1h", age)
						return nil
					},
				})
			}
			wf := wfv1.MustUnmarshalWorkflow(memoizedInlineWorkflow)
			step := &wf.Spec.Templates[0].Steps[0].Steps[0]
			if strings.Contains(variant, "named") {
				tmpl := step.Inline.DeepCopy()
				tmpl.Name = "memoized"
				step.Template, step.Inline = tmpl.Name, nil
				wf.Spec.Templates = append(wf.Spec.Templates, *tmpl)
			}
			if strings.HasSuffix(variant, "dag") {
				wf.Spec.Templates[0].DAG = &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: step.Name, Inline: step.Inline}}}
				wf.Spec.Templates[0].Steps = nil
			}
			_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
			require.NoError(t, err)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			started := woc.wf.Status.Nodes.FindByDisplayName("a")
			require.NotNil(t, started)
			require.NotNil(t, started.MemoizationStatus)
			assert.Equal(t, "1h", started.MemoizationStatus.MaxAge)
			makePodsPhase(ctx, woc, apiv1.PodSucceeded)
			// Completion may be handled by another controller process after a restart.
			serialized, err := json.Marshal(woc.wf)
			require.NoError(t, err)
			var restored wfv1.Workflow
			require.NoError(t, json.Unmarshal(serialized, &restored))
			restored.Spec.Arguments.Parameters[0].Value = wfv1.AnyStringPtr("2h")
			woc = newWorkflowOperationCtx(ctx, &restored, controller)
			woc.operate(ctx)
			node := woc.wf.Status.Nodes.FindByDisplayName("a")
			require.NotNil(t, node)
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase, node.Message)
			assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
			if backend == "sql" {
				assert.True(t, saved)
			} else {
				cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "inline-cache", metav1.GetOptions{})
				require.NoError(t, err)
				assert.NotEmpty(t, cm.Data["k"])
			}
		})
	}
}

func TestConfigMapMemoizationTaskSetSaveFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/taskset/reconcile-task-set-with-memoization-workflow.yaml")
	var tasks wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal("@testdata/taskset/reconcile-task-set-with-memoization-taskset.yaml", &tasks)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	require.NoError(t, controller.wfTaskSetInformer.Informer().GetIndexer().Add(&tasks))
	controller.kubeclientset.(*fake.Clientset).PrependReactor("create", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "cache-demo-1", errors.New("write denied"))
	})
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	require.NoError(t, woc.reconcileTaskSet(ctx))
	for _, node := range woc.wf.Status.Nodes {
		assert.Equal(t, wfv1.NodeError, node.Phase)
		assert.Contains(t, node.Message, "write denied")
	}
}
