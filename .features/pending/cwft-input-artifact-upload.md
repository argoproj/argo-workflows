Description: Upload input artifacts when submitting ClusterWorkflowTemplates from the UI
Authors: [panicboat](https://github.com/panicboat)
Component: UI
Issues: 17290

When a ClusterWorkflowTemplate defines input artifacts in `spec.arguments.artifacts`, you can upload a file for each one from the UI submit panel.
The Argo Server stores the file under `uploads/{namespace}/{uuid}/{filename}` in the artifact repository of the namespace you submit to.
The upload endpoint accepts `?clusterScope=true` to target a ClusterWorkflowTemplate, and the existing URL for WorkflowTemplates is unchanged.
If you change the namespace after uploading, upload the files again.
