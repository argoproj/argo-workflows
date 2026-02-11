# Workflow Controller ConfigMap

## Introduction

The Workflow Controller ConfigMap is used to set controller-wide settings.

For a detailed example, please see [`workflow-controller-configmap.yaml`](./workflow-controller-configmap.yaml).

## Alternate Structure

In all versions, the configuration may be under a `config: |` key:

```yaml
# This file describes the config settings available in the workflow controller configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  config: |
    instanceID: my-ci-controller
    artifactRepository:
      archiveLogs: true
      s3:
        endpoint: s3.amazonaws.com
        bucket: my-bucket
        region: us-west-2
        insecure: false
        accessKeySecret:
          name: my-s3-credentials
          key: accessKey
        secretKeySecret:
          name: my-s3-credentials
          key: secretKey

```

In version 2.7+, the `config: |` key is optional. However, if the `config: |` key is not used, all nested maps under top level
keys should be strings. This makes it easier to generate the map with some configuration management tools like Kustomize.

```yaml
# This file describes the config settings available in the workflow controller configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:                      # "config: |" key is optional in 2.7+!
  instanceID: my-ci-controller
  artifactRepository: |    # However, all nested maps must be strings
   archiveLogs: true
   s3:
     endpoint: s3.amazonaws.com
     bucket: my-bucket
     region: us-west-2
     insecure: false
     accessKeySecret:
       name: my-s3-credentials
       key: accessKey
     secretKeySecret:
       name: my-s3-credentials
       key: secretKey
```

## Config

Config contains the root of the configuration settings for the workflow controller as read from the ConfigMap called workflow-controller-configmap

### Fields

