# Proposal for Artifact Garbage Collection

## Introduction
The motivation for this is to enable users to automatically have certain Artifacts specified to be automatically garbage collected. 

Artifacts can be specified for GC at different stages: currently "OnWorkflowCompletion" and "OnWorkflowDeletion".

## Proposal Specifics

### Workflow Spec changes
1. WorkflowSpec has an ArtifactGCStrategy, which is the default for all artifacts: can be "OnWorkflowCompletion", "OnWorkflowDeletion", or "Never"
2. Artifact has an ArtifactGCStrategy (only really applies to Output Artifacts), which can override the setting for WorkflowSpec
3. Artifact has a boolean 'Deleted' flag

### Deletion Process
We can have a Job that runs in the user's namespace and deletes all artifacts for a Workflow that are set to "OnWorkflowCompletion" which runs once the Workflow completes. And we can have a separate Job that deletes all artifacts that are set to "OnWorkflowDeletion" which runs when the Workflow is being deleted. A Finalizer will be added to the Workflow which will only be removed once the Jobs are completed (todo: determine what to do if Job completes in failure). 

The Job can be uniquely named by the Workflow and the GC strategy, so we can easily query if it already exists and prevent it from being reinstantiated.

The Job can run argoexec, which can handle a new command for artifact deletion.

![Artifact GC Flow Chart](../assets/artifact-gc-proposal-flow-chart.png)

### Considerations
1. Passing the artifact spec to the GC Pod: we could JSON-serialize the spec for the artifacts, or if we want to reduce the byte count we could base64 encode it. We could pass it as an environment variable or volume mount a ConfigMap. Is there potential to reach the maximum size of environment variable or ConfigMap if the Workflow has thousands or tens of thousands of artifacts? Could look into having multiple environment variables or ConfigMaps...
2. How do we want to handle the Job failing? Could be a transient error. Allow some number of retries?
3. Should we use a Job or a standard Pod? (I guess if a node fails, Kubernetes will reschedule a Job, but not a Pod.)
