Description: Add metrics for the rate limiter
Authors: [Alan Clucas](https://github.com/Joibel)
Component: Telemetry
Issues: 15245

Add two rate limiter metrics to help us understand the effects:
  - the k8s API client rate limiter (enabled by default and set quite low, configurable via --qps)
  - and the resource rate limiter configured in the configmap and disabled by default.
These produce histogram metrics
