//go:build metrics

package e2e

import (
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
				httpexpect.NewDebugPrinter(s.T(), true),
			},
			Client: httpClient,
		})
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
	s.Given().
		Workflow(`@testdata/workflow-retry-metrics.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`runs_exit_status_counter{exit_code="1",status="Failed"} 3`)
		})
}

func (s *MetricsSuite) TestDAGMetrics() {
	s.Given().
		Workflow(`@testdata/workflow-dag-metrics.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`argo_workflows_result_counter{status="Succeeded"} 5`)
		})
}

func (s *MetricsSuite) TestDeprecatedCronSchedule() {
	s.Given().
		CronWorkflow(`@testdata/cronworkflow-deprecated-schedule.yaml`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`deprecated_feature{feature="cronworkflow schedule",namespace="argo"}`) // Count unimportant and unknown
		})
}

func (s *MetricsSuite) TestDeprecatedMutex() {
	s.Given().
		Workflow(`@testdata/synchronization-deprecated-mutex.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`deprecated_feature{feature="synchronization mutex",namespace="argo"}`) // Count unimportant and unknown
		})
}

func (s *MetricsSuite) TestDeprecatedSemaphore() {
	s.Given().
		Workflow(`@testdata/synchronization-deprecated-semaphore.yaml`).
		When().
		CreateConfigMap("my-config", map[string]string{"workflow": "1"}, map[string]string{}).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		DeleteConfigMap("my-config").
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`deprecated_feature{feature="synchronization semaphore",namespace="argo"}`) // Count unimportant and unknown
		})
}

func (s *MetricsSuite) TestFailedMetric() {
	s.Given().
		Workflow(`@testdata/template-status-failed-conditional-metric.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`argo_workflows_task_failure 1`)
		})
}

func (s *MetricsSuite) TestCronCountersForbid() {
	s.Given().
		CronWorkflow(`@testdata/cronworkflow-metrics-forbid.yaml`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Wait(time.Minute). // This pattern is used in cron_test.go too
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`cronworkflows_triggered_total{name="test-cron-metric-forbid",namespace="argo"} 1`). // 2nd run was Forbid
				Contains(`cronworkflows_concurrencypolicy_triggered{concurrency_policy="Forbid",name="test-cron-metric-forbid",namespace="argo"} 1`)
		})
}

func (s *MetricsSuite) TestCronCountersReplace() {
	s.Given().
		CronWorkflow(`@testdata/cronworkflow-metrics-replace.yaml`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		WaitForNewWorkflow(fixtures.ToBeRunning).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`cronworkflows_triggered_total{name="test-cron-metric-replace",namespace="argo"} 2`).
				Contains(`cronworkflows_concurrencypolicy_triggered{concurrency_policy="Replace",name="test-cron-metric-replace",namespace="argo"} 1`)
		})
}

func (s *MetricsSuite) TestPodPendingMetric() {
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
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`pod_pending_count{namespace="argo",reason="Unschedulable"} 1`)
		}).
		When().
		DeleteWorkflow().
		WaitForWorkflowDeletion()
}

func (s *MetricsSuite) TestTemplateMetrics() {
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
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`total_count{namespace="argo",phase="Running"}`). // Count for this depends on other tests
				Contains(`total_count{namespace="argo",phase="Succeeded"}`).
				Contains(`workflowtemplate_triggered_total{cluster_scope="false",name="basic",namespace="argo",phase="New"} 1`).
				Contains(`workflowtemplate_triggered_total{cluster_scope="false",name="basic",namespace="argo",phase="Running"} 1`).
				Contains(`workflowtemplate_triggered_total{cluster_scope="false",name="basic",namespace="argo",phase="Succeeded"} 1`).
				Contains(`workflowtemplate_runtime_count{cluster_scope="false",name="basic",namespace="argo"} 1`).
				Contains(`workflowtemplate_runtime_bucket{cluster_scope="false",name="basic",namespace="argo",le="0"} 0`).
				Contains(`workflowtemplate_runtime_bucket{cluster_scope="false",name="basic",namespace="argo",le="+Inf"} 1`)
		})
}

func (s *MetricsSuite) TestClusterTemplateMetrics() {
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
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`workflowtemplate_triggered_total{cluster_scope="true",name="basic",namespace="argo",phase="New"} 1`).
				Contains(`workflowtemplate_triggered_total{cluster_scope="true",name="basic",namespace="argo",phase="Running"} 1`).
				Contains(`workflowtemplate_triggered_total{cluster_scope="true",name="basic",namespace="argo",phase="Succeeded"} 1`).
				Contains(`workflowtemplate_runtime_count{cluster_scope="true",name="basic",namespace="argo"} 1`).
				Contains(`workflowtemplate_runtime_bucket{cluster_scope="true",name="basic",namespace="argo",le="0"} 0`).
				Contains(`workflowtemplate_runtime_bucket{cluster_scope="true",name="basic",namespace="argo",le="+Inf"} 1`)
		})
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}
