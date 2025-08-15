Component: General
Issues: 14689
Description: Support update total parallelism without restart controller.
Author: [Shuangkun Tian](https://github.com/shuangkun)

When modify the global parallelism in workflow-controller-configmap, the change takes effect directly without restarting the controller.