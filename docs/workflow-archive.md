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

Note that IAM-based authentication is not currently supported. However, you can start your database proxy as a sidecar
(e.g. via [CloudSQL Proxy](https://github.com/GoogleCloudPlatform/cloud-sql-proxy) on GCP) and then specify your local
proxy address, IAM username, and an empty string as your password in the persistence configuration to connect to it.

The following tables will be created in the database when you start the workflow controller with enabled archive:

* `argo_workflows`
* `argo_archived_workflows`
* `argo_archived_workflows_labels`
* `schema_history`

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

The database user/role must have `CREATE` and `USAGE` permissions on the schema of the database (either `public` in default or the configured value of `persistence.postgresql.schema`) so that the tables can be created during the migration.

When using a custom PostgreSQL schema via `persistence.postgresql.schema`, the user/role must also have permissions to create the schema.

The database user or role requires specific permissions on the database schema (either the default `public` schema or a custom schema defined via `persistence.postgresql.schema`) to ensure tables can be successfully created during migration.

If you are using a custom PostgreSQL schema, you must follow one of these two supported setup paths:

* Path 1: Automatic Schema Creation (Database-level permissions)  
If the custom schema does not yet exist and you want the migration process to create it automatically, you must grant the role `CREATE` privileges on the database.

* Path 2: Pre-created Schema (Schema-level permissions)  
If you prefer to manually create the custom schema beforehand, you must grant the role `USAGE` and `CREATE` privileges on that specific schema.

Important Note: Schema-level permissions alone are only applicable if the schema already exists. If the schema is absent, granting permissions on it will not work, and you must use the database-level permissions outlined in Path 1.

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

## Customizing PostgreSQL Database Schema

To change the schema for the tables in PostgreSQL, set `schema:` to the desired schema in the `persistence` section of [your configuration](workflow-controller-configmap.yaml).
Only available on PostgreSQL.

Example:

    persistence:
      postgresql:
        schema: argo
