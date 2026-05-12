
Description: Support for global agent pod that can execute tasks from multiple workflows using label selectors
Authors: [Gaurang Mishra](https://github.com/gaurang9991)
Component: General
Issues: 7891

Enables a single global agent pod per service account to execute tasks from multiple workflows. Instead of creating one agent pod per workflow, users can opt-in to a shared agent pod model using same service account.The agent pod uses label selectors to watch and process WorkflowTaskSets from multiple workflows, the configuration also allows users to fully control the life cycle of the agent pod.

