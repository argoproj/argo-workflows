<!-- Required: All of these fields are required, including at least one issue -->

Description: Add Previous Runs section to Sensor details page to display workflows triggered by sensors, matching CronWorkflow behavior for consistent user experience.
Author: [puretension](https://github.com/puretension)
Component: UI
Issues: 14809

<!--
Optional
Additional details about the feature written in markdown, aimed at users who want to learn about it
* Explain when you would want to use the feature
* Include code examples if applicable
  * Provide working examples
  * Format code using back-ticks
* Use Kubernetes style
* One sentence per line of markdown
-->

- Added Previous Runs section below Sensor editor tabs using `workflows.argoproj.io/sensor` label filtering
- Implemented identical UI pattern as CronWorkflow with WorkflowDetailsList component for consistency
- Fixed empty state handling with proper array length check to display triggered workflows correctly
