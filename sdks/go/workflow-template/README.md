# Workflow Template Example

This example demonstrates how to work with WorkflowTemplates - reusable workflow definitions.

## See first

Look at `basic-workflow` for a simpler starting example.

## What it does

- Creates or updates a WorkflowTemplate with parameters
- Submits a workflow using the template with default parameters
- Submits a workflow using the template with custom parameters
- Lists workflows created from the template
- Shows template details

## Running the example

```bash
# Use default kubeconfig
go run main.go

# Specify custom kubeconfig and namespace
go run main.go -kubeconfig /path/to/kubeconfig -namespace argo
```

## Expected output

```
=== Workflow Template Example ===

Step 1: Creating WorkflowTemplate...
✓ WorkflowTemplate 'hello-world-template' created

Step 2: Submitting workflow from template (default params)...
✓ Workflow 'from-template-default-abc123' submitted with default parameters

Step 3: Submitting workflow with custom parameters...
✓ Workflow 'from-template-custom-xyz789' submitted with custom message

Step 4: Listing workflows created from template...
Found 2 workflow(s) from template 'hello-world-template':
  1. from-template-default-abc123 (Running)
  2. from-template-custom-xyz789 (Pending)

Step 5: Template details:
  Name: hello-world-template
  Entrypoint: greet
  Templates: 1
  Parameters:
    - message = Hello World

=== Summary ===
✓ Created WorkflowTemplate: hello-world-template
✓ Submitted 2 workflows from template

View workflows with:
  argo get from-template-default-abc123 -n default
  argo get from-template-custom-xyz789 -n default

Cleanup with:
  kubectl delete workflowtemplate hello-world-template -n default
  kubectl delete workflow from-template-default-abc123 from-template-custom-xyz789 -n default
```

## Code walkthrough

1. **Create template**: Define a `WorkflowTemplate` with parameters
2. **Submit with defaults**: Create workflow referencing template
3. **Submit with custom params**: Override template parameters
4. **List workflows**: Find workflows created from template using label selector
5. **Get details**: Retrieve template information

## Key concepts

### WorkflowTemplate

A WorkflowTemplate is a reusable workflow definition:

```go
template := &wfv1.WorkflowTemplate{
    ObjectMeta: metav1.ObjectMeta{
        Name: "my-template",
    },
    Spec: wfv1.WorkflowSpec{
        Entrypoint: "main",
        Arguments: wfv1.Arguments{
            Parameters: []wfv1.Parameter{
                {Name: "message", Value: wfv1.AnyStringPtr("default")},
            },
        },
        Templates: []wfv1.Template{...},
    },
}
```

### Referencing Templates

Submit workflows by referencing the template:

```go
workflow := &wfv1.Workflow{
    ObjectMeta: metav1.ObjectMeta{
        GenerateName: "from-template-",
    },
    Spec: wfv1.WorkflowSpec{
        WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{
            Name: "my-template",
        },
    },
}
```

### Overriding Parameters

Pass custom parameters to override template defaults:

```go
workflow := &wfv1.Workflow{
    Spec: wfv1.WorkflowSpec{
        WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{
            Name: "my-template",
        },
        Arguments: wfv1.Arguments{
            Parameters: []wfv1.Parameter{
                {Name: "message", Value: wfv1.AnyStringPtr("custom value")},
            },
        },
    },
}
```

### Finding Workflows from Template

Workflows created from templates get automatic labels:

```go
labelSelector := fmt.Sprintf("workflows.argoproj.io/workflow-template=%s", templateName)
list, err := wfClient.List(ctx, metav1.ListOptions{
    LabelSelector: labelSelector,
})
```

## Benefits of WorkflowTemplates

1. **Reusability**: Define once, use many times
2. **Parameterization**: Customize behavior without changing template
3. **Version control**: Update template to affect all future workflows
4. **Organization**: Centralized workflow definitions
5. **Permissions**: Can grant template execution without modify permissions

## ClusterWorkflowTemplate

For cluster-wide templates, use `ClusterWorkflowTemplate`:

```go
cwftClient := clientset.ArgoprojV1alpha1().ClusterWorkflowTemplates()
clusterTemplate := &wfv1.ClusterWorkflowTemplate{
    ObjectMeta: metav1.ObjectMeta{
        Name: "cluster-template",
    },
    Spec: wfv1.WorkflowSpec{...},
}
```

## Next steps

- See `watch-workflow` for tracking workflow progress
- See `grpc-client` for remote Argo Server access
- Explore CronWorkflows for scheduled execution
