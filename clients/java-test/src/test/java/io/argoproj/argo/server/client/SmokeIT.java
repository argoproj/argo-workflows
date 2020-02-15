package io.argoproj.argo.server.client;

import org.junit.Test;
import org.openapitools.client.model.*;

import java.util.ArrayList;
import java.util.Collections;

import static org.junit.Assert.assertNotNull;

public class SmokeIT {

    private final V1alpha1Workflow wf = new V1alpha1Workflow()
            .metadata(new V1ObjectMeta()
                    .generateName("hello-world-")
                    .namespace("argo"))
            .spec(new V1alpha1WorkflowSpec()
                    .entrypoint("whalesay")
                    .templates(new ArrayList<>())
                    .addTemplatesItem(new V1alpha1Template()
                            .name("whalesay")
                            .container(new V1Container()
                                    .image("cowsay:v1")
                                    .command(Collections.singletonList("cowsay"))
                                    .args(Collections.singletonList("hello world")))));

    @Test
    public void testKubeAPI() throws Exception {
        V1alpha1Workflow created = new KubeAPI().createWorkflow(wf);
        assertNotNull(created.getMetadata());
        assertNotNull(created.getMetadata().getUid());
    }

    @Test
    public void testArgoServerAPI() throws Exception {
        V1alpha1Workflow created = new ArgoServerAPI().createWorkflow(wf);
        assertNotNull(created.getMetadata());
        assertNotNull(created.getMetadata().getUid());
    }
}
