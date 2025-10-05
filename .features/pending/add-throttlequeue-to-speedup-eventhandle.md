Component: General
Issues: 14791
Description: Add wfThrottleQueue to accelerate event handling.
Author: [Shuangkun Tian](https://github.com/shuangkun)

In large-scale scenarios, the throttler's concurrent count calculation can become a bottleneck.
This feature improves performance by decoupling event reception from processing.
The new `wfThrottleQueue` allows the controller to handle workflow events more efficiently by separating throttle operations from the main workflow processing queue.
This reduces contention and improves throughput under high load conditions.
The feature is automatically enabled and can be configured using the `--workflow-throttle-workers` parameter. 