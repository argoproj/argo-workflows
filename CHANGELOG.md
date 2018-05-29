# Changelog

## 2.1.1 (2018-05-29)

### Changelog since v2.1.0
- Switch to an UnstructuredInformer to guard controller against malformed workflow manifests (issue #632)
- Fix issue where suspend templates were not properly being connected to their children (issue #869)
- Fix issue where a failed step in a template with parallelism would not complete (issue #868)
- Fix issue where `argo list` age column maxed out at 1d (issue #857)
- Fix issue where volumes were not supported in script templates (issue #852)
- Fix implementation of DAG task targets (issue #865)
- Retrying failed steps templates could potentially result in disconnected children
- [UI] Fix crash while rendering failed workflow with exit handler (issue #815)
- [UI] Fix locating outbound nodes for skipped node
- [UI] Fix JS crash caused by inconsistent workflow state
- [UI] Fix blank help page when using browser navigation
- [UI] API server can filter workflows managed by specific workflow controller (@kzadorozhny)
- [UI] Restore support for accessing the UI using `kubectl proxy` (@mthx)
- [UI] Pass the namespace when querying the logs (issue #777) (@mthx)
- [UI] Improve workflow sorting (issue #866)
+ Add windows support for Argo CLI (@cuericlee)
* Documentation fixes (@mthx, @bodepd)

## 2.1.0 (2018-05-01)

### Changelog since v2.0
+ Support for DAG based definition of workflows
+ Add `spec.parallelism` field to limit concurrent pod execution at a workflow level
+ Add `template.parallelism` field to limit concurrent pod execution at a template level
+ Add `argo suspend`, `argo resume` to suspend and resume workflows
+ Add `argo resubmit` to resubmit a failed workflow
+ Add `argo retry` to retry a failed workflow with the same name
+ Add `--instanceid` flag to `argo submit` command to submit workflow with controller's specific instance id label
+ Add `--name` and `--generate-name` to override metadata.name and/or metadata.generateName during submission
+ Add `argo logs -w` to support rendering combined workflow logs
+ Experimental support for resubmitting workflows with memoized steps
+ Improved parameters and output validation
+ UI migrated to React
+ Workflow details page redesigned: added DAG view support, added workflow timeline tab.
+ Workflow details page enhancements: added sidecar containers details; workflow exist handler is available on DAG diagram and timeline view.
+ Support for pod tolerations (@discordianfish)
+ Make `workflow.namespace` available as a global variable (@vreon)
+ Support for exported global output parameters and artifacts
+ Trim a trailing newline from path-based output parameters
+ Add ability to reference global parameters in spec level fields
+ Make {{pod.name}} available as a parameter in pod templates
+ Argo CLI shell completion support (@mthx)
+ Add ability to pass pod annotations and labels at the template levels (@wookasz)
+ Add ability to use IAM role from EC2 instance for AWS S3 credentials (@wookasz)
* Abstract the container runtime as an interface to support mocking and future runtimes
* Documentation and examples fixes (@IronPan, @dmonakhov, @bodepd, @mthx, @javierbq, @sebdoido)
* Rewrite the installer
* install & uninstall commands use --namespace flag (@Fitzse)
* Trim spaces from aws keys (@bodepd)
* Update base image to debian 9.4 (from 9.1) (@mthx)
- Global parameters were not referenceable from artifact arguments
- spec.arguments are optionally supplied during linting
- Fix for CLI not rendering edges correctly for nested workflows
- Fix template.parallelism limiting parallelism of entire workflow
- Fix artifact saving to artifactory (@dougsc)
- Use socket type for hostPath to mount docker.sock (@DSchmidtDev)
- Fix rbac resource versions in install (@dvavili)
- Fix input parameters on a steps template prevent daemon pods from terminating (@adampearse)
- Fix locating outbound nodes for skipped node (issue #825)
- Avoid `println` which outputs to stderr (@mthx)
- Fix issue where daemoned steps were not terminated properly in DAG templates

## 2.1.0-beta2 (2018-03-29)

### Changelog since 2.1.0-beta1
- Fix `argo install` does not install argo ui deployment

## 2.1.0-beta1 (2018-03-29)

### Changelog since 2.1.0-alpha1
+ Support for exported global output parameters and artifacts
+ Introduce `argo retry` to retry a failed workflow with the same name
+ Trim a trailing newline from path-based output parameters
+ Add ability to reference global parameters in spec level fields
+ Make {{pod.name}} available as a parameter in pod templates
+ Argo CLI shell completion support (@mthx)
+ Support rendering combined workflow logs using `argo logs -w`
+ Add ability to pass pod annotations and labels at the template levels (@wookasz)
+ Add ability to use IAM role from EC2 instance for AWS S3 credentials (@wookasz)
* Rewrite the installer
* Abstract the container runtime as an interface to support mocking and future runtimes
* Documentation and examples fixes (@IronPan, @dmonakhov)
- Global parameters were not referenceable from artifact arguments
- spec.arguments are optionally supplied during linting
- Fix for CLI not rendering edges correctly for nested workflows
- Fix template.parallelism limiting parallelism of entire workflow
- Fix artifact saving to artifactory (@dougsc)
- Use socket type for hostPath to mount docker.sock (@DSchmidtDev)

## 2.1.0-alpha1 (2018-02-21)

### Changelog since 2.0
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
