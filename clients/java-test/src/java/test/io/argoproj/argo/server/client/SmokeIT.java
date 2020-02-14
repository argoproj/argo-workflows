package io.argoproj.argo.server.client;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonDeserializer;
import com.google.gson.JsonParseException;
import io.argoproj.argo.server.client.api.WorkflowServiceApi;
import io.argoproj.argo.server.client.model.*;
import okhttp3.Response;
import org.junit.Test;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

public class SmokeIT {

    @Test
    public void smokeTest() throws Exception {

        V1ObjectMeta metadata = new V1ObjectMeta()
                .generateName("hello-world-")
                .namespace("argo");

        V1Container container = new V1Container()
                .image("cowsay:v1")
                .command(Collections.singletonList("cowsay"))
                .args(Collections.singletonList("hello world"));

        V1alpha1Template t = new V1alpha1Template()
                .name("whalesay")
                .container(container);

        V1alpha1WorkflowSpec spec = new V1alpha1WorkflowSpec()
                .entrypoint("whalesay")
                .templates(new ArrayList<>())
                .addTemplatesItem(t);

        V1alpha1Workflow wf = new V1alpha1Workflow()
                .metadata(metadata)
                .spec(spec);
        {
            ApiClient client = new ApiClient()
                    .setBasePath("https://localhost:6443")
                    .setVerifyingSsl(false)
                    .addDefaultHeader("Authorization", System.getenv("ARGO_TOKEN"))
                    .setDebugging(true);
            Response r = client.buildCall("/apis/argoproj.io/v1alpha1/namespaces/argo/workflows", "POST",
                    Collections.emptyList(), Collections.emptyList(), wf, Collections.emptyMap(), Collections.emptyMap(),
                    Collections.emptyMap(), new String[0], null)
                    .execute();
            assertEquals(201, r.code());
        }
        {
            Gson gson = new GsonBuilder().registerTypeAdapter(V1Time.class, (JsonDeserializer<V1Time>) (json, typeOfT, context) -> {
                try {
                    Date date = new SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'").parse(json.getAsString());
                    return new V1Time().nanos(1000 * (int) (date.getTime()));
                } catch (ParseException e) {
                    throw new JsonParseException(e);
                }
            }).create();
            ApiClient client = new ApiClient()
                    .setBasePath("http://localhost:2746")
                    .setVerifyingSsl(false)
                    .addDefaultHeader("Authorization", System.getenv("ARGO_TOKEN"))
                    .setDebugging(true)
                    .setJSON(new JSON().setGson(gson));
            WorkflowServiceApi api = new WorkflowServiceApi(client);

            V1alpha1Workflow workflow = api.createWorkflow("argo", new WorkflowWorkflowCreateRequest().workflow(wf));

            assertNotNull(workflow.getMetadata());
            assertNotNull(workflow.getMetadata().getUid());
        }
    }
}