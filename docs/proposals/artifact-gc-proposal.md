notes:
- what it does: OnCompletion, OnDeletion
- how it works: 
- -- finalizer prevents deleting the Workflow until it's been done
- -- Do it in user namespace rather than Controller which should have the needed access permissions
- -- For a given Workflow, one Job or pod can perform deletion for all artifacts marked with "OnCompletion", and one job for OnDeletion. Pod can be uniquely named this way to prevent reinstantiating multiple times
- -- Need to pass to Job n number of Artifacts (JSON serialized) - if we put it in a ConfigMap and volume mount it max size is 1MB
- -- Could use argoexec
- -- Show image
todo: 
- what is the ConfigMap for artifacts currently used for?
- what to do if Job fails?



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
We can have a Job that deletes all artifacts for a Workflow that are set to "OnWorkflowCompletion" which runs once the Workflow completes. And we can have a separate Job that deletes all artifacts that are set to "OnWorkflowDeletion" which runs when the Workflow is being deleted. A Finalizer will be added to the Workflow which will only be removed once the Jobs are completed (todo: determine what to do if Job completes in failure). 

The Job can be uniquely named by the Workflow and the GC strategy, so we can easily query if it already exists and prevent it from being reinstantiated.

The Job can run argoexec, which can handle a new command for artifact deletion.

