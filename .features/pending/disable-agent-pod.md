Description: Disable agent pod creation for plugins
Authors: [Gaurang Mishra](https://github.com/gaurang9991)
Component: General
Issues: 7891

Allow users to disable agent pod creation for plugins. Workflow Controller watches the task sets updated by exeternal controllers or agents. User should be careful using this, when enabled it stop creating default agent pods for HTTP templates.