package estimation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	sqldbmocks "github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	testutil "github.com/argoproj/argo/test/util"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller/indexes"
	hydratorfake "github.com/argoproj/argo/workflow/hydrator/fake"
)

func Test_estimatorFactory(t *testing.T) {
	informer := testutil.NewSharedIndexInformer()
	wfFailed := testutil.MustUnmarshallUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: bad-baseline
  labels:
    workflows.argoproj.io/phase: Failed
`)
	informer.Indexer.SetByIndex(indexes.ClusterWorkflowTemplateIndex, "my-ns/my-cwft", testutil.MustUnmarshallUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-cwft-baseline
  labels:
    workflows.argoproj.io/phase: Succeeded
`), wfFailed)
	informer.Indexer.SetByIndex(indexes.CronWorkflowIndex, "my-ns/my-cwf", testutil.MustUnmarshallUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-cwf-baseline
  labels:
    workflows.argoproj.io/phase: Succeeded
`), wfFailed)
	informer.Indexer.SetByIndex(indexes.WorkflowTemplateIndex, "my-ns/my-wftmpl", testutil.MustUnmarshallUnstructured(`
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
	wfArchive.On("ListWorkflows", "my-ns", time.Time{}, time.Time{}, labels.Requirements(r), 1, 0).Return(wfv1.Workflows{
		*testutil.MustUnmarshallWorkflow(`
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
