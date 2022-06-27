notes:
- what it does: OnCompletion, OnDeletion
- how it works: 
- -- finalizer prevents deleting the Workflow until it's been done
- -- Do it in user namespace rather than Controller which should have the needed access permissions
- -- For a given Workflow, one Job or pod can perform deletion for all artifacts marked with "OnCompletion", and one job for OnDeletion. Pod can be uniquely named this way to prevent reinstantiating multiple times
- -- Need to pass to Job n number of Artifacts (JSON serialized) - if we put it in a ConfigMap and volume mount it max size is 1MB
- -- Could use argoexec
- -- Show image


# Proposal for Artifact Garbage Collection

## Introduction
The motivation for this is to enable users to automatically have certain Artifacts specified to be automatically garbage collected. 

Artifacts can be specified for GC at different stages: currently "OnWorkflowCompletion" and "OnWorkflowDeletion".

## Proposal Specifics

