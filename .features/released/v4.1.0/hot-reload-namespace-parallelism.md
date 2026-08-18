Description: Hot-reload namespaceParallelism from the controller ConfigMap
Author: [shuangkun](https://github.com/shuangkun)
Component: General
Issues: 16490

Changes to `namespaceParallelism` in the workflow controller ConfigMap now take effect on reload without restarting the controller, matching the existing `parallelism` behavior.
Namespaces without an explicit parallelism label override track the live default.
