# Java SDK

## Overview

The Java SDK provides model and APIs for accessing the Argo Server API rather.

If you wish to access the Kubernetes APIs, you can use the models to do this.

## Download

⚠️ The Java SDK is published to Github packages, not Maven Central. You must update your Maven settings.xml
file: [how to do that](https://github.com/argoproj/argo-workflows/packages).

Recommended:

```xml
<dependency>
    <groupId>io.argoproj.workflow</groupId>
    <artifactId>argo-client-java</artifactId>
    <version>3.3.0</version>
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

* [Event service](api/docs/EventServiceApi.md)
* [Sensor service](api/docs/SensorServiceApi.md)
* [Event source service](api/docs/EventSourceServiceApi.md)
* [Info service](api/docs/InfoServiceApi.md )
* [Pipeline service](api/docs/PipelineServiceApi.md)
* [Workflow service](api/docs/WorkflowServiceApi.md)
