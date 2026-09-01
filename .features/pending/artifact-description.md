Description: Show artifact descriptions in the workflow submission UI
Authors: [panicboat](https://github.com/panicboat)
Component: UI
Issues: 16573

Workflow argument artifacts (`spec.arguments.artifacts`) can now define a `description` that explains the expected file or purpose.
When submitting from a WorkflowTemplate detail page, the submission panel displays the description in a tooltip next to the artifact name.
Descriptions are available on the common `Artifact` API and remain available when workflow-level artifact overrides omit the field.
