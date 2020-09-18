package prediction

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller/indexes"
	"github.com/argoproj/argo/workflow/hydrator"
)

type DurationPredictorFactory interface {
	NewDurationPredictor(wf *wfv1.Workflow) (*DurationPredictor, error)
}

type durationPredictorFactory struct {
	wfInformer cache.SharedIndexInformer
	hydrator   hydrator.Interface
	wfArchive  sqldb.WorkflowArchive
}

var _ DurationPredictorFactory = &durationPredictorFactory{}

func NewDurationPredictorFactory(wfInformer cache.SharedIndexInformer, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive) DurationPredictorFactory {
	return &durationPredictorFactory{wfInformer, hydrator, wfArchive}
}

func (woc *durationPredictorFactory) NewDurationPredictor(wf *wfv1.Workflow) (*DurationPredictor, error) {
	for labelName, indexName := range map[string]string{
		common.LabelKeyWorkflowTemplate:        indexes.WorkflowTemplateIndex,
		common.LabelKeyClusterWorkflowTemplate: indexes.ClusterWorkflowTemplateIndex,
		common.LabelKeyCronWorkflow:            indexes.CronWorkflowIndex,
	} {
		labelValue, exists := wf.Labels[labelName]
		if exists {
			objs, err := woc.wfInformer.GetIndexer().ByIndex(indexName, indexes.MetaNamespaceLabelIndex(wf.Namespace, labelValue))
			if err != nil {
				return nil, fmt.Errorf("failed to list workflows by index: %v", err)
			}
			var newestWf *wfv1.Workflow
			for _, obj := range objs {
				un, ok := obj.(*unstructured.Unstructured)
				if !ok {
					return nil, fmt.Errorf("failed convert object to unstructured")
				}
				if un.GetLabels()[common.LabelKeyPhase] != string(wfv1.NodeSucceeded) {
					continue
				}
				candidateWf := &wfv1.Workflow{}
				err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, candidateWf)
				if err != nil {
					return nil, fmt.Errorf("failed convert unstructured to workflow: %w", err)
				}
				// we use `startedAt` because that's same as how the archive sorts
				if newestWf == nil || candidateWf.Status.StartedAt.Time.After(newestWf.Status.StartedAt.Time) {
					newestWf = candidateWf
				}
			}
			if newestWf != nil {
				err = woc.hydrator.Hydrate(newestWf)
				if err != nil {
					return nil, fmt.Errorf("failed hydrate last workflow: %w", err)
				}
				return &DurationPredictor{wf, newestWf}, nil
			}
			// we failed to find a base-line in the live set, so we now look in the archive
			labelRequirements, err := labels.ParseToRequirements(common.LabelKeyPhase + "=" + string(wfv1.NodeSucceeded) + "," + labelName + "=" + labelValue)
			if err != nil {
				return nil, fmt.Errorf("failed to parse selector to requirements: %v", err)
			}
			workflows, err := woc.wfArchive.ListWorkflows(wf.Namespace, time.Time{}, time.Time{}, labelRequirements, 1, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to list archived workflows: %v", err)
			}
			if len(workflows) > 0 {
				return &DurationPredictor{wf, &workflows[0]}, nil
			}
		}
	}
	return NullDurationPredictor, nil
}
