# Argo Workflows As CI

Argo Workflow has a number of features that mean you can use it for CI.

CI features:

* Visualizing a build pipelines
* Finding builds by labels
* Sharing a single work volume between steps
* Capture test results
* Uploading build artifacts
* Store and restore build caches (using optional artifacts)
* Building Docker images (using DIND)
* Running E2E tests (using daemons or sidecars)
* Garbage collecting old builds (using workflow GC)
* Exit handlers (e.g. sending Slack notification)
* Retry builds 

Examples:

* [ci.yaml](../examples/ci.yaml) - using shared work volume
* [influx-ci.yaml](../examples/influxdb-ci.yaml) - artifacts, capture test results, e2e tests
* [argo-ci.yaml](../examples/argo-ci.yaml) - build caches, capture test results, building Docker images, e2e tests, GC


For a complete CI solution:

* Trigger builds from Git using Argo Events.
* Mount secrets (e.g. Maven settings) using secrets.