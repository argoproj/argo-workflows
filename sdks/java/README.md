# Java SDK

## Client Library

This provides model and APIs for accessing the Argo Server API rather.

If you wish to access the Kubernetes APIs, you can use the models to do this. You'll need to write your own code to
speak to the API.

⚠️ The Java SDK is published to GitHub Packages, not Maven Central. You must update your Maven `settings.xml`
file: [how to do that](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-apache-maven-registry).

Recommended:

```xml
<dependency>
    <groupId>io.argoproj.workflow</groupId>
    <artifactId>argo-client-java</artifactId>
    <version>v3.3.8</version>
</dependency>
```

The very latest version:

```xml
<dependency>
    <groupId>io.argoproj.workflow</groupId>
    <artifactId>argo-client-java</artifactId>
    <version>0.0.0-SNAPSHOT</version>
</dependency>
```

## Docs

* [Event service](client/docs/EventServiceApi.md)
* [Sensor service](client/docs/SensorServiceApi.md)
* [Event source service](client/docs/EventSourceServiceApi.md)
* [Info service](client/docs/InfoServiceApi.md )
* [Pipeline service](client/docs/PipelineServiceApi.md)
* [Workflow service](client/docs/WorkflowServiceApi.md)
