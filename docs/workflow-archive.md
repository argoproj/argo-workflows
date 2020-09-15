# Workflow Archive

![GA](assets/ga.svg)

> v2.5 and after

For many uses, you may wish to keep workflows for a long time. Argo can save completed workflows to an SQL database. 

To enable this feature, configure a Postgres or MySQL (>= 5.7.8) database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `archive: true`.
