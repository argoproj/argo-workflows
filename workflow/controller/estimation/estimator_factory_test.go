package estimation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	sqldbmocks "github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	testutil "github.com/argoproj/argo-workflows/v4/test/util"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
)

// mustUnmarshalUnstructuredFile loads an unstructured object from a file
// relative to the package directory, mirroring the "@testdata" convention
// supported by wfv1.MustUnmarshalWorkflow for workflow-typed fixtures.
func mustUnmarshalUnstructuredFile(path string) *unstructured.Unstructured {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		panic(err)
	}
	return testutil.MustUnmarshalUnstructured(string(content))
}

func Test_estimatorFactory(t *testing.T) {
	informer := testutil.NewSharedIndexInformer()
	ctx := logging.TestContext(t.Context())
	wfFailed := mustUnmarshalUnstructuredFile("testdata/estimator_factory/failed-baseline.yaml")
	informer.Indexer.SetByIndex(indexes.ClusterWorkflowTemplateIndex, "my-ns/my-cwft", mustUnmarshalUnstructuredFile("testdata/estimator_factory/cwft-baseline.yaml"), wfFailed)
	informer.Indexer.SetByIndex(indexes.CronWorkflowIndex, "my-ns/my-cwf", mustUnmarshalUnstructuredFile("testdata/estimator_factory/cwf-baseline.yaml"), wfFailed)
	informer.Indexer.SetByIndex(indexes.WorkflowTemplateIndex, "my-ns/my-wftmpl", mustUnmarshalUnstructuredFile("testdata/estimator_factory/wftmpl-baseline.yaml"), wfFailed)
	wfArchive := &sqldbmocks.WorkflowArchive{}
	r, err := labels.ParseToRequirements("workflows.argoproj.io/workflow-template=my-archived-wftmpl")
	require.NoError(t, err)
	wfArchive.On("GetWorkflowForEstimator", mock.Anything, "my-ns", r).Return(wfv1.MustUnmarshalWorkflow("@testdata/estimator_factory/archived-wftmpl-baseline.yaml"), nil)
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
