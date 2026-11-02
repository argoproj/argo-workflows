Description: Disable agent pod creation for plugins
Authors: [Gaurang Mishra](https://github.com/gaurang9991)
Component: General
Issues: 7891

Allow users to disable agent pod creation for plugins. Workflow Controller watches the task sets updated by external controllers or agents. Users should be careful when using this: when enabled, it stops creating default agent pods for HTTP templates.