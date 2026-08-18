# Workflow Archive

> v2.5 and after

If you want to keep completed workflows for a long time, you can use the workflow archive to save them in a Postgres (>=9.4), MySQL (>= 5.7.8), or MariaDB (>= 10.2) database.
The workflow archive stores the status of the workflow, which pods have been executed, what was the result etc.
The job logs of the workflow pods will not be archived.
If you need to save the logs of the pods, you must setup an [artifact repository](artifact-repository-ref.md) according to [this doc](configure-artifact-repository.md).

The quick-start deployment includes a Postgres database server.
In this case the workflow archive is already enabled.
Such a deployment is convenient for test environments, but in a production environment you must use a production quality database service.

## Enabling Workflow Archive

To enable archiving of the workflows, you must configure database parameters in the `persistence` section of [your configuration](workflow-controller-configmap.yaml) and set `archive:` to `true`.

Example:

    persistence:
      archive: true
      postgresql:
        host: localhost
        port: 5432
        database: postgres
        tableName: argo_workflows
        userNameSecret:
          name: argo-postgres-config
          key: username
        passwordSecret:
          name: argo-postgres-config
          key: password

You must also create the secret with database user and password in the namespace of the workflow controller.

Example:

    kubectl create secret generic argo-postgres-config -n argo --from-literal=password=mypassword --from-literal=username=argodbuser

Instead of a static password, you can authenticate to some managed PostgreSQL services with short-lived tokens.
See [IAM-based Authentication](#iam-based-authentication).

The following tables will be created in the database when you start the workflow controller with enabled archive:

* `argo_workflows`
* `argo_archived_workflows`
* `argo_archived_workflows_labels`
* `schema_history`

## IAM-based Authentication

> v4.1 and after

For PostgreSQL, the controller can authenticate to the database with short-lived cloud IAM tokens instead of a static password.
Use this to avoid storing long-lived database passwords in Kubernetes secrets.
When token authentication is enabled the `passwordSecret` is not used, but you must still provide a `userNameSecret` containing the database user that is mapped to your cloud identity.
Only one token mechanism can be enabled at a time.

These options are part of the shared database configuration, so they also apply to the databases used for [synchronization](synchronization.md#database-configuration) and [node status offloading](offloading-large-workflows.md).

### Microsoft Entra ID on Azure

To connect to an Azure Database for PostgreSQL using [Microsoft Entra ID](https://learn.microsoft.com/en-us/azure/postgresql/flexible-server/concepts-azure-ad-authentication), enable `azureToken`:

    persistence:
      archive: true
      postgresql:
        host: example.postgres.database.azure.com
        port: 5432
        database: postgres
        tableName: argo_workflows
        ssl: true
        sslMode: require
        userNameSecret:
          name: argo-postgres-config
          key: username
        azureToken:
          enabled: true

The controller requests a token for each new database connection using the default Azure credential chain, which supports Workload Identity, Managed Identity and other standard mechanisms.
The token scope defaults to `https://ossrdbms-aad.database.windows.net/.default` and can be overridden with `azureToken.scope`.

### AWS RDS IAM

To connect to an AWS RDS for PostgreSQL database using [IAM database authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAM.html), enable `awsRDSToken`:

    persistence:
      archive: true
      postgresql:
        host: example.eu-west-1.rds.amazonaws.com
        port: 5432
        database: postgres
        tableName: argo_workflows
        ssl: true
        userNameSecret:
          name: argo-postgres-config
          key: username
        awsRDSToken:
          enabled: true
          region: eu-west-1

The controller requests an IAM authentication token for each new database connection using the default AWS credential chain, which supports IAM Roles for Service Accounts (IRSA), instance profiles and other standard mechanisms.
The `region` field is optional and is auto-detected from the environment if omitted.
You must enable SSL with `ssl: true` to use AWS RDS IAM authentication.

### Other providers

For providers without direct support, you can start your database proxy as a sidecar
(e.g. via [CloudSQL Proxy](https://github.com/GoogleCloudPlatform/cloud-sql-proxy) on GCP) and then specify your local
proxy address, IAM username, and an empty string as your password in the persistence configuration to connect to it.

## Automatic Database Migration

Every time the Argo workflow-controller starts with persistence enabled, it tries to migrate the database to the correct version.
If the database migration fails, the workflow-controller will also fail to start.
In this case you can delete all the above tables and restart the workflow-controller.

If you know what are you doing you also have an option to skip migration:

    persistence:
      skipMigration: true

For the list of SQL statements applied during migration, see [Database Migrations](database-migrations.md).

## Required database permissions

### Postgres

The database user/role must have `CREATE` and `USAGE` permissions on the `public` schema of the database so that the tables can be created during the migration.

## Archive TTL

You can configure the time period to keep archived workflows before they will be deleted by the archived workflow garbage collection function.
The default is forever.

Example:

    persistence:
      archiveTTL: 10d

The `ARCHIVED_WORKFLOW_GC_PERIOD` variable defines the periodicity of running the garbage collection function.
The default value is documented [here](environment-variables.md).
When the workflow controller starts, it sets the ticker to run every `ARCHIVED_WORKFLOW_GC_PERIOD`.
It does not run the garbage collection function immediately and the first garbage collection happens only after the period defined in the `ARCHIVED_WORKFLOW_GC_PERIOD` variable.

## Cluster Name

Optionally you can set a unique name of your Kubernetes cluster. This name will populate the `clustername` field in the `argo_archived_workflows` table.

Example:

    persistence:
      clusterName: dev-cluster

## Disabling Workflow Archive

To disable archiving of the workflows, set `archive:` to  `false` in the `persistence` section of [your configuration](workflow-controller-configmap.yaml).

Example:

    persistence:
      archive: false
