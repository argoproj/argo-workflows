# Changelog

## 2.4.0 (2019-10-02)

### New Features
+ WorkflowTemplate CRD (#1312) (@dtaniwaki)
+ Centralized Longterm workflow persistence storage (#1344) (@sarabala1979)
+ Support PodSecurityContext (#1463) (@dtaniwaki)
+ Conditionally annotate outputs of script template only when consumed #1359 (#1462) (@sarabala1979)
+ Support for WorkflowSpec.ArtifactRepositoryRef (#1350) (@Ark-kun)
+ Ability to configure hostPath mount for `/var/run/docker.sock` (#1419) (@sarabala1979)
+ Support hostAliases in WorkflowSpec #1265 (#1365) (@sarabala1979)
+ Template level service account (#1354) (@sarabala1979)
+ Provide failFast flag, allow a DAG to run all branches of the DAG (either success or failure) (#1443) (@xianlubird)
+ Add paging function for list command (#1420) (@xianlubird)
+ Allow overriding workflow labels in 'argo submit' (#1475) (@mark9white)
+ Support git shallow clones and additional ref fetches (#1521) (@marxarelli)
+ Add --dry-run option to `argo submit` (#1506) (@AnesBenmerzoug)
+ Support AutomountServiceAccountToken and executor specific service account(#1480)
+ Support ability to assume IAM roles in S3 Artifacts  (#1587)

### Refactoring & Improvements:
* Allow output parameters with .value, not only .valueFrom (#1336) (@Ark-kun)
* Exposed workflow priority as a variable (#1476) (@mark9white)
+ Expose all input parameters to template as JSON (#1488) (@mark9white)
* Argo CLI should show warning if there is no workflow definition in file #1486 (@sarabala1979)
* `argo wait` and `argo submit --wait` should exit 1 if workflow fails (#1467) (@sarabala1979)
* Improve bash completion (#1437) (@edwinpjacques)
* Add --no-color flag to logs (#1479) (@dtaniwaki)
* mention sidecar in failure message for sidecar containers (#1430) (@tralexa)
* changing temp directory for output artifacts from root to tmp (#1458) (@alexcapras)
* Format sources and order imports with the help of goimports (#1504) (@muesli)
* Documentation (@ofaz, @bvwells, @pbrit, @shimmerjs, @ianCambrio, @jqueguiner, @ntwrkguru, @thundergolfer, @delwaterman, @Ziyang2go, @Aisuko, @mark9white, @mostaphaRoudsari, @commodus-sebastien, @thundergolfer, @xianlubird, @mostaphaRoudsari, @bpmericle)

### Bug Fixes
- Fix: Support the List within List type in withParam #1471 (#1473) (@sarabala1979)
- Fix #1366 unpredictable global artifact behavior (#1461) (@schrodit)
- Fix a compiler error (#1500) (@ian-howell)
- Fixed: failed to save outputs: verify serviceaccount default:default has necessary privileges (#1362) (@sarabala1979)
- Fixed: withParam parsing of JSON/YAML lists #1389 (#1397) (@ian-howell)
- Fixed: persistentvolumeclaims already exists #1130 (#1363) (@sarabala1979)
- PNS executor intermitently failed to capture entire log of script templates (#1406) (@jessesuen)
- Fix argo logs empty content when workflow run in virtual kubelet env (#1201) (@xianlubird)
- Terminate all containers within pod after main container completes (#1423) (@SeriousSem)
- Initialize the wfClientset before using it (#1548)
- Fix issue saving outputs which overlap paths with inputs (#1567)


## 2.3.0 (2019-05-20)

### Notes about upgrading from v2.2

* Artifact repository secrets are passed to the wait sidecar using volumeMounts instead of the
  previous behavior of performing K8s API calls performed by the executor. This is much more secure
  since it removes privileges of the workflow pod to no longer require secret access. However, as a
  consequence, workflow pods which reference a secret that does not exist, will now indefinitely
  stay in a Pending state, as opposed to the previous behavior of failing during runtime.

### Deprecation Notice
The workflow-controller-configmap introduces a new config field, `executor`, which is a container
spec and provides controls over the executor sidecar container (i.e. `init`/`wait`). The fields
`executorImage`, `executorResources`, and `executorImagePullPolicy` are deprecated and will be
removed in a future release.

### New Features:
+ Support for PNS (Process Namespace Sharing) executor (#1214)
+ Support for K8s API based Executor (#1010) (@dtaniwaki)
+ Adds limited support for Kubelet/K8s API artifact collection by mirroring volume mounts to wait sidecar
+ Support HDFS Artifact (#1159) (@dtaniwaki)
+ System level workflow parallelism limits & priorities (#1065)
+ Support larger workflows through node status compression (#1264)
+ Support nested steps workflow parallelism (#1046) (@WeiTang114)
+ Add feature to continue workflow on failed/error steps/tasks (#1205) (@schrodit)
+ Parameter and Argument names should support snake case (#1048) (@bbc88ks)
+ Add support for ppc64le and s390x (#1102) (@chenzhiwei)
+ Install mime-support in argoexec to set proper mime types for S3 artifacts
+ Allow owner reference to be set in submit util (#1120) (@nareshku)
+ add support for hostNetwork & dnsPolicy config (#1161) (@Dreamheart)
+ Add schedulerName to workflow and template spec (#1184) (@houz42)
+ Executor can access the k8s apiserver with a out-of-cluster config file (@houz42)
+ Proxy Priority and PriorityClassName to pods (#1179) (@dtaniwaki)
+ Add the `mergeStrategy` option to resource patching (#1269) (@ian-howell)
+ Add workflow labels and annotations global vars (#1280) (@discordianfish)
+ Support for optional input/output artifacts (#1277)
+ Add dns config support (#1301) (@xianlubird)
+ Added support for artifact path references (#1300) (@Ark-kun)
+ Add support for init containers (#1183) (@dtaniwaki)
+ Secrets should be passed to pods using volumes instead of API calls (#1302)
+ Azure AKS authentication issues #1079 (@gerardaus)
+ Support parameter substitution in the volumes attribute (#1238)

### Refactoring & Improvements:
* Update dependencies to K8s v1.12 and client-go 9.0
* Add namespace explicitly to pod metadata (#1059) (@dvavili)
* Raise not implemented error when artifact saving is unsupported (#1062) (@dtaniwaki)
* Retry logic to s3 load and save function (#1082) (@kshamajain99)
* Remove docker_lib mount volume which is not needed anymore (#1115) (@ywskycn)
* Documentation improvements and fixes (@protochron, @jmcarp, @locona, @kivio, @fischerjulian, @annawinkler, @jdfalko, @groodt, @migggy, @nstott, @adrienjt)
* Validate ArchiveLocation artifacts (#1167) (@dtaniwaki)
* Git cloning via SSH was not verifying host public key (#1261)
* Speed up podReconciliation using parallel goroutine (#1286) (@xianlubird)

### Bug Fixes
- Initialize child node before marking phase. Fixes panic on invalid `When` (#1075) (@jmcarp)
- Submodules are dirty after checkout -- need to update (#1052) (@andreimc)
- Fix output artifact and parameter conflict (#1125) (@Ark-kun)
- Remove container wait timeout from 'argo logs --follow' (#1142)
- Fix panic in ttl controller (#1143)
- Kill daemoned step if workflow consist of single daemoned step (#1144)
- Fix global artifact overwriting in nested workflow (#1086) (@WeiTang114)
- Fix issue where steps with exhausted retires would not complete (#1148)
- Fix metadata for DAG with loops (#1149)
- Replace exponential retry with poll (#1166) (@kzadorozhny)
- Dockerfile: argoexec base image correction (#1213) (@elikatsis)
- Set executor image pull policy for resource template (#1174) (@dtaniwaki)
- fix dag retries (#1221) (@houz42)
- Remove extra quotes around output parameter value (#1232) (@elikatsis)
- Include stderr when retrieving docker logs (#1225) (@shahin)
- Fix the Prometheus address references (#1237) (@spacez320)
- Kubernetes Resource action: patch is not supported (#1245)
- Fake outputs don't notify and task completes successfully (#1247)
- Reduce redundancy pod label action (#1271) (@xianlubird)
- Fix bug with DockerExecutor's CopyFile (#1275)
- Fix for Resource creation where template has same parameter templating (#1283)
- Fixes an issue where daemon steps were not getting terminated properly
- argo submit --wait and argo wait quits while workflow is running (#1347)
- Fix input artifacts with multiple ssh keys (#1338) (@almariah)
- Add when test for character that included `/` (@hideto0710)
- Fix parameter substitution bug (#1345) (@elikatsis)
- Fix missing template local volumes, Handle volumes only used in init containers (#1342)
- Export the methods of `KubernetesClientInterface` (#1294)


## 2.2.1 (2018-10-18)

### Changelog since v2.2.0
+ UI retrieve logs from artifacts location if logs archiving is enabled (issue #1018)
+ Add imagePullPolicy config for executors (@dtaniwaki)
+ Detect and indicate when container was OOMKilled
+ support force namespace isolation in UI
- Workflow executor panic: workflows.argoproj.io/template not found (issue #1033)
- gc-ttl dose not work (issue #1004)
- Resubmission of a terminated workflow creates a new workflow that is already terminated (issue #1011)
- ZIP containing single file cannot be used as an artifact due to errors in init container (issue #984) (@mthx)
- Regression when S3 secret has trailing newline (issue #981)
* Documentation fixes (@gsf, @davidB, @dtaniwaki)

## 2.2.0 (2018-08-30)

### Notes about upgrading from v2.1

* The `argo install` and `argo uninstall` commands have been removed from the CLI. Instead, plain
kubernetes manifests are provided to be installed using `kubectl apply`, or downstreamed into other
tools (e.g. helm chart, ksonnet prototype, kustomize, etc...).
* In 2.1, argo would install into the kube-system namespace by default. The new install instructions
have been updated to install into a different namespace, `argo`. In order to move to the recommended
installation location, you should delete the v2.1 resources from kube-system before applying the
new manifests to the `argo` namespace.

    The following commands migrates the workflow-controller-configmap from the `kube-system` to the
`argo` namespace, and deletes all argo resources from the `kube-system` namespace. Note that this
will delete the argo-ui service, resulting in the LoadBalancer being deleted (if created).

    ```
    kubectl get cm workflow-controller-configmap -o yaml -n kube-system --export | kubectl apply -n argo -f -
    kubectl delete -n kube-system cm workflow-controller-configmap
    kubectl delete -n kube-system deploy workflow-controller argo-ui
    kubectl delete -n kube-system sa argo argo-ui
    kubectl delete -n kube-system svc argo-ui
    ```

* In 2.1, the argoexec sidecar image was configured in the workflow-controller-configmap. This is
now configured using a new `--executor-image` flag in the `workflow-controller` deployment. This is
the preferred way to configure the executor image, since upgrades can now be performed without
changing the workflow-controller configmap. The executorImage setting in the config is deprecated
and may be removed/ignored in a future release.

### Changelog since v2.1
+ Support withItems/withParam and parameter aggregation with DAG templates (issue #801)
+ Add ability to aggregate and reference output parameters expanded by loops (issue #861)
+ Support for sophisticated expressions in `when` conditionals (issue #860)
+ Introduce Pending node state to highlight failures when starting workflow pods (issue #525)
+ Support additional container runtimes through kubelet executor (issue #902) (@JulienBalestra)
+ Introduce archive strategies with ability to disable tar.gz archiving (issue #784)
+ Introduce `keyFormat` workflow config to enable flexibility in archive location path (issue #953)
+ Introduce `argo watch` command to watch live workflows from terminal (issue #969)
+ Add ability to archive container logs to the artifact repository (issue #454)
+ Support for workflow level timeouts (issue #848)
+ Introduce `argo terminate` to terminate a workflow without deleting it (issue #527)
+ Introduce `withSequence` to iterate a range of numbers in a loop (issue #945)
+ Github login using go-git, with support for ssh keys (issue #793) (@andreimc)
+ Add TTLSecondsAfterFinished field and controller to garbage collect completed workflows (issue #911)
+ Add `argo delete --older` flag to delete completed workflows older than a duration
+ Support referencing of global workflow artifacts (issue #900)
+ Support submission of workflows from json files (issue #926)
+ Support submission of workflows from stdin (issue #926)
+ Prometheus metrics and telemetry (issue #896) (@bbc88ks)
+ Detect and fail upon unknown fields during argo submit & lint (issue #892)
+ Allow scaling of workflow and pod workers via controller CLI flags (issue #962)
+ Allow supplying of parameters from a file during `argo submit` (issue #796) (@vosmith)
+ [UI] UI support/spinning clock for pending pods (@EdanSneh)
* Remove installer/uninstaller (issue #928)
* Update golang compiler to v1.10.3
* Update k8s dependencies to v1.10 and client-go to v7.0
* Update argo-cluster-role to work with OpenShift
- Fix issue where retryStrategy with DAGs fails, even if the step passes after retries (issue #885)
- Fix issue where sidecars and daemons were not reliably killed (issue #879)
- Redundant verifyResolvedVariables check in controller precluded the ability to use {{ }} in other circumstances
- Fix issue where retryStrategy with DAGs fails, even if the step passes after retries (issue #885)
- Fix outbound node metadata with steps templates causing incorrect edges to be rendered in UI
- Fix outbound node metadata with retry nodes causing disconnected nodes to be rendered in UI (issue #880)
- Error workflows which hit k8s/etcd 1M resource size limit (issue #913)
- [UI] Fixed 'X' hiding under page (@EdanSneh)
- [UI] Beautified resource template. Yaml will now indent 2 spaces instead of one space

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
