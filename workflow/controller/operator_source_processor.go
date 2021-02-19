package controller

import (
	"fmt"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type operatorSourceProcessor struct {}

func newWorkflowExecutorSourceProcessor() *operatorSourceProcessor {
	return &operatorSourceProcessor{}
}

func (ep *operatorSourceProcessor) ProcessArtifactPaths(_ *wfv1.ArtifactPaths) (interface{}, error) {
	return nil, fmt.Errorf("operatorSourceProcessor is not able to process artifactPaths")
}
