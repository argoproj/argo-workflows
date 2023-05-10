# Workflow Archive

> v2.5 and after

If you want to keep completed workflows for a long time, you can use the workflow archive to save them in a Postgres or MySQL (>= 5.7.8) database.
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
