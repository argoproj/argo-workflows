Component: UI
Issues: 14809
Description: Add Previous Runs section to Sensor details page to display workflows triggered by sensors
Author: [puretension](https://github.com/puretension)

The Sensor details page now lists the Workflows that this Sensor's triggers created, below the Sensor editor.
Use it to check whether a Sensor is firing and to open the Workflows it produced, instead of going to the workflow list and filtering by label yourself.
When the Sensor has not triggered anything, the page tells you so.
The list works the same way as the one on the CronWorkflow details page.
