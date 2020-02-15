package io.argoproj.argo.server.client;

import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.JSON;
import org.openapitools.client.api.WorkflowServiceApi;
import org.openapitools.client.model.V1alpha1Workflow;
import org.openapitools.client.model.WorkflowWorkflowCreateRequest;

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
