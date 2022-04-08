package test;


import org.junit.Test;
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArchivedWorkflowServiceApi;

public class ClientTest {

    @Test
    public void testClient() {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArchivedWorkflowServiceApi apiInstance = new ArchivedWorkflowServiceApi(defaultClient);
        String uid = "uid_example"; // String |
        try {
            Object result = apiInstance.archivedWorkflowServiceDeleteArchivedWorkflow(uid);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling ArchivedWorkflowServiceApi#archivedWorkflowServiceDeleteArchivedWorkflow");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}