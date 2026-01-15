Description: Upload input artifacts when submitting workflows from the UI
Authors: [panicboat](https://github.com/panicboat)
Component: UI
Issues: 12656

When a WorkflowTemplate defines input artifacts in `spec.arguments.artifacts`, users can now upload files directly from the UI when submitting the workflow.

Previously, users had to manually upload files to the artifact repository, know the exact key path, and hard-code the key in the WorkflowTemplate.

Now, users can simply select a file in the submit dialog.
The system will upload the file to the artifact repository via the Argo Server, automatically override the artifact key with the uploaded file's location, and submit the workflow with the correct artifact configuration.

This feature works with all supported artifact repositories (S3, GCS, Azure Blob Storage, OSS, HDFS).
