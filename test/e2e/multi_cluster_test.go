//go:build multicluster
// +build multicluster

package e2e

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/stretchr/testify/suite"
)

type MultiClusterSuite struct {
	fixtures.E2ESuite
}

func (s *MultiClusterSuite) TestAllowedRemoteCluster() {
	s.Given().
		Workflow(`
metadata:
  generateName: allow-cluster-
spec:
  entrypoint: main
  artifactRepositoryRef:
    key: empty
  templates:
    - name: main
      cluster: cluster-1
      namespace: default
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *MultiClusterSuite) TestAllowedLocalNamespace() {
	s.Given().
		Workflow(`
metadata:
  generateName: allow-ns-
spec:
  entrypoint: main
  artifactRepositoryRef:
    key: empty
  templates:
    - name: main
      namespace: default
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *MultiClusterSuite) TestOutputResult() {
	s.Given().
		Workflow(`
metadata:
  generateName: output-result-
spec:
  entrypoint: main
  artifactRepositoryRef:
    key: empty
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            template: produce
          - name: b
            template: consume
            dependencies:
              - a
            arguments:
              parameters:
                - name: text
                  value: "{{tasks.a.outputs.result}}"

    - name: produce
      cluster: cluster-1
      namespace: default
      container:
        image: argoproj/argosay:v2

    - name: consume
      inputs:
        parameters:
          - name: text
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *MultiClusterSuite) TestDisallowedNamespace() {
	s.Given().
		Workflow(`
metadata:
  generateName: multi-cluster-
spec:
  entrypoint: main
  artifactRepositoryRef:
    key: empty
  templates:
    - name: main
      cluster: cluster-1
      namespace: argo
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored).
		Then().
		ExpectWorkflow(fixtures.StatusMessageContains(`namespace "argo" is forbidden from creating resources in cluster "cluster-1" namespace "default"`))
}

func (s *MultiClusterSuite) TestDisallowedCluster() {
	s.Given().
		Workflow(`
metadata:
  generateName: multi-cluster-
spec:
  entrypoint: main
  artifactRepositoryRef:
    key: empty
  templates:
    - name: main
      cluster: cluster-1
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored).
		Then().
		ExpectWorkflow(fixtures.StatusMessageContains(`namespace "argo" is forbidden from creating resources in cluster "cluster-1" namespace "argo"`))
}

func TestMultiClusterSuite(t *testing.T) {
	suite.Run(t, new(MultiClusterSuite))
}
