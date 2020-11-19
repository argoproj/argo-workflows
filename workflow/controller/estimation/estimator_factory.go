package estimation

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller/indexes"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/util"
)

type EstimatorFactory interface {
	// ALWAYS return as estimator, even if it also returns an error.
	NewEstimator(wf *wfv1.Workflow) (Estimator, error)
}

type estimatorFactory struct {
	wfInformer cache.SharedIndexInformer
	hydrator   hydrator.Interface
	wfArchive  sqldb.WorkflowArchive
}

var _ EstimatorFactory = &estimatorFactory{}

func NewEstimatorFactory(wfInformer cache.SharedIndexInformer, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive) EstimatorFactory {
	return &estimatorFactory{wfInformer, hydrator, wfArchive}
}

func (f *estimatorFactory) NewEstimator(wf *wfv1.Workflow) (Estimator, error) {
	defaultEstimator := &estimator{wf: wf}
	for labelName, indexName := range map[string]string{
		common.LabelKeyWorkflowTemplate:        indexes.WorkflowTemplateIndex,
		common.LabelKeyClusterWorkflowTemplate: indexes.ClusterWorkflowTemplateIndex,
		common.LabelKeyCronWorkflow:            indexes.CronWorkflowIndex,
	} {
		labelValue, exists := wf.Labels[labelName]
		if exists {
			objs, err := f.wfInformer.GetIndexer().ByIndex(indexName, indexes.MetaNamespaceLabelIndex(wf.Namespace, labelValue))
			if err != nil {
				return defaultEstimator, fmt.Errorf("failed to list workflows by index: %v", err)
			}
			var newestWf *wfv1.Workflow
			for _, obj := range objs {
				un, ok := obj.(*unstructured.Unstructured)
				if !ok {
					return defaultEstimator, fmt.Errorf("failed convert object to unstructured")
				}
				if un.GetLabels()[common.LabelKeyPhase] != string(wfv1.NodeSucceeded) {
					continue
				}
				candidateWf, err := util.FromUnstructured(un)
				if err != nil {
					return defaultEstimator, fmt.Errorf("failed convert unstructured to workflow: %w", err)
				}
				// we use `startedAt` because that's same as how the archive sorts
				if newestWf == nil || candidateWf.Status.StartedAt.Time.After(newestWf.Status.StartedAt.Time) {
					newestWf = candidateWf
				}
			}
			if newestWf != nil {
				err = f.hydrator.Hydrate(newestWf)
				if err != nil {
					return defaultEstimator, fmt.Errorf("failed hydrate last workflow: %w", err)
				}
				return &estimator{wf, newestWf}, nil
			}
			// we failed to find a base-line in the live set, so we now look in the archive
			requirements, err := labels.ParseToRequirements(common.LabelKeyPhase + "=" + string(wfv1.NodeSucceeded) + "," + labelName + "=" + labelValue)
			if err != nil {
				return defaultEstimator, fmt.Errorf("failed to parse selector to requirements: %v", err)
			}
			workflows, err := f.wfArchive.ListWorkflows(wf.Namespace, time.Time{}, time.Time{}, requirements, 1, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to list archived workflows: %v", err)
			}
			if len(workflows) > 0 {
				return &estimator{wf, &workflows[0]}, nil
			}
		}
	}
	return defaultEstimator, nil
}