|         Field Name         |                                                 Field Type                                                  |                                                                                                                                                                                                                                                                                                                                                                         Description                                                                                                                                                                                                                                                                                                                                                                         |
|----------------------------|-------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `NodeEvents`               | [`NodeEvents`](#nodeevents)                                                                                 | NodeEvents configures how node events are emitted                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `WorkflowEvents`           | [`WorkflowEvents`](#workflowevents)                                                                         | WorkflowEvents configures how workflow events are emitted                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `Executor`                 | [`apiv1.Container`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#container-v1-core) | Executor holds container customizations for the executor to use when running pods                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `MainContainer`            | [`apiv1.Container`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#container-v1-core) | MainContainer holds container customization for the main container                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `KubeConfig`               | [`KubeConfig`](#kubeconfig)                                                                                 | KubeConfig specifies a kube config file for the wait & init containers                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `ArtifactRepository`       | [`wfv1.ArtifactRepository`](fields.md#artifactrepository)                                                   | ArtifactRepository contains the default location of an artifact repository for container artifacts                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `Namespace`                | `string`                                                                                                    | Namespace is a label selector filter to limit the controller's watch to a specific namespace                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `InstanceID`               | `string`                                                                                                    | InstanceID is a label selector to limit the controller's watch to a specific instance. It contains an arbitrary value that is carried forward into its pod labels, under the key workflows.argoproj.io/controller-instanceid, for the purposes of workflow segregation. This enables a controller to only receive workflow and pod events that it is interested about, in order to support multiple controllers in a single cluster, and ultimately allows the controller itself to be bundled as part of a higher level application. If omitted, the controller watches workflows and pods that *are not* labeled with an instance id. See [Scaling - Instance ID](https://argo-workflows.readthedocs.io/en/latest/scaling/#instance-id) for more details. |
| `MetricsConfig`            | [`MetricsConfig`](#metricsconfig)                                                                           | MetricsConfig specifies configuration for metrics emission. Metrics are enabled and emitted on localhost:9090/metrics by default.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `TelemetryConfig`          | [`MetricsConfig`](#metricsconfig)                                                                           | TelemetryConfig specifies configuration for telemetry emission. Telemetry is enabled and emitted in the same endpoint as metrics by default, but can be overridden using this config.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `Parallelism`              | `int`                                                                                                       | Parallelism limits the max total parallel workflows that can execute at the same time                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `NamespaceParallelism`     | `int`                                                                                                       | NamespaceParallelism limits the max workflows that can execute at the same time in a namespace                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `ResourceRateLimit`        | [`ResourceRateLimit`](#resourceratelimit)                                                                   | ResourceRateLimit limits the rate at which pods are created                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `Persistence`              | [`PersistConfig`](#persistconfig)                                                                           | Persistence contains the workflow persistence DB configuration                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `Links`                    | `Array<`[`Link`](fields.md#link)`>`                                                                         | Links to related apps.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `Columns`                  | `Array<`[`Column`](fields.md#column)`>`                                                                     | Columns are custom columns that will be exposed in the Workflow List View.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `WorkflowDefaults`         | [`wfv1.Workflow`](fields.md#workflow)                                                                       | WorkflowDefaults are values that will apply to all Workflows from this controller, unless overridden on the Workflow-level                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `PodSpecLogStrategy`       | [`PodSpecLogStrategy`](#podspeclogstrategy)                                                                 | PodSpecLogStrategy enables the logging of podspec on controller log.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `PodGCGracePeriodSeconds`  | `int64`                                                                                                     | PodGCGracePeriodSeconds specifies the duration in seconds before a terminating pod is forcefully killed. Value must be non-negative integer. A zero value indicates that the pod will be forcefully terminated immediately. Defaults to the Kubernetes default of 30 seconds.                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `PodGCDeleteDelayDuration` | [`metav1.Duration`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)  | PodGCDeleteDelayDuration specifies the duration before pods in the GC queue get deleted. Value must be non-negative. A zero value indicates that the pods will be deleted immediately. Defaults to 5 seconds.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `WorkflowRestrictions`     | [`WorkflowRestrictions`](#workflowrestrictions)                                                             | WorkflowRestrictions restricts the controller to executing Workflows that meet certain restrictions                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `InitialDelay`             | [`metav1.Duration`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)  | Adds configurable initial delay (for K8S clusters with mutating webhooks) to prevent workflow getting modified by MWC.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `Images`                   | `Map<string,`[`Image`](#image)`>`                                                                           | The command/args for each image, needed when the command is not specified and the emissary executor is used. https://argo-workflows.readthedocs.io/en/latest/workflow-executors/#emissary-emissary                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `RetentionPolicy`          | [`RetentionPolicy`](#retentionpolicy)                                                                       | Workflow retention by number of workflows                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `NavColor`                 | `string`                                                                                                    | NavColor is an ui navigation bar background color                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `SSO`                      | [`SSOConfig`](#ssoconfig)                                                                                   | SSO in settings for single-sign on                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `Synchronization`          | [`SyncConfig`](#syncconfig)                                                                                 | Synchronization via databases config                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `ArtifactDrivers`          | `Array<`[`ArtifactDriver`](#artifactdriver)`>`                                                              | ArtifactDrivers lists artifact driver plugins we can use                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `FailedPodRestart`         | [`FailedPodRestartConfig`](#failedpodrestartconfig)                                                         | FailedPodRestart configures automatic restart of pods that fail before entering Running state (e.g., due to Eviction, DiskPressure, Preemption). This allows recovery from transient infrastructure issues without requiring a retryStrategy on templates.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |

## NodeEvents

NodeEvents configures how node events are emitted

### Fields

| Field Name  | Field Type |                                                     Description                                                      |
|-------------|------------|----------------------------------------------------------------------------------------------------------------------|
| `Enabled`   | `bool`     | Enabled controls whether node events are emitted                                                                     |
| `SendAsPod` | `bool`     | SendAsPod emits events as if from the Pod instead of the Workflow with annotations linking the event to the Workflow |

## WorkflowEvents

WorkflowEvents configures how workflow events are emitted

### Fields

| Field Name | Field Type |                     Description                      |
|------------|------------|------------------------------------------------------|
| `Enabled`  | `bool`     | Enabled controls whether workflow events are emitted |

## KubeConfig

KubeConfig is used for wait & init sidecar containers to communicate with a k8s apiserver by an out-of-cluster method; it is used when the workflow controller is in a different cluster from the workflow workloads

### Fields

|  Field Name  | Field Type |                                    Description                                     |
|--------------|------------|------------------------------------------------------------------------------------|
| `SecretName` | `string`   | SecretName of the kubeconfig secret may not be empty if kuebConfig specified       |
| `SecretKey`  | `string`   | SecretKey of the kubeconfig in the secret may not be empty if kubeConfig specified |
| `VolumeName` | `string`   | VolumeName of kubeconfig, default to 'kubeconfig'                                  |
| `MountPath`  | `string`   | MountPath of the kubeconfig secret, default to '/kube/config'                      |

## MetricsConfig

MetricsConfig defines a config for a metrics server

### Fields

|   Field Name    |                                                                                               Field Type                                                                                                |                                                                          Description                                                                           |
|-----------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Enabled`       | `bool`                                                                                                                                                                                                  | Enabled controls metric emission. Default is true, set "enabled: false" to turn off                                                                            |
| `DisableLegacy` | `bool`                                                                                                                                                                                                  | DisableLegacy turns off legacy metrics. Deprecated: Legacy metrics are now removed, this field is ignored.                                                     |
| `MetricsTTL`    | `TTL` (time.Duration forces you to specify in millis, and does not support days see https://stackoverflow.com/questions/48050945/how-to-unmarshal-json-into-durations (underlying type: time.Duration)) | MetricsTTL sets how often custom metrics are cleared from memory                                                                                               |
| `Path`          | `string`                                                                                                                                                                                                | Path is the path where metrics are emitted. Must start with a "/". Default is "/metrics"                                                                       |
| `Port`          | `int`                                                                                                                                                                                                   | Port is the port where metrics are emitted. Default is "9090"                                                                                                  |
| `IgnoreErrors`  | `bool`                                                                                                                                                                                                  | IgnoreErrors is a flag that instructs prometheus to ignore metric emission errors                                                                              |
| `Secure`        | `bool`                                                                                                                                                                                                  | Secure is a flag that starts the metrics servers using TLS, defaults to true                                                                                   |
| `Modifiers`     | `Map<string,`[`MetricModifier`](#metricmodifier)`>`                                                                                                                                                     | Modifiers configure metrics by name                                                                                                                            |
| `Temporality`   | `MetricsTemporality` (MetricsTemporality defines the temporality of OpenTelemetry metrics (underlying type: string))                                                                                    | Temporality of the OpenTelemetry metrics. Enum of Cumulative or Delta, defaulting to Cumulative. No effect on Prometheus metrics, which are always Cumulative. |

## MetricModifier

MetricModifier are modifiers for an individual named metric to change their behaviour

### Fields

|      Field Name      |    Field Type    |                                                 Description                                                  |
|----------------------|------------------|--------------------------------------------------------------------------------------------------------------|
| `Disabled`           | `bool`           | Disabled disables the emission of this metric completely                                                     |
| `DisabledAttributes` | `Array<string>`  | DisabledAttributes lists labels for this metric to remove those attributes to save on cardinality            |
| `HistogramBuckets`   | `Array<float64>` | HistogramBuckets allow configuring of the buckets used in a histogram Has no effect on non-histogram buckets |

## ResourceRateLimit

### Fields

| Field Name | Field Type |                      Description                       |
|------------|------------|--------------------------------------------------------|
| `Limit`    | `float64`  | Limit is the maximum rate at which pods can be created |
| `Burst`    | `int`      | Burst allows temporary spikes above the limit          |

## PersistConfig

PersistConfig contains workflow persistence configuration

### Fields

|       Field Name       |                                                                                               Field Type                                                                                                |                                                 Description                                                 |
|------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------|
| `PostgreSQL`           | [`PostgreSQLConfig`](#postgresqlconfig)                                                                                                                                                                 | PostgreSQL configuration for PostgreSQL database, don't use MySQL at the same time                          |
| `MySQL`                | [`MySQLConfig`](#mysqlconfig)                                                                                                                                                                           | MySQL configuration for MySQL database, don't use PostgreSQL at the same time                               |
| `ConnectionPool`       | [`ConnectionPool`](#connectionpool)                                                                                                                                                                     | Pooled connection settings for all types of database connections                                            |
| `DBReconnectConfig`    | [`DBReconnectConfig`](#dbreconnectconfig)                                                                                                                                                               | DBReconnectConfig are configuration options for database retries and reconnections                          |
| `NodeStatusOffload`    | `bool`                                                                                                                                                                                                  | NodeStatusOffload saves node status only to the persistence DB to avoid the 1MB limit in etcd               |
| `Archive`              | `bool`                                                                                                                                                                                                  | Archive completed and Workflows to persistence so you can access them after they're removed from kubernetes |
| `ArchiveLabelSelector` | [`metav1.LabelSelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#labelselector-v1-meta)                                                                                    | ArchiveLabelSelector holds LabelSelector to determine which Workflows to archive                            |
| `ArchiveTTL`           | `TTL` (time.Duration forces you to specify in millis, and does not support days see https://stackoverflow.com/questions/48050945/how-to-unmarshal-json-into-durations (underlying type: time.Duration)) | ArchiveTTL is the time to live for archived Workflows                                                       |
| `ClusterName`          | `string`                                                                                                                                                                                                | ClusterName is the name of the cluster (or technically controller) for the persistence database             |
| `SkipMigration`        | `bool`                                                                                                                                                                                                  | SkipMigration skips database migration even if needed                                                       |

## PostgreSQLConfig

PostgreSQLConfig contains PostgreSQL-specific database configuration

### Fields

|    Field Name    |                                                         Field Type                                                          |                                Description                                |
|------------------|-----------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------|
| `Host`           | `string`                                                                                                                    | Host is the database server hostname                                      |
| `Port`           | `int`                                                                                                                       | Port is the database server port                                          |
| `Database`       | `string`                                                                                                                    | Database is the name of the database to connect to                        |
| `TableName`      | `string`                                                                                                                    | TableName is the name of the table to use, must be set                    |
| `UsernameSecret` | [`apiv1.SecretKeySelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | UsernameSecret references a secret containing the database username       |
| `PasswordSecret` | [`apiv1.SecretKeySelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | PasswordSecret references a secret containing the database password       |
| `SSL`            | `bool`                                                                                                                      | SSL enables SSL connection to the database                                |
| `SSLMode`        | `string`                                                                                                                    | SSLMode specifies the SSL mode (disable, require, verify-ca, verify-full) |
| `AzureToken`     | [`AzureTokenConfig`](#azuretokenconfig)                                                                                     | AzureToken specifies if the password should be fetched as an Azure token  |

## AzureTokenConfig

### Fields

| Field Name | Field Type |                                                       Description                                                       |
|------------|------------|-------------------------------------------------------------------------------------------------------------------------|
| `Enabled`  | `bool`     | Enabled enables Azure token fetching                                                                                    |
| `Scope`    | `string`   | Scope is the scope to request the token for. Defaults to "https://ossrdbms-aad.database.windows.net/.default" if empty. |

## MySQLConfig

MySQLConfig contains MySQL-specific database configuration

### Fields

|    Field Name    |                                                         Field Type                                                          |                             Description                             |
|------------------|-----------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------|
| `Host`           | `string`                                                                                                                    | Host is the database server hostname                                |
| `Port`           | `int`                                                                                                                       | Port is the database server port                                    |
| `Database`       | `string`                                                                                                                    | Database is the name of the database to connect to                  |
| `TableName`      | `string`                                                                                                                    | TableName is the name of the table to use, must be set              |
| `UsernameSecret` | [`apiv1.SecretKeySelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | UsernameSecret references a secret containing the database username |
| `PasswordSecret` | [`apiv1.SecretKeySelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | PasswordSecret references a secret containing the database password |
| `Options`        | `Map<string,string>`                                                                                                        | Options contains additional MySQL connection options                |

## ConnectionPool

ConnectionPool contains database connection pool settings

### Fields

|    Field Name     |                                                                                               Field Type                                                                                                |                                Description                                 |
|-------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------|
| `MaxIdleConns`    | `int`                                                                                                                                                                                                   | MaxIdleConns sets the maximum number of idle connections in the pool       |
| `MaxOpenConns`    | `int`                                                                                                                                                                                                   | MaxOpenConns sets the maximum number of open connections to the database   |
| `ConnMaxLifetime` | `TTL` (time.Duration forces you to specify in millis, and does not support days see https://stackoverflow.com/questions/48050945/how-to-unmarshal-json-into-durations (underlying type: time.Duration)) | ConnMaxLifetime sets the maximum amount of time a connection may be reused |

## DBReconnectConfig

DBReconnectConfig contains database reconnect settings

### Fields

|     Field Name     | Field Type |                                                 Description                                                 |
|--------------------|------------|-------------------------------------------------------------------------------------------------------------|
| `MaxRetries`       | `int`      | MaxRetries defines how many connection attempts should be made before we give up                            |
| `BaseDelaySeconds` | `int`      | BaseDelaySeconds delays retries by this amount multiplied by the retryMultiple, capped to `maxDelaySeconds` |
| `MaxDelaySeconds`  | `int`      | MaxDelaySeconds the absolute upper limit to wait before retrying                                            |
| `RetryMultiple`    | `float64`  | RetryMultiple is the growth factor for `baseDelaySeconds`                                                   |

## PodSpecLogStrategy

PodSpecLogStrategy contains the configuration for logging the pod spec in controller log for debugging purpose

### Fields

| Field Name  | Field Type | Description |
|-------------|------------|-------------|
| `FailedPod` | `bool`     | -           |
| `AllPods`   | `bool`     | -           |

## WorkflowRestrictions

WorkflowRestrictions contains restrictions for workflow execution

### Fields

|      Field Name       |                                                         Field Type                                                         |                         Description                          |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------|
| `TemplateReferencing` | `TemplateReferencing` (TemplateReferencing defines how templates can be referenced in workflows (underlying type: string)) | TemplateReferencing controls how templates can be referenced |

## Image

Image contains command and entrypoint configuration for container images

### Fields

|  Field Name  |   Field Type    |                  Description                  |
|--------------|-----------------|-----------------------------------------------|
| `Entrypoint` | `Array<string>` | Entrypoint overrides the container entrypoint |
| `Cmd`        | `Array<string>` | Cmd overrides the container command           |

## RetentionPolicy

Workflow retention by number of workflows

### Fields

| Field Name  | Field Type |                       Description                        |
|-------------|------------|----------------------------------------------------------|
| `Completed` | `int`      | Completed is the number of completed Workflows to retain |
| `Failed`    | `int`      | Failed is the number of failed Workflows to retain       |
| `Errored`   | `int`      | Errored is the number of errored Workflows to retain     |

## SSOConfig

SSOConfig contains single sign-on configuration settings

### Fields

|       Field Name       |                                                         Field Type                                                          |                            Description                             |
|------------------------|-----------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------|
| `Issuer`               | `string`                                                                                                                    | Issuer is the OIDC issuer URL                                      |
| `IssuerAlias`          | `string`                                                                                                                    | IssuerAlias is an optional alias for the issuer                    |
| `ClientID`             | [`apiv1.SecretKeySelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | ClientID references a secret containing the OIDC client ID         |
| `ClientSecret`         | [`apiv1.SecretKeySelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | ClientSecret references a secret containing the OIDC client secret |
| `RedirectURL`          | `string`                                                                                                                    | RedirectURL is the OIDC redirect URL                               |
| `RBAC`                 | [`RBACConfig`](#rbacconfig)                                                                                                 | RBAC contains role-based access control settings                   |
| `Scopes`               | `Array<string>`                                                                                                             | additional scopes (on top of "openid")                             |
| `SessionExpiry`        | [`metav1.Duration`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)                  | SessionExpiry specifies how long user sessions last                |
| `CustomGroupClaimName` | `string`                                                                                                                    | CustomGroupClaimName will override the groups claim name           |
| `UserInfoPath`         | `string`                                                                                                                    | UserInfoPath specifies the path to user info endpoint              |
| `InsecureSkipVerify`   | `bool`                                                                                                                      | InsecureSkipVerify skips TLS certificate verification              |
| `FilterGroupsRegex`    | `Array<string>`                                                                                                             | FilterGroupsRegex filters groups using regular expressions         |
| `RootCA`               | `string`                                                                                                                    | custom PEM encoded CA certificate file contents                    |

## RBACConfig

RBACConfig contains role-based access control configuration

### Fields

| Field Name | Field Type |               Description                |
|------------|------------|------------------------------------------|
| `Enabled`  | `bool`     | Enabled controls whether RBAC is enabled |

## SyncConfig

SyncConfig contains synchronization configuration for database locks (semaphores and mutexes)

### Fields

|          Field Name          |                Field Type                 |                                                                                                                Description                                                                                                                 |
|------------------------------|-------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `PostgreSQL`                 | [`PostgreSQLConfig`](#postgresqlconfig)   | PostgreSQL configuration for PostgreSQL database, don't use MySQL at the same time                                                                                                                                                         |
| `MySQL`                      | [`MySQLConfig`](#mysqlconfig)             | MySQL configuration for MySQL database, don't use PostgreSQL at the same time                                                                                                                                                              |
| `ConnectionPool`             | [`ConnectionPool`](#connectionpool)       | Pooled connection settings for all types of database connections                                                                                                                                                                           |
| `DBReconnectConfig`          | [`DBReconnectConfig`](#dbreconnectconfig) | DBReconnectConfig are configuration options for database retries and reconnections                                                                                                                                                         |
| `EnableAPI`                  | `bool`                                    | EnableAPI enables the database synchronization API                                                                                                                                                                                         |
| `ControllerName`             | `string`                                  | ControllerName sets a unique name for this controller instance                                                                                                                                                                             |
| `SkipMigration`              | `bool`                                    | SkipMigration skips database migration if needed                                                                                                                                                                                           |
| `LimitTableName`             | `string`                                  | LimitTableName customizes the table name for semaphore limits, if not set, the default value is "sync_limit"                                                                                                                               |
| `StateTableName`             | `string`                                  | StateTableName customizes the table name for current lock state, if not set, the default value is "sync_state"                                                                                                                             |
| `ControllerTableName`        | `string`                                  | ControllerTableName customizes the table name for controller heartbeats, if not set, the default value is "sync_controller"                                                                                                                |
| `LockTableName`              | `string`                                  | LockTableName customizes the table name for lock coordination data, if not set, the default value is "sync_lock"                                                                                                                           |
| `PollSeconds`                | `int`                                     | PollSeconds specifies how often to check for lock changes, if not set, the default value is 5 seconds                                                                                                                                      |
| `HeartbeatSeconds`           | `int`                                     | HeartbeatSeconds specifies how often to update controller heartbeat, if not set, the default value is 60 seconds                                                                                                                           |
| `InactiveControllerSeconds`  | `int`                                     | InactiveControllerSeconds specifies when to consider a controller dead, if not set, the default value is 300 seconds                                                                                                                       |
| `SemaphoreLimitCacheSeconds` | `int64`                                   | SemaphoreLimitCacheSeconds specifies the duration in seconds before the workflow controller will re-fetch the limit for a semaphore from its associated data source. Defaults to 0 seconds (re-fetch every time the semaphore is checked). |

## ArtifactDriver

ArtifactDriver is a plugin for an artifact driver

### Fields

|         Field Name         |                           Field Type                            |                                           Description                                            |
|----------------------------|-----------------------------------------------------------------|--------------------------------------------------------------------------------------------------|
| `Name`                     | `wfv1.ArtifactPluginName` (string (name of an artifact plugin)) | Name is the name of the artifact driver plugin                                                   |
| `Image`                    | `string`                                                        | Image is the docker image of the artifact driver                                                 |
| `ConnectionTimeoutSeconds` | `int32`                                                         | ConnectionTimeoutSeconds is the timeout for the artifact driver connection, 5 seconds if not set |

## FailedPodRestartConfig

FailedPodRestartConfig configures automatic restart of pods that fail before entering Running state. This is useful for recovering from transient infrastructure issues like node eviction due to DiskPressure or MemoryPressure without requiring a retryStrategy on every template.

### Fields

|  Field Name   | Field Type |                                                                                                                        Description                                                                                                                        |
|---------------|------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Enabled`     | `bool`     | Enabled enables automatic restart of pods that fail before entering Running state. When enabled, pods that fail due to infrastructure issues (like eviction) without ever running their main container will be automatically recreated. Default is false. |
| `MaxRestarts` | `int32`    | MaxRestarts is the maximum number of automatic restarts per node before giving up. This prevents infinite restart loops. Default is 3.                                                                                                                    |
