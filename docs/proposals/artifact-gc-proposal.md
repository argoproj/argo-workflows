# Proposal for Artifact Garbage Collection

## Introduction
The motivation for this is to enable users to automatically have certain Artifacts specified to be automatically garbage collected. 

Artifacts can be specified for Garbage Collection at different stages: `OnWorkflowCompletion`, `OnWorkflowDeletion`, `OnWorkflowSuccess`, `OnWorkflowFailure`, or `Never`

## Proposal Specifics

### Workflow Spec changes
1. `WorkflowSpec` has an `ArtifactGCStrategy`, which is the default for all artifacts: can be `OnWorkflowCompletion`, `OnWorkflowDeletion`, `OnWorkflowSuccess`, `OnWorkflowFailure`, or `Never`
2. Artifact has an `ArtifactGCStrategy` (only really applies to Output Artifacts), which can override the setting for `WorkflowSpec`
3. `WorkflowSpec` can add specification of Service Account and/or Annotations to be used for the Garbage Collection pods, to enable their access to the storage.

### Workflow Status changes
1. Artifact has a boolean `Deleted` flag
2. `WorkflowStatus.Conditions` can be set to `ArtifactGCError`

### Proposal Options

These [slides](../assets/artifact-gc-proposal.pptx) go over the trade offs in options that were presented to the Contributor meeting on 7/12/22.

### Decision

Following the meeting, these were the decisions:

We will go with Option 2 from that presentation. We'll have one Pod that runs in the user's namespace and deletes all artifacts pertaining to an individual Garbage Collection strategy. Since `OnWorkflowSuccess` happens at the same time as `OnWorkflowCompletion` and `OnWorkflowFailure` also happens at the same time as `OnWorkflowCompletion`, we can consider consolidating these GC Strategies together. 

We will use one or more `
` to specify the Templates (containing Artifacts), which the GC Pod will read and also update Status to (note individual artifacts have individual statuses). The Controller will read the Status and reflect that in the Workflow Status. The Controller will deem the `WorkflowTaskSets` ready to read once the Pod has completed (in success or failure).

Once the GC Pod has completed and the Workflow status has been persisted, assuming the Pod completed with Success, the Controller can delete the `WorkflowTaskSets`, which will cause the GC Pod to also get deleted as it will be "owned" by the `WorkflowTaskSets`.

The Workflow will have a Finalizer on it to prevent it from being deleted until Artifact GC has occurred. Once all deletions for all GC Strategies have occurred, the Controller will remove the Finalizer.

#### Failures

If a deletion fails, the Pod will retry a few times through exponential back off. Note: it will not be considered a failure if the key does not exist - the principal of idempotence will allow this (i.e. if a Pod were to get evicted and then re-run it should be okay if some artifacts were previously deleted).

Once it retries a few times, if it didn't succeed, it will end in a "Failed" state. The user will manually need to delete the `WorkflowTaskSets` (which will delete the GC Pod), and remove the Finalizer on the Workflow.

The Failure will be reflected in both the Workflow Conditions as well as as a Kubernetes Event (and the Artifacts that failed will have "Deleted"=false).

#### Pod vs Job

We'll use a Pod rather than a Job, and the Controller will be responsible for re-generating that Pod if it gets evicted or deleted.

#### Service Account/IAM roles

Slide 12 references the different options for passing Service Accounts and Annotations (sometimes containing IAM roles), which can provide access for deleting artifacts. We will go with Option 2 on that slide, in order to reduce the complexity of the code and reduce the potential number of Pods running. The user should set the Service Account or Annotations for Artifact GC as a whole.

#### MVP vs post-MVP

We will start with just S3. 

We can also make other determinations if it makes sense to postpone some parts for after MVP.

#### Workflow Spec Validation

We can reject the Workflow during validation if `ArtifactGC` is configured along with a non-supported storage engine (for now probably anything besides S3).


### Documentation

Need to clarify certain things in our documentation:
1. Users need to know that if they don't name their artifacts with unique keys, they risk the same key being deleted by one Workflow and created by another at the same time. One recommendation is to parameterize the key, e.g. `{{workflow.uid}}/hello.txt`.
2. Requirement to specify Service Account or Annotation for `ArtifactGC` specifically if they are needed (we won't fall back to default Workflow SA/annotations).


 
