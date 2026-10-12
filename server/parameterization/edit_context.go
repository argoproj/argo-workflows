package parameterization

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// SourceExecution links an explicit new submission to its original execution.
// It is not a request to retry or mutate that Workflow.
type SourceExecution = parameterapi.SourceExecution

type EditContext struct {
	Source   SourceExecution            `json:"source"`
	Binding  string                     `json:"binding"`
	Revision *unstructured.Unstructured `json:"revision"`
	Set      *unstructured.Unstructured `json:"set"`
}

func (h Handler) editContext(ctx context.Context, s Store, source SourceExecution) (*EditContext, error) {
	if source.Workflow == "" || source.UID == "" {
		return nil, fmt.Errorf("source execution needs Workflow name and UID")
	}
	wf, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(s.Namespace).Get(ctx, source.Workflow, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if string(wf.UID) != source.UID {
		return nil, fmt.Errorf("source workflow identity changed")
	}
	submission := wf.Annotations[p.Group+"/submission"]
	revisionName := wf.Annotations[p.Group+"/revision"]
	if submission == "" || revisionName == "" {
		return nil, fmt.Errorf("source workflow has no structured parameter submission")
	}
	receipt, err := s.get(ctx, "workflowsubmissions", submission)
	if err != nil {
		return nil, err
	}
	var saved struct {
		Binding  string `json:"binding"`
		Revision string `json:"revision"`
		Workflow struct {
			Metadata metav1.ObjectMeta `json:"metadata"`
		} `json:"workflow"`
	}
	if err = decode(receipt, &saved); err != nil {
		return nil, err
	}
	if saved.Workflow.Metadata.Name != wf.Name || saved.Workflow.Metadata.Namespace != s.Namespace || saved.Binding == "" {
		return nil, fmt.Errorf("source submission does not match workflow and revision")
	}
	if saved.Revision != revisionName {
		retry, retryErr := s.get(ctx, "parameterretries", wf.Annotations[p.Group+"/retry"])
		if retryErr != nil {
			return nil, retryErr
		}
		var attempt parameterapi.ParameterRetrySpec
		if retryErr = decode(retry, &attempt); retryErr != nil {
			return nil, retryErr
		}
		// An original-input retry after a partial retry still points at the adopted
		// revision, but does not need another copy of the earlier execution snapshot.
		if attempt.WorkflowUID != source.UID || attempt.Workflow != source.Workflow || attempt.Revision != revisionName {
			return nil, fmt.Errorf("retry receipt does not match current workflow revision")
		}
		if attempt.Plan != nil && attempt.Plan.Binding != "" {
			saved.Binding = attempt.Plan.Binding
		} else {
			// A whole-workflow original retry preserves its predecessor's
			// binding rather than reverting to the first submission.
			previous := attempt.PreviousWorkflow
			for depth := 0; previous != nil && previous.Annotations[p.Group+"/retry"] != ""; depth++ {
				if depth >= 64 {
					return nil, fmt.Errorf("retry binding history exceeds its limit")
				}
				entry, e := s.get(ctx, "parameterretries", previous.Annotations[p.Group+"/retry"])
				if e != nil {
					return nil, e
				}
				var prior parameterapi.ParameterRetrySpec
				if e = decode(entry, &prior); e != nil {
					return nil, e
				}
				if prior.WorkflowUID != source.UID || prior.Workflow != source.Workflow || prior.Revision != previous.Annotations[p.Group+"/revision"] {
					return nil, fmt.Errorf("retry binding history identity changed")
				}
				if prior.Plan != nil && prior.Plan.Binding != "" {
					saved.Binding = prior.Plan.Binding
					break
				}
				previous = prior.PreviousWorkflow
			}
		}
	}
	revision, err := s.get(ctx, "parameterrevisions", revisionName)
	if err != nil {
		return nil, err
	}
	var rev Revision
	if err = decode(revision, &rev); err != nil {
		return nil, err
	}
	set, err := s.get(ctx, "parametersets", rev.Set)
	if err != nil {
		return nil, err
	}
	return &EditContext{Source: source, Binding: saved.Binding, Revision: revision, Set: set}, nil
}
