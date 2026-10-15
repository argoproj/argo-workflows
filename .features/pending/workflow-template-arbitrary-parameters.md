Description: Add parameters the template doesn't declare when submitting a WorkflowTemplate or ClusterWorkflowTemplate from the UI
Authors: [Ruslan Shaydullin](https://github.com/ruslan-shaydullin)
Component: UI
Issues: 13035

Use "Add a parameter" on the "Submit Workflow" panel when a template references `{{workflow.parameters.<name>}}` for a parameter it doesn't declare, such as one with no sensible default.
