package estimation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	sqldbmocks "github.com/argoproj/argo-workflows/v3/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/utils"
	testutil "github.com/argoproj/argo-workflows/v3/test/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	hydratorfake "github.com/argoproj/argo-workflows/v3/workflow/hydrator/fake"
)

func Test_estimatorFactory(t *testing.T) {
	informer := testutil.NewSharedIndexInformer()
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
	r, err := labels.ParseToRequirements("workflows.argoproj.io/phase=Succeeded,workflows.argoproj.io/workflow-template=my-archived-wftmpl")
	assert.NoError(t, err)
	wfArchive.On("ListWorkflows", utils.ListOptions{
		Namespace:         "my-ns",
		LabelRequirements: r,
		Limit:             1,
	}).Return(wfv1.Workflows{
		*testutil.MustUnmarshalWorkflow(`
metadata:
  name: my-archived-wftmpl-baseline`),
	}, nil)
	f := NewEstimatorFactory(informer, hydratorfake.Always, wfArchive)
	t.Run("None", func(t *testing.T) {
		p, err := f.NewEstimator(&wfv1.Workflow{})
		if assert.NoError(t, err) && assert.NotNil(t, p) {
			e := p.(*estimator)
			assert.Nil(t, e.baselineWF)
		}
	})
	t.Run("WorkflowTemplate", func(t *testing.T) {
		p, err := f.NewEstimator(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyWorkflowTemplate: "my-wftmpl"}},
		})
		if assert.NoError(t, err) && assert.NotNil(t, p) {
			e := p.(*estimator)
			if assert.NotNil(t, e) && assert.NotNil(t, e.baselineWF) {
				assert.Equal(t, "my-wftmpl-baseline", e.baselineWF.Name)
			}
		}
	})
	t.Run("ClusterWorkflowTemplate", func(t *testing.T) {
		p, err := f.NewEstimator(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyClusterWorkflowTemplate: "my-cwft"}},
		})
		if assert.NoError(t, err) && assert.NotNil(t, p) {
			e := p.(*estimator)
			if assert.NotNil(t, e) && assert.NotNil(t, e.baselineWF) {
				assert.Equal(t, "my-cwft-baseline", e.baselineWF.Name)
			}
		}
	})
	t.Run("CronWorkflowTemplate", func(t *testing.T) {
		p, err := f.NewEstimator(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyCronWorkflow: "my-cwf"}},
		})
		if assert.NoError(t, err) && assert.NotNil(t, p) {
			e := p.(*estimator)
			if assert.NotNil(t, e) && assert.NotNil(t, e.baselineWF) {
				assert.Equal(t, "my-cwf-baseline", e.baselineWF.Name)
			}
		}
	})
	t.Run("WorkflowArchive", func(t *testing.T) {
		p, err := f.NewEstimator(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{common.LabelKeyWorkflowTemplate: "my-archived-wftmpl"}},
		})
		if assert.NoError(t, err) && assert.NotNil(t, p) {
			e := p.(*estimator)
			if assert.NotNil(t, e) && assert.NotNil(t, e.baselineWF) {
				assert.Equal(t, "my-archived-wftmpl-baseline", e.baselineWF.Name)
			}
		}
	})
}
