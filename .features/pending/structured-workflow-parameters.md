Description: Add opt-in structured workflow parameters, forms and reviewed retries
Authors: [jansuzhang](https://github.com/bjzhangshuai)
Component: General
Issues: 17221

Workflow authors can offer a form for nested inputs, compare saved versions and let callers review changed inputs before retrying selected tasks.
The experimental extension maps those inputs to existing WorkflowTemplates and preserves unrelated successful results and approvals when a supported retry can use the same Workflow.
Authors can also bind reusable actions and result views to individual calls.
The feature is disabled by default and requires separately installed CRDs; its API remains experimental.
See the Structured parameters user guide for setup, examples and supported limits.
