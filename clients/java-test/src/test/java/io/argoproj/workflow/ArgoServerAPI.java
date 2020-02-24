package io.argoproj.workflow;

import io.argoproj.workflow.apis.WorkflowServiceApi;
import io.argoproj.workflow.models.Workflow;
import io.argoproj.workflow.models.WorkflowCreateRequest;

public class ArgoServerAPI {
  /*
     By default, the Argo Server runs on port 2746. We need to provide a token - which can be found by running
     `argo auth token`.
  */
  private final ApiClient client =
      new ApiClient()
          .setDebugging(true)
          .addDefaultHeader("Authorization", "Bearer " + System.getenv("ARGO_TOKEN"));

  private final WorkflowServiceApi api = new WorkflowServiceApi(client);

  public Workflow createWorkflow(Workflow wf) throws ApiException {
    return api.createWorkflow("argo", new WorkflowCreateRequest().workflow(wf));
  }
}
