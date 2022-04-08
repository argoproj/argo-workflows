package tests;


import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.apis.WorkflowServiceApi;
import io.argoproj.workflow.auth.ApiKeyAuth;
import org.junit.Test;

import static org.junit.Assert.assertNotNull;

public class ClientTest {

    private final ApiClient defaultClient = Configuration.getDefaultApiClient();

    {
        ApiKeyAuth bearerAuth = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
        bearerAuth.setApiKey(System.getenv().get("ARGO_TOKEN"));
    }

    private final WorkflowServiceApi apiInstance = new WorkflowServiceApi(defaultClient);

    @Test
    public void testClient() throws Exception {
        assertNotNull(apiInstance.workflowServiceListWorkflows(
                "argo",
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null
        ));
    }
}