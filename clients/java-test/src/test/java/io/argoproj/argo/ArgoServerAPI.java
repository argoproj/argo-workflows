package io.argoproj.argo;

import io.argoproj.argo.api.WorkflowServiceApi;
import io.argoproj.argo.model.V1alpha1Workflow;
import io.argoproj.argo.model.WorkflowWorkflowCreateRequest;

public class ArgoServerAPI {

    /*
        By default, the Argo Server runs on port 2746. We need to provide a token - which can be found by running
        `argo auth token`.
     */
    private final ApiClient client = new ApiClient()
            .setVerifyingSsl(false)
            .setDebugging(true)
            .setBasePath("http://localhost:2746")
            .addDefaultHeader("Authorization", "Bearer " + System.getenv("ARGO_TOKEN"))
            .setJSON(new JSON().setGson(GsonFactory.GSON));

    private final WorkflowServiceApi api = new WorkflowServiceApi(client);

    public V1alpha1Workflow createWorkflow(V1alpha1Workflow wf) throws ApiException {
        return api.createWorkflow("argo", new WorkflowWorkflowCreateRequest().workflow(wf));
    }
}
