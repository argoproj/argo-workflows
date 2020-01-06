# Workflow Achieve

![alpha](assets/alpha.svg)

For many uses, you may wish to keep workflows for a long time. Argo can save completed workflows to an SQL database. 

To enable this feature, configure a Postgres or MySQL database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `archive: true`.
