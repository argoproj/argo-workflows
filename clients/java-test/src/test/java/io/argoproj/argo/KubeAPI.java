package io.argoproj.argo;

import com.google.gson.JsonObject;
import com.google.gson.JsonPrimitive;
import io.argoproj.argo.models.Workflow;
import okhttp3.Response;

import java.io.IOException;

import static java.util.Collections.emptyList;
import static java.util.Collections.emptyMap;

public class KubeAPI {
    /*
        By default, the Kubernetes API Server runs on port 6443. We need to provide a token - which can be found by
        running `argo auth token`.
     */
    private final ApiClient client = new ApiClient()
            .setVerifyingSsl(false)
            .setDebugging(true)
            .setBasePath("https://localhost:6443")
            .addDefaultHeader("Authorization", "Bearer " + System.getenv("ARGO_TOKEN"));

    public Workflow createWorkflow(Workflow wf) throws ApiException, IOException {
        Response r = client.buildCall("/apis/argoproj.io/v1alpha1/namespaces/argo/workflows", "POST",
                emptyList(), emptyList(),
                withKindAPIVersion(wf),
                emptyMap(), emptyMap(), emptyMap(), new String[0], null
        ).execute();
        if (r.code() != 201) {
            throw new ApiException("failed to create workflow");
        }
        return client.getJSON().deserialize(r.body().string(), Workflow.class);
    }

    // For Kubernetes, we must additionally add `kind` and `apiVersion` to our requests.
    public JsonObject withKindAPIVersion(Workflow wf) {
        JsonObject o = (JsonObject) client.getJSON().getGson().toJsonTree(wf);
        o.add("kind", new JsonPrimitive("Workflow"));
        o.add("apiVersion", new JsonPrimitive("argoproj.io/v1alpha1"));
        return o;
    }
}
