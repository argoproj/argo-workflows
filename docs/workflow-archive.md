# Workflow Archive

![GA](assets/ga.svg)

> v2.5 and after

For many uses, you may wish to keep workflows for a long time. Argo can save completed workflows to an SQL database. 

To enable this feature, configure a Postgres or MySQL (>= 5.7.8) database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `archive: true`.

Be aware that this feature will only archive the statuses of the workflows (which pods have been executed, what was the result, ...)

However, the logs of each pod will NOT be archived. If you need to access the logs of the pods, you need to setup [an artifact repository](artifact-repository-ref.md) thanks to [this doc](configure-artifact-repository.md).

In addition the table specified in the configmap above, the following tables are created when enabling archiving:

* argo_archived_workflows
* argo_archived_workflows_labels
* schema_history

The database migration will only occur successfully if none of the tables exist. If a partial set of the tables exist, the database migration may fail and the Argo workflow-controller pod may fail to start. If this occurs delete all of the tables and try restarting the deployment.

## Required database permissions

### Postgres
The database user/role needs to have `CREATE` and `USAGE` permissions on the `public` schema of the database so that the necessary table can be generated during the migration.
