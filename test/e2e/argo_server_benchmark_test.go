//go:build api

package e2e

import (
	"testing"
)

func BenchmarkArgoServer(b *testing.B) {
	suite := new(ArgoServerSuite)
	suite.SetT(&testing.T{})
	suite.SetupSuite()
	suite.BeforeTest("ArgoServerBenchmark", "CreateWorkflowWithMultipleRefs")

	suite.Given().
		WorkflowTemplate("@benchmarks/multiple-ref-echo-1-workflowtemplate.yaml").
		When().
		CreateWorkflowTemplates()
	suite.Given().
		WorkflowTemplate("@benchmarks/multiple-ref-echo-2-workflowtemplate.yaml").
		When().
		CreateWorkflowTemplates()
	suite.Given().
		WorkflowTemplate("@benchmarks/multiple-ref-main-workflowtemplate.yaml").
		When().
		CreateWorkflowTemplates()

	b.Run("Submit workflow with multiple refs", func(b *testing.B) {
		for range b.N {
			suite.expectB(b).POST("/api/v1/workflows/argo").
				WithBytes([]byte(`{
					"workflow": {
						"metadata": {
							"generateName": "multiple-ref-template-benchmark-",
							"labels": {
								"workflows.argoproj.io/test": "true",
								"workflows.argoproj.io/workflow": "multiple-ref-template-benchmark"
							}
						},
						"spec": {
							"workflowTemplateRef": {"name": "multiple-ref-main"},
							"ttlStrategy": {"secondsAfterFailure": 1}
						}
					}
				}`)).
				Expect().
				Status(200)
		}
	})

	suite.DeleteResources()
	suite.TearDownSuite()
}
