Description: Allow for configuring the allow list when using workflow template refs.
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 16345

When the controller runs with `templateReferencing: Strict` or `Secure`, workflows using a `workflowTemplateRef` may only set an allow-listed set of `WorkflowSpec` fields (`arguments`, `entrypoint`, `suspend`, and other benign knobs); every other field is blocked so a submitter cannot override the template's security settings.
The `WORKFLOW_USER_OVERRIDE_ALLOWLIST` environment variable lets operators add fields to this allow-list.
Set it on the workflow-controller to a comma-separated list of `WorkflowSpec` field names, using the YAML/JSON names as written in a workflow, for example `WORKFLOW_USER_OVERRIDE_ALLOWLIST=podSpecPatch,volumes`.
Use this when your environment has decided a normally-blocked field is safe for submitters to override.
An unknown field name fails the controller at startup rather than being silently ignored, surfacing typos.
The nested `artifactGC.podSpecPatch`, `artifactGC.serviceAccountName`, and `artifactGC.podMetadata` fields remain blocked and are not relaxed by this variable.
