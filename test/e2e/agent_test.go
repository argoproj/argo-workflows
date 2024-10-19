//go:build functional

package e2e

// import (
// 	"sort"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"github.com/stretchr/testify/suite"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// 	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
// 	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
// )

// type AgentSuite struct {
// 	fixtures.E2ESuite
// }

// func (s *AgentSuite) TestParallel() {
// 	s.Given().
// 		Workflow(`apiVersion: argoproj.io/v1alpha1
// kind: Workflow
// metadata:
//   generateName: http-template-par-
//   workflowMetadata:
//     labels:
//       workflows.argoproj.io/test: "true"
// spec:
//   entrypoint: main
//   templates:
//     - name: main
//       steps:
//         - - name: one
//             template: http
//             arguments:
//               parameters: [{name: url, value: "https://argoproj.github.io"}]
//           - name: two
//             template: http
//             arguments:
//               parameters: [{name: url, value: "https://argoproj.github.io"}]
//           - name: three
//             template: http
//             arguments:
//               parameters: [{name: url, value: "https://argoproj.github.io"}]
//           - name: four
//             template: http
//             arguments:
//               parameters: [{name: url, value: "https://argoproj.github.io"}]
//     - name: http
//       inputs:
//         parameters:
//           - name: url
//       http:
//         url: "{{inputs.parameters.url}}"
// `).
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(fixtures.ToBeCompleted).
// 		Then().
// 		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
// 			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
// 			// Ensure that the workflow ran for less than 10 seconds
// 			assert.Less(t, status.FinishedAt.Sub(status.StartedAt.Time), time.Duration(10*fixtures.EnvFactor)*time.Second)

// 			var finishedTimes []time.Time
// 			var startTimes []time.Time
// 			for _, node := range status.Nodes {
// 				if node.Type != wfv1.NodeTypeHTTP {
// 					continue
// 				}
// 				startTimes = append(startTimes, node.StartedAt.Time)
// 				finishedTimes = append(finishedTimes, node.FinishedAt.Time)
// 			}

// 			require.Len(t, finishedTimes, 4)
// 			sort.Slice(finishedTimes, func(i, j int) bool {
// 				return finishedTimes[i].Before(finishedTimes[j])
// 			})
// 			// Everything finished with a two second tolerance window
// 			assert.Less(t, finishedTimes[3].Sub(finishedTimes[0]), time.Duration(2)*time.Second)

// 			require.Len(t, startTimes, 4)
// 			sort.Slice(startTimes, func(i, j int) bool {
// 				return startTimes[i].Before(startTimes[j])
// 			})
// 			// Everything started with same time
// 			assert.Equal(t, time.Duration(0), startTimes[3].Sub(startTimes[0]))
// 		})
// }

// func (s *AgentSuite) TestStatusCondition() {
// 	s.Given().
// 		Workflow(`apiVersion: argoproj.io/v1alpha1
// kind: Workflow
// metadata:
//   generateName: http-template-condition-
//   workflowMetadata:
//     labels:
//       workflows.argoproj.io/test: "true"
// spec:
//   entrypoint: main
//   templates:
//     - name: main
//       steps:
//         - - name: http-status-is-201-fails
//             template: http-status-is-201
//             arguments:
//               parameters: [{name: url, value: "http://httpbin:9100/status/200"}]
//           - name: http-status-is-201-succeeds
//             template: http-status-is-201
//             arguments:
//               parameters: [{name: url, value: "http://httpbin:9100/status/201"}]
//           - name: http-body-contains-google-fails
//             template: http-body-contains-google
//             arguments:
//               parameters: [{name: url, value: "http://httpbin:9100/status/200"}]
//           - name: http-body-contains-google-succeeds
//             template: http-body-contains-google
//             arguments:
//               parameters: [{name: url, value: "https://google.com"}]
//     - name: http-status-is-201
//       inputs:
//         parameters:
//           - name: url
//       http:
//         successCondition: "response.statusCode == 201"
//         url: "{{inputs.parameters.url}}"
//     - name: http-body-contains-google
//       inputs:
//         parameters:
//           - name: url
//       http:
//         successCondition: "response.body contains \"google\""
//         url: "{{inputs.parameters.url}}"
// `).
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(2 * time.Minute).
// 		Then().
// 		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
// 			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)

// 			containsFails := status.Nodes.FindByDisplayName("http-body-contains-google-fails")
// 			require.NotNil(t, containsFails)
// 			assert.Equal(t, wfv1.NodeFailed, containsFails.Phase)

// 			containsSucceeds := status.Nodes.FindByDisplayName("http-body-contains-google-succeeds")
// 			require.NotNil(t, containsFails)
// 			assert.Equal(t, wfv1.NodeSucceeded, containsSucceeds.Phase)

// 			statusFails := status.Nodes.FindByDisplayName("http-status-is-201-fails")
// 			require.NotNil(t, statusFails)
// 			assert.Equal(t, wfv1.NodeFailed, statusFails.Phase)

// 			statusSucceeds := status.Nodes.FindByDisplayName("http-status-is-201-succeeds")
// 			require.NotNil(t, statusFails)
// 			assert.Equal(t, wfv1.NodeSucceeded, statusSucceeds.Phase)
// 		})
// }

// func TestAgentSuite(t *testing.T) {
// 	suite.Run(t, new(AgentSuite))
// }
