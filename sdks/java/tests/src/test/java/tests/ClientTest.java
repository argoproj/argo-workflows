package tests;


import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.apis.WorkflowServiceApi;
import io.argoproj.workflow.models.IoArgoprojWorkflowV1alpha1WorkflowList;
import org.junit.Test;

public class ClientTest {

    @Test
    public void testClient() throws ApiException {
        ApiClient defaultClient = Configuration.getDefaultApiClient()
                .setBasePath("http://localhost:2746");

        WorkflowServiceApi apiInstance = new WorkflowServiceApi(defaultClient);

        IoArgoprojWorkflowV1alpha1WorkflowList result = apiInstance.workflowServiceListWorkflows("argo",
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
        );
        System.out.println(result);
    }
}