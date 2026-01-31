Component: UI
Issues: 14809
Description: Add Previous Runs section to Sensor details page to display workflows triggered by sensors
Author: [puretension](https://github.com/puretension)

- Added Previous Runs section below Sensor editor tabs using `workflows.argoproj.io/sensor` label filtering
- Implemented identical UI pattern as CronWorkflow with WorkflowDetailsList component for consistency
- Fixed empty state handling with proper array length check to display triggered workflows correctly
