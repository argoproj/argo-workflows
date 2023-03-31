# Title and Description

If you add specific title and description annotations to your workflow they will show up on the workflow lists.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-wf
  annotations:
    workflows.argoproj.io/title: 'Test Title'
    workflows.argoproj.io/description: 'Test Description'
```

Note:

- no title or description, defaults to `workflow.metadata.name`
- no title, description (title defaults to `workflow.metadata.name`)
- title, no description
- title and description

![Workflow Title And Description](assets/workflow-title-and-description.png)
