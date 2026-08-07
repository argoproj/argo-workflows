Description: Show artifact descriptions in the workflow submission UI
Authors: [panicboat](https://github.com/panicboat)
Component: UI
Issues: 12656

Input artifacts can now define a `description` that explains the expected file or purpose.
The workflow submission UI displays the description in a tooltip next to the artifact name.
Descriptions are available on the common `Artifact` API and remain available when workflow-level artifact overrides omit the field.
