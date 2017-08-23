# Roadmap

* Make it easier to use and contribute to the proejct.
* Install Argo on any existing k8s cluster.
* Integrate Argo with k8s RBAC & secrets.
* Support for running (KinK) Kubernetes in Kubernetes.

# History

* M1: Nov 2015
  * Complete "hardwired" CI/CD workflow for a simple web application.
  * Using Mesos and rabbitmq/celery for the workflow engine.
* M2: Jan 2016
  * Persistent volume support using flocker.
  * Initial "Cashboard" implementation.
  * Automated installer for AWS.
* M3: May 2016
  * GUI.
  * GUI-based DSL.
  * Artifacts for workflows.
  * Container log management.
  * Cluster autoscaling.
  * Many, many volume management bugs.
* M4: Jul 2016
  * Nested workflows.
  * Time-based job scheduling.
* M5: Oct 2016
  * Switched to K8s.
  * Spot instances.
  * Fixtures.
  * Improve artifacts.
  * YAML DSL.
  * Email notificaiton.
  * Non-disruptive upgrades of platform software.
  * Make flocker really work on AWS
* M6: Dec 2016
  * Scale AXDB.
  * Scale internal event handling
  * Performance
  * Run chaos monkey
  * Hardening.
* M7: Mar 2017
  * AppStore.
  * Spot instances.
  * Artifact management.
  * Deployment.
  * Improved artifact management.
  * Improve non-disruptive upgrade.
* M8: May 2017
  * Persistent volumes.
  * Notification center.
  * Secret management.
* M9: Jun 2017
  * Rolling upgrade of deployments.
  * Secret management v2.
  * Remove rabbitmq.
  * Managed ELBs.
  * Prometheus.
  * Managed fixtures (RDS, VM).
  * Initial GCP/GKE support.
* M10: Aug 2017
  * Ready to release to the world!
  * Remove rabbitmq.
  * YAML checker v2
  * Kubernetes 1.6.
  * Dev CLI tool.
  * Move to kops
