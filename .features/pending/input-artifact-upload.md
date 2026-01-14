Description: Upload input artifacts when submitting workflows from the UI
Authors: [panicboat](https://github.com/panicboat)
Component: UI
Issues: 12656

When a WorkflowTemplate defines input artifacts in `spec.arguments.artifacts`, users can now upload files directly from the UI when submitting the workflow.

Previously, users had to:
1. Manually upload files to the artifact repository
2. Know the exact key path
3. Hard-code the key in the WorkflowTemplate

Now, users can simply select a file in the submit dialog, and the system will:
1. Upload the file to the artifact repository via the Argo Server
2. Automatically override the artifact key with the uploaded file's location
3. Submit the workflow with the correct artifact configuration

This feature works with all supported artifact repositories (S3, GCS, Azure Blob Storage, OSS, HDFS).
