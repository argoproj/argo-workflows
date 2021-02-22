package controller

import (
	"fmt"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type operatorDataSourceProcessor struct {}

func newOperatorDataSourceProcessor() *operatorDataSourceProcessor {
	return &operatorDataSourceProcessor{}
}

func (ep *operatorDataSourceProcessor) ProcessArtifactPaths(_ *wfv1.ArtifactPaths) (interface{}, error) {
	return nil, fmt.Errorf("internal error: operatorDataSourceProcessor is not able to process artifactPaths, a pod should have been launched")
}
