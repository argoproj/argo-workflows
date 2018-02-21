# Changelog

## 2.1.0-alpha1 (Unreleased)
+ Support for DAG based definition of workflows
+ Add `spec.parallelism` field to limit concurrent pod execution at a workflow level
+ Add `template.parallelism` field to limit concurrent pod execution at a template level
+ Add `argo suspend`, `argo resume` to suspend and resume workflows
+ Add `argo resubmit` to resubmit a failed workflow
+ Add `instanceid` parameter support to `argo submit` command to submit workflow with controller's specific instance id label
+ Experimental support for resubmitting workflows with memoized steps
+ Improved parameters and output validation
+ UI migrated to React.
+ Workflow details page redesigned: added DAG view support, added workflow timeline tab.
+ Workflow details page enhancements: added sidecar containers details; workflow exist handler is available on DAG diagram and timeline view.
+ Support for pod tolerations (@discordianfish)
+ Make `workflow.namespace` available as a global variable (@vreon)
* Trim spaces from aws keys (@bodepd)
* Documentation fixes (@bodepd)
- Fix rbac resource versions in install (@dvavili)

## 2.0.0 (2018-02-06)
+ Add ability to specify affinity rules at both the workflow and template level
+ Add ability to specify imagePullSecrets in the workflow.spec
+ Generate OpenAPI models for the workflow spec
+ Support setting the UI base url
- Fix issue preventing the referencing of artifacts in a container with retries
- Fix issue preventing the use of volumes in a sidecar

## 2.0.0-beta1 (2018-01-18)
+ Use and install minimal RBAC ClusterRoles for workflow-controller and argo-ui deployments
+ Introduce `retryStrategy` field to control set retries for failed/errored containers
+ Introduce `raw` input artifacts
+ Add `argo install --dry-run` to print Kubernetes YAML manifests without installing
+ Add `argo list` sorts by running pods, then by completion time
+ Add `argo list -o wide` to show pod counts and parameter information
+ Add `argo list --running --completed --status` workflow filtering
+ Add `argo list --since DURATION` to filter workflows based on a time duration
+ Add ability for steps and resource templates to have outputs parameters
+ OpenID Connect auth support (@mthx)
* Increase controller rate limits for much faster processing of highly parallized workflows
* Executor sidecar hardening (retrying of Kube API queries)
* Switch to k8s-codegen generated workflow client and informer
* {{workflow.uuid}} variable corrected to {{workflow.uid}}
* Documentation fixes (@reasonthearchitect, @mthx)
- Prevent a potential k8s scheduler panic from incomplete setting of pod ownership reference
- Fix issues in controller operating on stale workflow state, and incorrectly identifying deleted pods

## 2.0.0-alpha3 (2018-01-02)
+ Introduce the "resource" template type for performing CRUD operations on k8s resources
+ Support for workflow exit handlers
+ Support artifactory as an artifact repository
+ Add ability to timeout a container/script using activeDeadlineSeconds
+ Add CLI command and flags to wait for a workflow to complete `argo wait`/`argo submit --wait`
+ Add ability to run multiple workflow controllers operating on separate instance ids
+ Add ability to run workflows using a specified service account
* Scalability improvements for highly parallelized workflows
* Improved validation of volume mounts with input artifacts
* Argo UI bug fixes and improvements
* Documentation fixes (@javierbq, @anshumanbh)
- Recover from unexpected panics when operating on workflows
- Fix a controller panic when using a script templates with input artifacts
- Fix issue preventing ability to pass JSON as a command line argument

## 2.0.0-alpha2 (2017-12-04)
* Argo release for KubeCon 2017

## 2.0.0-alpha1 (2017-11-16)
* Initial release of Argo as a Kubernetes CRD (presented at Bay Area Kubernetes Meetup)

## 1.1.0 (2017-11-08)
* Reduce sizes of axdb, zookeeper, kafka images by a combined total of ~1.7GB

## 1.0.1 (2017-10-04)
+ Add `argo app list` and `argo app show` commands
+ Add `argo job logs` for displaying and following job logs
- Fix issues preventing proper handling of input parameters and output artifacts with dynamic fixtures

## 1.0.0 (2017-07-23)
+ Initial release
