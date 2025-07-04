package estimation

import (
	"context"
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	sqldbmocks "github.com/argoproj/argo-workflows/v3/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	testutil "github.com/argoproj/argo-workflows/v3/test/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	hydratorfake "github.com/argoproj/argo-workflows/v3/workflow/hydrator/fake"
)

func Test_estimatorFactory(t *testing.T) {
	informer := testutil.NewSharedIndexInformer()
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	wfFailed := testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: bad-baseline
  labels:
    workflows.argoproj.io/phase: Failed
`)
	informer.Indexer.SetByIndex(indexes.ClusterWorkflowTemplateIndex, "my-ns/my-cwft", testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-cwft-baseline
  labels:
    workflows.argoproj.io/phase: Succeeded
`), wfFailed)
	informer.Indexer.SetByIndex(indexes.CronWorkflowIndex, "my-ns/my-cwf", testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-cwf-baseline
  labels:
    workflows.argoproj.io/phase: Succeeded
`), wfFailed)
	informer.Indexer.SetByIndex(indexes.WorkflowTemplateIndex, "my-ns/my-wftmpl", testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-wftmpl-baseline
  labels:
    workflows.argoproj.io/phase: Succeeded
`), wfFailed)
	wfArchive := &sqldbmocks.WorkflowArchive{}
	r, err := labels.ParseToRequirements("workflows.argoproj.io/workflow-template=my-archived-wftmpl")
	require.NoError(t, err)
	wfArchive.On("GetWorkflowForEstimator", mock.Anything, "my-ns", r).Return(testutil.MustUnmarshalWorkflow(`
metadata:
  name: my-archived-wftmpl-baseline`), nil)
	f := NewEstimatorFactory(ctx, informer, hydratorfake.Always, wfArchive)
	t.Run("None", func(t *testing.T) {
		p, err := f.NewEstimator(ctx, &wfv1.Workflow{})
		require.NoError(t, err)
		require.NotNil(t, p)
		e := p.(*estimator)
		assert.Nil(t, e.baselineWF)
	})
	t.Run("WorkflowTemplate", func(t *testing.T) {
		p, err := f.NewEstimator(ctx, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyWorkflowTemplate: "my-wftmpl"}},
		})
		require.NoError(t, err)
		require.NotNil(t, p)
		e := p.(*estimator)
		require.NotNil(t, e)
		require.NotNil(t, e.baselineWF)
		assert.Equal(t, "my-wftmpl-baseline", e.baselineWF.Name)
	})
	t.Run("ClusterWorkflowTemplate", func(t *testing.T) {
		p, err := f.NewEstimator(ctx, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyClusterWorkflowTemplate: "my-cwft"}},
		})
		require.NoError(t, err)
		require.NotNil(t, p)
		e := p.(*estimator)
		require.NotNil(t, e)
		require.NotNil(t, e.baselineWF)
		assert.Equal(t, "my-cwft-baseline", e.baselineWF.Name)
	})
	t.Run("CronWorkflowTemplate", func(t *testing.T) {
		p, err := f.NewEstimator(ctx, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyCronWorkflow: "my-cwf"}},
		})
		require.NoError(t, err)
		require.NotNil(t, p)
		e := p.(*estimator)
		require.NotNil(t, e)
		require.NotNil(t, e.baselineWF)
		assert.Equal(t, "my-cwf-baseline", e.baselineWF.Name)
	})
	t.Run("WorkflowArchive", func(t *testing.T) {
		p, err := f.NewEstimator(ctx, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyWorkflowTemplate: "my-archived-wftmpl"}},
		})
		require.NoError(t, err)
		require.NotNil(t, p)
		e := p.(*estimator)
		require.NotNil(t, e)
		require.NotNil(t, e.baselineWF)
		assert.Equal(t, "my-archived-wftmpl-baseline", e.baselineWF.Name)
	})
}
