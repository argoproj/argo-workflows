package sqldb

import (
	"encoding/json"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type WorkflowMetadata struct {
	Id              string         `db:"id"`
	Name            string         `db:"name"`
	Phase           wfv1.NodePhase `db:"phase"`
	Namespace       string         `db:"namespace"`
	ResourceVersion string         `db:"resourceversion"`
	StartedAt       time.Time      `db:"startedat"`
	FinishedAt      time.Time      `db:"finishedat"`
}

type WorkflowOnlyRecord struct {
	Workflow string `db:"workflow"`
}

type WorkflowRecord struct {
	WorkflowMetadata
	WorkflowOnlyRecord
}

func toRecord(wf *wfv1.Workflow) (*WorkflowRecord, error) {
	jsonWf, err := json.Marshal(wf)
	if err != nil {
		return nil, err
	}
	startT, err := time.Parse(time.RFC3339, wf.Status.StartedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	endT, err := time.Parse(time.RFC3339, wf.Status.FinishedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	return &WorkflowRecord{
		WorkflowMetadata: WorkflowMetadata{
			Id:              string(wf.UID),
			Name:            wf.Name,
			Namespace:       wf.Namespace,
			ResourceVersion: wf.ResourceVersion,
			Phase:           wf.Status.Phase,
			StartedAt:       startT,
			FinishedAt:      endT,
		},
		WorkflowOnlyRecord: WorkflowOnlyRecord{Workflow: string(jsonWf)},
	}, nil
}

func toWorkflows(workflows []WorkflowOnlyRecord) (wfv1.Workflows, error) {
	wfs := make(wfv1.Workflows, len(workflows))
	for i, wf := range workflows {
		wf, err := toWorkflow(&wf)
		if err != nil {
			return nil, err
		}
		wfs[i] = *wf
	}
	return wfs, nil
}
func toWorkflow(workflow *WorkflowOnlyRecord) (*wfv1.Workflow, error) {
	var wf *wfv1.Workflow
	err := json.Unmarshal([]byte(workflow.Workflow), &wf)
	return wf, err
}

func toSlimWorkflows(mds []WorkflowMetadata) wfv1.Workflows {
	wfs := make(wfv1.Workflows, len(mds))
	for i, md := range mds {
		wfs[i] = wfv1.Workflow{
			ObjectMeta: v1.ObjectMeta{
				Name:              md.Name,
				Namespace:         md.Namespace,
				UID:               types.UID(md.Id),
				CreationTimestamp: v1.Time{Time: md.StartedAt},
			},
			Status: wfv1.WorkflowStatus{
				Phase:      md.Phase,
				StartedAt:  v1.Time{Time: md.StartedAt},
				FinishedAt: v1.Time{Time: md.FinishedAt},
			},
		}
	}
	return wfs
}
