# Workflow Archive

![GA](assets/ga.svg)

> v2.5 and after

For many uses, you may wish to keep workflows for a long time. Argo can save completed workflows to an SQL database. 

To enable this feature, configure a Postgres or MySQL (>= 5.7.8) database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `archive: true`.

Be aware that this feature will only archive the statuses of the workflows (which pods have been executed, what was the result, ...)

However, the logs of each pod will NOT be archived. If you need to access the logs of the pods, you need to setup [an artifact repository](artifact-repository-ref.md) thanks to [this doc](configure-artifact-repository.md)
