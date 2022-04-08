package tests;


import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.JSON;
import io.argoproj.workflow.apis.WorkflowServiceApi;
import io.argoproj.workflow.auth.ApiKeyAuth;
import io.argoproj.workflow.models.IoArgoprojWorkflowV1alpha1Template;
import io.argoproj.workflow.models.IoArgoprojWorkflowV1alpha1Workflow;
import io.argoproj.workflow.models.IoArgoprojWorkflowV1alpha1WorkflowCreateRequest;
import io.argoproj.workflow.models.IoArgoprojWorkflowV1alpha1WorkflowSpec;
import io.kubernetes.client.openapi.models.V1Container;
import io.kubernetes.client.openapi.models.V1ObjectMeta;
import org.junit.Test;

import java.util.Collections;

public class ClientTest {

    private final ApiClient defaultClient = Configuration.getDefaultApiClient();

    public static final String argoToken = System.getenv().get("ARGO_TOKEN");

    {
        ApiKeyAuth bearerAuth = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
        bearerAuth.setApiKey(argoToken);
    }

    private final WorkflowServiceApi apiInstance = new WorkflowServiceApi(defaultClient);
    private final JSON json = new JSON();

    @Test
    public void testClient() throws Exception {
        // create a workflow
        IoArgoprojWorkflowV1alpha1WorkflowCreateRequest req = new IoArgoprojWorkflowV1alpha1WorkflowCreateRequest();
        req.setWorkflow(
                new IoArgoprojWorkflowV1alpha1Workflow()
                        .metadata(new V1ObjectMeta().generateName("test-"))
                        .spec(
                                new IoArgoprojWorkflowV1alpha1WorkflowSpec()
                                        .entrypoint("main")
                                        .templates(
                                                Collections.singletonList(
                                                        new IoArgoprojWorkflowV1alpha1Template()
                                                                .name("main")
                                                                .container(
                                                                        new V1Container()
                                                                                .image("argoproj/argosay:v2")
                                                                )
                                                )
                                        )
                        )
        );
        apiInstance.workflowServiceCreateWorkflow("argo",
                req);

    }
}