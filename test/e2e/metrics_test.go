//go:build metrics

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

const baseURLMetrics = "https://localhost:9090/metrics"

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type MetricsSuite struct {
	fixtures.E2ESuite
}

func (s *MetricsSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
}

func (s *MetricsSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  baseURLMetrics,
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(s.T(), false),
			},
			Client: httpClient,
		})
}

// Helper method to create a metric baseline tracker from expected increases map
func (s *MetricsSuite) captureBaseline(expectedIncreases map[string]float64) *fixtures.MetricBaseline {
	baseline := fixtures.NewMetricBaseline(s.T(), func() *httpexpect.Expect { return s.e(s.T()) })
	return baseline.CaptureBaseline(expectedIncreases)
}

func (s *MetricsSuite) TestMetricsEndpoint() {
	s.Run("Metrics", func() {
		s.e(s.T()).GET("").
			Expect().
			Status(200).
			Body().
			Contains(`HELP argo_workflows_gauge`).
			Contains(`HELP argo_workflows_k8s_request_total`).
			Contains(`argo_workflows_k8s_request_total{kind="leases",status_code="404",verb="Get"}`).
			Contains(`argo_workflows_k8s_request_total{kind="workflowtemplates",status_code="200",verb="List"}`).
			Contains(`argo_workflows_k8s_request_total{kind="workflowtemplates",status_code="200",verb="Watch"}`).
			Contains(`HELP argo_workflows_pods_gauge`).
			Contains(`HELP argo_workflows_workers_busy`).
			Contains(`HELP argo_workflows_workflow_condition`).
			Contains(`log_messages{level="info"}`).
			Contains(`log_messages{level="warning"}`).
			Contains(`log_messages{level="error"}`)
	})
}

