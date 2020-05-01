# Workflow Creator

Today, is not possible for Argo Workflows to determine who created a workflow. 

The recommended approach is to add a label to your workflow, e.g.

```
argo submit -l creator=alex
``` 