package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
)

var table = map[string]string{
	"/service.TestService/GetTests":                                                         "get tests",
	"/workflowarchive.ArchivedWorkflowService/ListArchivedWorkflow":                         "list workflows",
	"/workflowarchive.ArchivedWorkflowService/DeleteArchivedWorkflow":                       "delete workflows",
	"/workflowarchive.ArchivedWorkflowService/GetArchivedWorkflow":                          "get workflows",
	"/workflowarchive.ArchivedWorkflowService/ListArchivedWorkflowLabelKeys":                "list workflows",
	"/workflowarchive.ArchivedWorkflowService/ListArchivedWorkflowLabelValues":              "list workflows",
	"/workflowarchive.ArchivedWorkflowService/ListArchivedWorkflows":                        "list workflows",
	"/workflowarchive.ArchivedWorkflowService/ResubmitArchivedWorkflow":                     "create workflows",
	"/workflowarchive.ArchivedWorkflowService/RetryArchivedWorkflow":                        "create workflows",
	"/clusterworkflowtemplate.ClusterWorkflowTemplateService/CreateClusterWorkflowTemplate": "create clusterworkflowtemplates",
	"/clusterworkflowtemplate.ClusterWorkflowTemplateService/DeleteClusterWorkflowTemplate": "delete clusterworkflowtemplates",
	"/clusterworkflowtemplate.ClusterWorkflowTemplateService/GetClusterWorkflowTemplate":    "get clusterworkflowtemplates",
	"/clusterworkflowtemplate.ClusterWorkflowTemplateService/LintClusterWorkflowTemplate":   "get clusterworkflowtemplates",
	"/clusterworkflowtemplate.ClusterWorkflowTemplateService/ListClusterWorkflowTemplates":  "list clusterworkflowtemplates",
	"/clusterworkflowtemplate.ClusterWorkflowTemplateService/UpdateClusterWorkflowTemplate": "update clusterworkflowtemplates",
	"/cronworkflow.CronWorkflowService/CreateCronWorkflow":                                  "create cronworkflows",
	"/cronworkflow.CronWorkflowService/DeleteCronWorkflow":                                  "delete cronworkflows",
	"/cronworkflow.CronWorkflowService/GetCronWorkflow":                                     "get cronworkflows",
	"/cronworkflow.CronWorkflowService/LintCronWorkflow":                                    "get cronworkflows",
	"/cronworkflow.CronWorkflowService/ListCronWorkflows":                                   "list cronworkflows",
	"/cronworkflow.CronWorkflowService/ResumeCronWorkflow":                                  "update cronworkflows",
	"/cronworkflow.CronWorkflowService/SuspendCronWorkflow":                                 "update cronworkflows",
	"/cronworkflow.CronWorkflowService/UpdateCronWorkflow":                                  "update cronworkflows",
	"/event.EventService/ListWorkflowEventBindings":                                         "list workfloweventbindings",
	"/event.EventService/ReceiveEvent":                                                      "get events",
	"/eventsource.EventSourceService/CreateEventSource":                                     "get eventsources",
	"/eventsource.EventSourceService/DeleteEventSource":                                     "get eventsources",
	"/eventsource.EventSourceService/EventSourcesLogs":                                      "get eventsourcelogs",
	"/eventsource.EventSourceService/GetEventSource":                                        "get eventsources",
	"/eventsource.EventSourceService/ListEventSources":                                      "list eventsources",
	"/eventsource.EventSourceService/UpdateEventSource":                                     "update eventsources",
	"/eventsource.EventSourceService/WatchEventSources":                                     "watch eventsources",
	"/info.InfoService/GetInfo":                                                             "get infos",
	"/info.InfoService/GetUserInfo":                                                         "get userinfos",
	"/info.InfoService/GetVersion":                                                          "get versions",
	"/pipeline.PipelineService/DeletePipeline":                                              "delete pipelines",
	"/pipeline.PipelineService/GetPipeline":                                                 "get pipelines",
	"/pipeline.PipelineService/ListPipelines":                                               "list pipelines",
	"/pipeline.PipelineService/PipelineLogs":                                                "get pipelinelogs",
	"/pipeline.PipelineService/RestartPipeline":                                             "update pipelines",
	"/pipeline.PipelineService/WatchPipelines":                                              "update pipelines",
	"/pipeline.PipelineService/WatchSteps":                                                  "watch steps",
	"/sensor.SensorService/CreateSensor":                                                    "create sensor",
	"/sensor.SensorService/DeleteSensor":                                                    "delete sensors",
	"/sensor.SensorService/GetSensor":                                                       "get sensors",
	"/sensor.SensorService/ListSensors":                                                     "list sensors",
	"/sensor.SensorService/SensorsLogs":                                                     "get workflows",
	"/sensor.SensorService/UpdateSensor":                                                    "get workflows",
	"/sensor.SensorService/WatchSensors":                                                    "get workflows",
	"/workflow.WorkflowService/CreateWorkflow":                                              "create workflows",
	"/workflow.WorkflowService/DeleteWorkflow":                                              "delete workflows",
	"/workflow.WorkflowService/GetWorkflow":                                                 "get workflows",
	"/workflow.WorkflowService/LintWorkflow":                                                "get workflows",
	"/workflow.WorkflowService/ListWorkflows":                                               "list workflows",
	"/workflow.WorkflowService/PodLogs":                                                     "get podlogs",
	"/workflow.WorkflowService/ResubmitWorkflow":                                            "create workflows",
	"/workflow.WorkflowService/ResumeWorkflow":                                              "update workflows",
	"/workflow.WorkflowService/RetryWorkflow":                                               "update workflows",
	"/workflow.WorkflowService/SetWorkflow":                                                 "update workflows",
	"/workflow.WorkflowService/StopWorkflow":                                                "update workflows",
	"/workflow.WorkflowService/SubmitWorkflow":                                              "create workflows",
	"/workflow.WorkflowService/SuspendWorkflow":                                             "suspend workflows",
	"/workflow.WorkflowService/TerminateWorkflow":                                           "update workflows",
	"/workflow.WorkflowService/WatchEvents":                                                 "watch events",
	"/workflow.WorkflowService/WatchWorkflows":                                              "watch workflows",
	"/workflow.WorkflowService/WorkflowLogs":                                                "get workflowlogs",
	"/workflowtemplate.WorkflowTemplateService/CreateWorkflowTemplate":                      "create workflowtemplates",
	"/workflowtemplate.WorkflowTemplateService/DeleteWorkflowTemplate":                      "delete workflowtemplates",
	"/workflowtemplate.WorkflowTemplateService/GetWorkflowTemplate":                         "get workflowtemplates",
	"/workflowtemplate.WorkflowTemplateService/LintWorkflowTemplate":                        "get workflowtemplates",
	"/workflowtemplate.WorkflowTemplateService/ListWorkflowTemplates":                       "list workflowtemplates",
	"/workflowtemplate.WorkflowTemplateService/UpdateWorkflowTemplate":                      "update workflowtemplates",
}

func getOperationID(ctx context.Context) (string, error) {
	s := grpc.ServerTransportStreamFromContext(ctx)
	if s == nil {
		return "", fmt.Errorf("unable to get transport stream from context")
	}
	m := s.Method()
	op, ok := table[m]
	if !ok {
		return "", fmt.Errorf("failed to find operation ID: unknown method %q", m)
	}
	return op, nil
}

func splitOp(method string) (string, string) {
	parts := strings.Split(method, " ")
	if len(parts) != 2 {
		panic(fmt.Errorf("expected 2 parts in %q", method))
	}
	return parts[0], parts[1]
}