func (s *MetricsSuite) TestRetryMetrics() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`runs_exit_status_counter{exit_code="1",status="Failed"}`: 3, // initial attempt + 2 retries
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		Workflow(`@testdata/workflow-retry-metrics.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)

			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func (s *MetricsSuite) TestDAGMetrics() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`argo_workflows_result_counter{status="Succeeded"}`: 5, // for the 5 DAG tasks: A, B, C, D, and the root task
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		Workflow(`@testdata/workflow-dag-metrics.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)

			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func (s *MetricsSuite) TestFailedMetric() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`argo_workflows_task_failure`: 1,
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		Workflow(`@testdata/template-status-failed-conditional-metric.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)

			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func (s *MetricsSuite) TestCronCountersForbid() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`cronworkflows_triggered_total{name="test-cron-metric-forbid",namespace="argo"}`:                                         1, // 2nd run was Forbid
		`cronworkflows_concurrencypolicy_triggered{concurrency_policy="Forbid",name="test-cron-metric-forbid",namespace="argo"}`: 1,
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		CronWorkflow(`@testdata/cronworkflow-metrics-forbid.yaml`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Wait(time.Minute). // This pattern is used in cron_test.go too
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func (s *MetricsSuite) TestCronCountersReplace() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`cronworkflows_triggered_total{name="test-cron-metric-replace",namespace="argo"}`:                                          2, // Two triggers
		`cronworkflows_concurrencypolicy_triggered{concurrency_policy="Replace",name="test-cron-metric-replace",namespace="argo"}`: 1, // One replace action
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		CronWorkflow(`@testdata/cronworkflow-metrics-replace.yaml`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		WaitForNewWorkflow(fixtures.ToBeRunning).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func (s *MetricsSuite) TestPodPendingMetric() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`pod_pending_count{namespace="argo",reason="Unschedulable"}`: 1,
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		Workflow(`@testdata/workflow-pending-metrics.yaml`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCondition(func(p *corev1.Pod) bool {
			if p.Status.Phase == corev1.PodPending {
				for _, cond := range p.Status.Conditions {
					if cond.Reason == corev1.PodReasonUnschedulable {
						return true
					}
				}
			}
			return false
		})).
		Wait(2 * time.Second). // Hack: We may well observe the pod change faster than the controller
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowRunning, status.Phase)

			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		}).
		When().
		DeleteWorkflow().
		WaitForWorkflowDeletion()
}

func (s *MetricsSuite) TestTemplateMetrics() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`total_count{namespace="argo",phase="Running"}`:                                                           1,
		`total_count{namespace="argo",phase="Succeeded"}`:                                                         1,
		`workflowtemplate_triggered_total{cluster_scope="false",name="basic",namespace="argo",phase="New"}`:       1,
		`workflowtemplate_triggered_total{cluster_scope="false",name="basic",namespace="argo",phase="Running"}`:   1,
		`workflowtemplate_triggered_total{cluster_scope="false",name="basic",namespace="argo",phase="Succeeded"}`: 1,
		`workflowtemplate_runtime_count{cluster_scope="false",name="basic",namespace="argo"}`:                     1,
		`workflowtemplate_runtime_bucket{cluster_scope="false",name="basic",namespace="argo",le="0"}`:             0, // Should not increase
		`workflowtemplate_runtime_bucket{cluster_scope="false",name="basic",namespace="argo",le="+Inf"}`:          1,
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		Workflow(`@testdata/templateref-metrics.yaml`).
		WorkflowTemplate(`@testdata/basic-workflowtemplate.yaml`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)

			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func (s *MetricsSuite) TestClusterTemplateMetrics() {
	// Define expected increases once
	expectedIncreases := map[string]float64{
		`workflowtemplate_triggered_total{cluster_scope="true",name="basic",namespace="argo",phase="New"}`:       1,
		`workflowtemplate_triggered_total{cluster_scope="true",name="basic",namespace="argo",phase="Running"}`:   1,
		`workflowtemplate_triggered_total{cluster_scope="true",name="basic",namespace="argo",phase="Succeeded"}`: 1,
		`workflowtemplate_runtime_count{cluster_scope="true",name="basic",namespace="argo"}`:                     1,
		`workflowtemplate_runtime_bucket{cluster_scope="true",name="basic",namespace="argo",le="0"}`:             0, // Should not increase
		`workflowtemplate_runtime_bucket{cluster_scope="true",name="basic",namespace="argo",le="+Inf"}`:          1,
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	s.Given().
		Workflow(`@testdata/clustertemplateref-metrics.yaml`).
		ClusterWorkflowTemplate(`@testdata/basic-clusterworkflowtemplate.yaml`).
		When().
		CreateClusterWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)

			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func (s *MetricsSuite) TestPodRestartMetric() {
	// Define expected increases - pod restart with Evicted reason and DiskPressure condition
	expectedIncreases := map[string]float64{
		`pod_restarts_total{condition="DiskPressure",namespace="argo",reason="Evicted"}`: 1,
	}

	// Capture baseline metrics for all expected metrics
	baseline := s.captureBaseline(expectedIncreases)

	var podName string

	s.Given().
		Workflow(`@testdata/workflow-pod-restart.yaml`).
		When().
		SubmitWorkflow().
		// Wait for the pod to be created
		WaitForPod(func(p *corev1.Pod) bool {
			if p.Status.Phase != corev1.PodRunning && p.Status.Phase != corev1.PodPending {
				return false
			}
			podName = p.Name
			return true
		}).
		And(func() {
			// Patch the pod status to simulate an eviction before main container started
			ctx := context.Background()

			patch := map[string]interface{}{
				"status": map[string]interface{}{
					"phase":   "Failed",
					"reason":  "Evicted",
					"message": "The node had condition: [DiskPressure]",
					"initContainerStatuses": []map[string]interface{}{
						{
							"name":  "init",
							"image": "alpine:latest",
							"state": map[string]interface{}{
								"terminated": map[string]interface{}{
									"exitCode": 0,
									"reason":   "Completed",
								},
							},
							"ready":        true,
							"restartCount": 0,
						},
						{
							"name":  "delay",
							"image": "alpine:latest",
							"state": map[string]interface{}{
								"terminated": map[string]interface{}{
									"exitCode": 137,
									"reason":   "Error",
								},
							},
							"ready":        false,
							"restartCount": 0,
						},
					},
					"containerStatuses": []map[string]interface{}{
						{
							"name":  "main",
							"image": "alpine:latest",
							"state": map[string]interface{}{
								"waiting": map[string]interface{}{
									"reason": "PodInitializing",
								},
							},
							"ready":        false,
							"restartCount": 0,
						},
					},
				},
			}
			patchBytes, err := json.Marshal(patch)
			s.Require().NoError(err)

			_, err = s.KubeClient.CoreV1().Pods(fixtures.Namespace).Patch(
				ctx,
				podName,
				types.MergePatchType,
				patchBytes,
				metav1.PatchOptions{},
				"status",
			)
			s.Require().NoError(err)
			s.T().Logf("Patched pod %s to simulate eviction", podName)
		}).
		WaitForWorkflow(fixtures.ToBeSucceeded, 60*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)

			// Check that metrics increased by the expected amounts
			baseline.ExpectIncrease()
		})
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}
