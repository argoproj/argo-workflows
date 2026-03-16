package estimation

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

type EstimatorFactory interface {
	// ALWAYS return as estimator, even if it also returns an error.
	NewEstimator(ctx context.Context, wf *wfv1.Workflow) (Estimator, error)
}

type estimatorFactory struct {
	wfInformer cache.SharedIndexInformer
	hydrator   hydrator.Interface
	wfArchive  sqldb.WorkflowArchive
}

var _ EstimatorFactory = &estimatorFactory{}

var (
	skipWorkflowDurationEstimation = env.LookupEnvStringOr("SKIP_WORKFLOW_DURATION_ESTIMATION", "false")
)

func NewEstimatorFactory(ctx context.Context, wfInformer cache.SharedIndexInformer, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive) EstimatorFactory {
	return &estimatorFactory{wfInformer, hydrator, wfArchive}
}

func (f *estimatorFactory) NewEstimator(ctx context.Context, wf *wfv1.Workflow) (Estimator, error) {
	defaultEstimator := &estimator{wf: wf}
	if skipWorkflowDurationEstimation == "true" {
		return defaultEstimator, nil
	}
	for labelName, indexName := range map[string]string{
		common.LabelKeyWorkflowTemplate:        indexes.WorkflowTemplateIndex,
		common.LabelKeyClusterWorkflowTemplate: indexes.ClusterWorkflowTemplateIndex,
		common.LabelKeyCronWorkflow:            indexes.CronWorkflowIndex,
	} {
		labelValue, exists := wf.Labels[labelName]
		if exists {
			objs, err := f.wfInformer.GetIndexer().ByIndex(indexName, indexes.MetaNamespaceLabelIndex(wf.Namespace, labelValue))
			if err != nil {
				return defaultEstimator, fmt.Errorf("failed to list workflows by index: %w", err)
			}
			var newestUn *unstructured.Unstructured
			for _, obj := range objs {
				un, ok := obj.(*unstructured.Unstructured)
				if !ok {
					return defaultEstimator, fmt.Errorf("failed convert object to unstructured")
				}
				if un.GetLabels()[common.LabelKeyPhase] != string(wfv1.NodeSucceeded) {
					continue
				}
				// we use `creationTimestamp` because it's fast
				if newestUn == nil || un.GetCreationTimestamp().After(newestUn.GetCreationTimestamp().Time) {
					newestUn = un
				}
			}
			if newestUn != nil {
				newestWf, convErr := util.FromUnstructured(newestUn)
				if convErr != nil {
					return defaultEstimator, fmt.Errorf("failed convert unstructured to workflow: %w", convErr)
				}
				err = f.hydrator.Hydrate(ctx, newestWf)
				if err != nil {
					return defaultEstimator, fmt.Errorf("failed hydrate last workflow: %w", err)
				}
				return &estimator{wf, newestWf}, nil
			}
			// we failed to find a base-line in the live set, so we now look in the archive
			requirements, err := labels.ParseToRequirements(labelName + "=" + labelValue)
			if err != nil {
				return defaultEstimator, fmt.Errorf("failed to parse selector to requirements: %w", err)
			}
			baselineWF, err := f.wfArchive.GetWorkflowForEstimator(ctx, wf.Namespace, requirements)
			if err != nil {
				return defaultEstimator, fmt.Errorf("failed to get archived workflow for estimator: %w", err)
			}
			return &estimator{wf, baselineWF}, nil
		}
	}
	return defaultEstimator, nil
}
