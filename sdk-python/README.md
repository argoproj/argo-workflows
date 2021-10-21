# Argo Workflows Python Client

Python client for Argo Workflows

Argo Version: 2.12.2

## Installation

```bash
pip install argo-workflows
```

## Examples

A quick start example with one of the example workflow
```python
import yaml
import requests
from argo.workflows.client import (ApiClient,
                                   WorkflowServiceApi,
                                   Configuration,
                                   V1alpha1WorkflowCreateRequest)

# assume we ran `kubectl -n argo port-forward deployment/argo-server 2746:2746`

config = Configuration(host="http://localhost:2746")
client = ApiClient(configuration=config)
service = WorkflowServiceApi(api_client=client)
WORKFLOW = 'https://raw.githubusercontent.com/argoproj/argo/v2.12.2/examples/dag-diamond-steps.yaml'

resp = requests.get(WORKFLOW)
manifest: dict = yaml.safe_load(resp.text)

service.create_workflow('argo', V1alpha1WorkflowCreateRequest(workflow=manifest))
```

## Documentation for API Endpoints

All URIs are relative to *http://localhost*

Class | Method | HTTP request | Description
------------ | ------------- | ------------- | -------------
*ArchivedWorkflowServiceApi* | [**deleteArchivedWorkflow**](docs/ArchivedWorkflowServiceApi.md#deleteArchivedWorkflow) | **DELETE** /api/v1/archived-workflows/{uid} |
*ArchivedWorkflowServiceApi* | [**getArchivedWorkflow**](docs/ArchivedWorkflowServiceApi.md#getArchivedWorkflow) | **GET** /api/v1/archived-workflows/{uid} |
*ArchivedWorkflowServiceApi* | [**listArchivedWorkflows**](docs/ArchivedWorkflowServiceApi.md#listArchivedWorkflows) | **GET** /api/v1/archived-workflows |
*ClusterWorkflowTemplateServiceApi* | [**createClusterWorkflowTemplate**](docs/ClusterWorkflowTemplateServiceApi.md#createClusterWorkflowTemplate) | **POST** /api/v1/cluster-workflow-templates |
*ClusterWorkflowTemplateServiceApi* | [**deleteClusterWorkflowTemplate**](docs/ClusterWorkflowTemplateServiceApi.md#deleteClusterWorkflowTemplate) | **DELETE** /api/v1/cluster-workflow-templates/{name} |
*ClusterWorkflowTemplateServiceApi* | [**getClusterWorkflowTemplate**](docs/ClusterWorkflowTemplateServiceApi.md#getClusterWorkflowTemplate) | **GET** /api/v1/cluster-workflow-templates/{name} |
*ClusterWorkflowTemplateServiceApi* | [**lintClusterWorkflowTemplate**](docs/ClusterWorkflowTemplateServiceApi.md#lintClusterWorkflowTemplate) | **POST** /api/v1/cluster-workflow-templates/lint |
*ClusterWorkflowTemplateServiceApi* | [**listClusterWorkflowTemplates**](docs/ClusterWorkflowTemplateServiceApi.md#listClusterWorkflowTemplates) | **GET** /api/v1/cluster-workflow-templates |
*ClusterWorkflowTemplateServiceApi* | [**updateClusterWorkflowTemplate**](docs/ClusterWorkflowTemplateServiceApi.md#updateClusterWorkflowTemplate) | **PUT** /api/v1/cluster-workflow-templates/{name} |
*CronWorkflowServiceApi* | [**createCronWorkflow**](docs/CronWorkflowServiceApi.md#createCronWorkflow) | **POST** /api/v1/cron-workflows/{namespace} |
*CronWorkflowServiceApi* | [**deleteCronWorkflow**](docs/CronWorkflowServiceApi.md#deleteCronWorkflow) | **DELETE** /api/v1/cron-workflows/{namespace}/{name} |
*CronWorkflowServiceApi* | [**getCronWorkflow**](docs/CronWorkflowServiceApi.md#getCronWorkflow) | **GET** /api/v1/cron-workflows/{namespace}/{name} |
*CronWorkflowServiceApi* | [**lintCronWorkflow**](docs/CronWorkflowServiceApi.md#lintCronWorkflow) | **POST** /api/v1/cron-workflows/{namespace}/lint |
*CronWorkflowServiceApi* | [**listCronWorkflows**](docs/CronWorkflowServiceApi.md#listCronWorkflows) | **GET** /api/v1/cron-workflows/{namespace} |
*CronWorkflowServiceApi* | [**updateCronWorkflow**](docs/CronWorkflowServiceApi.md#updateCronWorkflow) | **PUT** /api/v1/cron-workflows/{namespace}/{name} |
*EventServiceApi* | [**receiveEvent**](docs/EventServiceApi.md#receiveEvent) | **POST** /api/v1/events/{namespace}/{discriminator} |
*InfoServiceApi* | [**getInfo**](docs/InfoServiceApi.md#getInfo) | **GET** /api/v1/info |
*InfoServiceApi* | [**getUserInfo**](docs/InfoServiceApi.md#getUserInfo) | **GET** /api/v1/userinfo |
*InfoServiceApi* | [**getVersion**](docs/InfoServiceApi.md#getVersion) | **GET** /api/v1/version |
*WorkflowServiceApi* | [**createWorkflow**](docs/WorkflowServiceApi.md#createWorkflow) | **POST** /api/v1/workflows/{namespace} |
*WorkflowServiceApi* | [**deleteWorkflow**](docs/WorkflowServiceApi.md#deleteWorkflow) | **DELETE** /api/v1/workflows/{namespace}/{name} |
*WorkflowServiceApi* | [**getWorkflow**](docs/WorkflowServiceApi.md#getWorkflow) | **GET** /api/v1/workflows/{namespace}/{name} |
*WorkflowServiceApi* | [**lintWorkflow**](docs/WorkflowServiceApi.md#lintWorkflow) | **POST** /api/v1/workflows/{namespace}/lint |
*WorkflowServiceApi* | [**listWorkflows**](docs/WorkflowServiceApi.md#listWorkflows) | **GET** /api/v1/workflows/{namespace} |
*WorkflowServiceApi* | [**podLogs**](docs/WorkflowServiceApi.md#podLogs) | **GET** /api/v1/workflows/{namespace}/{name}/{podName}/log |
*WorkflowServiceApi* | [**resubmitWorkflow**](docs/WorkflowServiceApi.md#resubmitWorkflow) | **PUT** /api/v1/workflows/{namespace}/{name}/resubmit |
*WorkflowServiceApi* | [**resumeWorkflow**](docs/WorkflowServiceApi.md#resumeWorkflow) | **PUT** /api/v1/workflows/{namespace}/{name}/resume |
*WorkflowServiceApi* | [**retryWorkflow**](docs/WorkflowServiceApi.md#retryWorkflow) | **PUT** /api/v1/workflows/{namespace}/{name}/retry |
*WorkflowServiceApi* | [**setWorkflow**](docs/WorkflowServiceApi.md#setWorkflow) | **PUT** /api/v1/workflows/{namespace}/{name}/set |
*WorkflowServiceApi* | [**stopWorkflow**](docs/WorkflowServiceApi.md#stopWorkflow) | **PUT** /api/v1/workflows/{namespace}/{name}/stop |
*WorkflowServiceApi* | [**submitWorkflow**](docs/WorkflowServiceApi.md#submitWorkflow) | **POST** /api/v1/workflows/{namespace}/submit |
*WorkflowServiceApi* | [**suspendWorkflow**](docs/WorkflowServiceApi.md#suspendWorkflow) | **PUT** /api/v1/workflows/{namespace}/{name}/suspend |
*WorkflowServiceApi* | [**terminateWorkflow**](docs/WorkflowServiceApi.md#terminateWorkflow) | **PUT** /api/v1/workflows/{namespace}/{name}/terminate |
*WorkflowServiceApi* | [**watchEvents**](docs/WorkflowServiceApi.md#watchEvents) | **GET** /api/v1/stream/events/{namespace} |
*WorkflowServiceApi* | [**watchWorkflows**](docs/WorkflowServiceApi.md#watchWorkflows) | **GET** /api/v1/workflow-events/{namespace} |
*WorkflowTemplateServiceApi* | [**createWorkflowTemplate**](docs/WorkflowTemplateServiceApi.md#createWorkflowTemplate) | **POST** /api/v1/workflow-templates/{namespace} |
*WorkflowTemplateServiceApi* | [**deleteWorkflowTemplate**](docs/WorkflowTemplateServiceApi.md#deleteWorkflowTemplate) | **DELETE** /api/v1/workflow-templates/{namespace}/{name} |
*WorkflowTemplateServiceApi* | [**getWorkflowTemplate**](docs/WorkflowTemplateServiceApi.md#getWorkflowTemplate) | **GET** /api/v1/workflow-templates/{namespace}/{name} |
*WorkflowTemplateServiceApi* | [**lintWorkflowTemplate**](docs/WorkflowTemplateServiceApi.md#lintWorkflowTemplate) | **POST** /api/v1/workflow-templates/{namespace}/lint |
*WorkflowTemplateServiceApi* | [**listWorkflowTemplates**](docs/WorkflowTemplateServiceApi.md#listWorkflowTemplates) | **GET** /api/v1/workflow-templates/{namespace} |
*WorkflowTemplateServiceApi* | [**updateWorkflowTemplate**](docs/WorkflowTemplateServiceApi.md#updateWorkflowTemplate) | **PUT** /api/v1/workflow-templates/{namespace}/{name} |


## Documentation for Models
 - [V1AzureDiskVolumeSource](docs/V1AzureDiskVolumeSource.md)
 - [V1alpha1Arguments](docs/V1alpha1Arguments.md)
 - [V1alpha1CronWorkflowSpec](docs/V1alpha1CronWorkflowSpec.md)
 - [V1alpha1WorkflowSetRequest](docs/V1alpha1WorkflowSetRequest.md)
 - [V1ConfigMapProjection](docs/V1ConfigMapProjection.md)
 - [V1Volume](docs/V1Volume.md)
 - [V1alpha1WorkflowLintRequest](docs/V1alpha1WorkflowLintRequest.md)
 - [V1alpha1MetricLabel](docs/V1alpha1MetricLabel.md)
 - [V1PortworxVolumeSource](docs/V1PortworxVolumeSource.md)
 - [V1alpha1Histogram](docs/V1alpha1Histogram.md)
 - [V1alpha1ArtifactoryArtifact](docs/V1alpha1ArtifactoryArtifact.md)
 - [V1alpha1ClusterWorkflowTemplateCreateRequest](docs/V1alpha1ClusterWorkflowTemplateCreateRequest.md)
 - [V1EnvVar](docs/V1EnvVar.md)
 - [V1SecretKeySelector](docs/V1SecretKeySelector.md)
 - [V1StorageOSVolumeSource](docs/V1StorageOSVolumeSource.md)
 - [V1StatusDetails](docs/V1StatusDetails.md)
 - [V1ObjectMeta](docs/V1ObjectMeta.md)
 - [V1alpha1WorkflowResubmitRequest](docs/V1alpha1WorkflowResubmitRequest.md)
 - [V1alpha1ContinueOn](docs/V1alpha1ContinueOn.md)
 - [V1alpha1Sequence](docs/V1alpha1Sequence.md)
 - [V1alpha1WorkflowResumeRequest](docs/V1alpha1WorkflowResumeRequest.md)
 - [V1PodAffinityTerm](docs/V1PodAffinityTerm.md)
 - [V1alpha1ValueFrom](docs/V1alpha1ValueFrom.md)
 - [V1ServiceAccountTokenProjection](docs/V1ServiceAccountTokenProjection.md)
 - [V1TypedLocalObjectReference](docs/V1TypedLocalObjectReference.md)
 - [V1alpha1LogEntry](docs/V1alpha1LogEntry.md)
 - [V1ISCSIVolumeSource](docs/V1ISCSIVolumeSource.md)
 - [V1EventSeries](docs/V1EventSeries.md)
 - [V1alpha1WorkflowTemplate](docs/V1alpha1WorkflowTemplate.md)
 - [V1TCPSocketAction](docs/V1TCPSocketAction.md)
 - [V1Initializer](docs/V1Initializer.md)
 - [V1NodeSelectorRequirement](docs/V1NodeSelectorRequirement.md)
 - [V1CreateOptions](docs/V1CreateOptions.md)
 - [V1ConfigMapEnvSource](docs/V1ConfigMapEnvSource.md)
 - [V1VolumeDevice](docs/V1VolumeDevice.md)
 - [V1VolumeProjection](docs/V1VolumeProjection.md)
 - [V1DownwardAPIVolumeFile](docs/V1DownwardAPIVolumeFile.md)
 - [V1alpha1WorkflowStep](docs/V1alpha1WorkflowStep.md)
 - [V1PodDNSConfigOption](docs/V1PodDNSConfigOption.md)
 - [V1PodDNSConfig](docs/V1PodDNSConfig.md)
 - [V1alpha1UserContainer](docs/V1alpha1UserContainer.md)
 - [V1StatusCause](docs/V1StatusCause.md)
 - [V1Capabilities](docs/V1Capabilities.md)
 - [V1RBDVolumeSource](docs/V1RBDVolumeSource.md)
 - [V1HTTPGetAction](docs/V1HTTPGetAction.md)
 - [V1OwnerReference](docs/V1OwnerReference.md)
 - [V1alpha1ResourceTemplate](docs/V1alpha1ResourceTemplate.md)
 - [V1Initializers](docs/V1Initializers.md)
 - [V1alpha1WorkflowTemplateList](docs/V1alpha1WorkflowTemplateList.md)
 - [V1alpha1Inputs](docs/V1alpha1Inputs.md)
 - [V1alpha1DAGTemplate](docs/V1alpha1DAGTemplate.md)
 - [V1ExecAction](docs/V1ExecAction.md)
 - [V1alpha1CronWorkflowList](docs/V1alpha1CronWorkflowList.md)
 - [V1alpha1WorkflowCreateRequest](docs/V1alpha1WorkflowCreateRequest.md)
 - [V1alpha1HDFSArtifact](docs/V1alpha1HDFSArtifact.md)
 - [V1KeyToPath](docs/V1KeyToPath.md)
 - [V1Event](docs/V1Event.md)
 - [V1alpha1ClusterWorkflowTemplateList](docs/V1alpha1ClusterWorkflowTemplateList.md)
 - [V1SELinuxOptions](docs/V1SELinuxOptions.md)
 - [V1HostPathVolumeSource](docs/V1HostPathVolumeSource.md)
 - [V1Sysctl](docs/V1Sysctl.md)
 - [V1ConfigMapKeySelector](docs/V1ConfigMapKeySelector.md)
 - [V1EmptyDirVolumeSource](docs/V1EmptyDirVolumeSource.md)
 - [V1GlusterfsVolumeSource](docs/V1GlusterfsVolumeSource.md)
 - [V1ContainerPort](docs/V1ContainerPort.md)
 - [V1AWSElasticBlockStoreVolumeSource](docs/V1AWSElasticBlockStoreVolumeSource.md)
 - [V1PreferredSchedulingTerm](docs/V1PreferredSchedulingTerm.md)
 - [V1ObjectReference](docs/V1ObjectReference.md)
 - [V1alpha1ScriptTemplate](docs/V1alpha1ScriptTemplate.md)
 - [V1PodAntiAffinity](docs/V1PodAntiAffinity.md)
 - [V1NodeSelectorTerm](docs/V1NodeSelectorTerm.md)
 - [V1EnvFromSource](docs/V1EnvFromSource.md)
 - [V1LabelSelectorRequirement](docs/V1LabelSelectorRequirement.md)
 - [V1ManagedFieldsEntry](docs/V1ManagedFieldsEntry.md)
 - [V1alpha1WorkflowEventBindingSpec](docs/V1alpha1WorkflowEventBindingSpec.md)
 - [V1alpha1MutexHolding](docs/V1alpha1MutexHolding.md)
 - [V1alpha1S3Artifact](docs/V1alpha1S3Artifact.md)
 - [V1ConfigMapVolumeSource](docs/V1ConfigMapVolumeSource.md)
 - [V1alpha1CronWorkflow](docs/V1alpha1CronWorkflow.md)
 - [V1alpha1SubmitOpts](docs/V1alpha1SubmitOpts.md)
 - [V1PodAffinity](docs/V1PodAffinity.md)
 - [V1alpha1Mutex](docs/V1alpha1Mutex.md)
 - [V1Toleration](docs/V1Toleration.md)
 - [V1alpha1Outputs](docs/V1alpha1Outputs.md)
 - [V1CinderVolumeSource](docs/V1CinderVolumeSource.md)
 - [V1SecretProjection](docs/V1SecretProjection.md)
 - [V1SecurityContext](docs/V1SecurityContext.md)
 - [V1alpha1WorkflowRetryRequest](docs/V1alpha1WorkflowRetryRequest.md)
 - [V1GitRepoVolumeSource](docs/V1GitRepoVolumeSource.md)
 - [V1alpha1Metadata](docs/V1alpha1Metadata.md)
 - [V1Status](docs/V1Status.md)
 - [V1alpha1NodeSynchronizationStatus](docs/V1alpha1NodeSynchronizationStatus.md)
 - [V1ScaleIOVolumeSource](docs/V1ScaleIOVolumeSource.md)
 - [V1PersistentVolumeClaim](docs/V1PersistentVolumeClaim.md)
 - [V1alpha1Artifact](docs/V1alpha1Artifact.md)
 - [V1alpha1NodeStatus](docs/V1alpha1NodeStatus.md)
 - [V1FlexVolumeSource](docs/V1FlexVolumeSource.md)
 - [V1alpha1OSSArtifact](docs/V1alpha1OSSArtifact.md)
 - [V1alpha1WorkflowTemplateRef](docs/V1alpha1WorkflowTemplateRef.md)
 - [V1WeightedPodAffinityTerm](docs/V1WeightedPodAffinityTerm.md)
 - [V1alpha1ClusterWorkflowTemplateUpdateRequest](docs/V1alpha1ClusterWorkflowTemplateUpdateRequest.md)
 - [V1alpha1Counter](docs/V1alpha1Counter.md)
 - [V1alpha1SemaphoreRef](docs/V1alpha1SemaphoreRef.md)
 - [V1alpha1ArchiveStrategy](docs/V1alpha1ArchiveStrategy.md)
 - [V1alpha1WorkflowTemplateSpec](docs/V1alpha1WorkflowTemplateSpec.md)
 - [V1EnvVarSource](docs/V1EnvVarSource.md)
 - [V1alpha1Synchronization](docs/V1alpha1Synchronization.md)
 - [V1alpha1Metrics](docs/V1alpha1Metrics.md)
 - [V1AzureFileVolumeSource](docs/V1AzureFileVolumeSource.md)
 - [V1alpha1Event](docs/V1alpha1Event.md)
 - [V1alpha1Memoize](docs/V1alpha1Memoize.md)
 - [V1alpha1ClusterWorkflowTemplateLintRequest](docs/V1alpha1ClusterWorkflowTemplateLintRequest.md)
 - [V1alpha1WorkflowList](docs/V1alpha1WorkflowList.md)
 - [V1alpha1Gauge](docs/V1alpha1Gauge.md)
 - [V1alpha1SemaphoreStatus](docs/V1alpha1SemaphoreStatus.md)
 - [V1GCEPersistentDiskVolumeSource](docs/V1GCEPersistentDiskVolumeSource.md)
 - [V1alpha1RawArtifact](docs/V1alpha1RawArtifact.md)
 - [V1alpha1ArtifactRepositoryRef](docs/V1alpha1ArtifactRepositoryRef.md)
 - [V1ResourceFieldSelector](docs/V1ResourceFieldSelector.md)
 - [V1PersistentVolumeClaimSpec](docs/V1PersistentVolumeClaimSpec.md)
 - [V1alpha1Parameter](docs/V1alpha1Parameter.md)
 - [V1PersistentVolumeClaimCondition](docs/V1PersistentVolumeClaimCondition.md)
 - [V1Lifecycle](docs/V1Lifecycle.md)
 - [V1alpha1PodGC](docs/V1alpha1PodGC.md)
 - [V1alpha1LintCronWorkflowRequest](docs/V1alpha1LintCronWorkflowRequest.md)
 - [V1DownwardAPIVolumeSource](docs/V1DownwardAPIVolumeSource.md)
 - [V1alpha1Workflow](docs/V1alpha1Workflow.md)
 - [V1VolumeMount](docs/V1VolumeMount.md)
 - [V1EventSource](docs/V1EventSource.md)
 - [V1LabelSelector](docs/V1LabelSelector.md)
 - [V1VsphereVirtualDiskVolumeSource](docs/V1VsphereVirtualDiskVolumeSource.md)
 - [V1alpha1TemplateRef](docs/V1alpha1TemplateRef.md)
 - [V1alpha1CreateCronWorkflowRequest](docs/V1alpha1CreateCronWorkflowRequest.md)
 - [V1alpha1GitArtifact](docs/V1alpha1GitArtifact.md)
 - [V1ProjectedVolumeSource](docs/V1ProjectedVolumeSource.md)
 - [V1SecretEnvSource](docs/V1SecretEnvSource.md)
 - [V1PhotonPersistentDiskVolumeSource](docs/V1PhotonPersistentDiskVolumeSource.md)
 - [V1alpha1WorkflowWatchEvent](docs/V1alpha1WorkflowWatchEvent.md)
 - [V1alpha1DAGTask](docs/V1alpha1DAGTask.md)
 - [V1alpha1CronWorkflowStatus](docs/V1alpha1CronWorkflowStatus.md)
 - [V1alpha1WorkflowSuspendRequest](docs/V1alpha1WorkflowSuspendRequest.md)
 - [V1CSIVolumeSource](docs/V1CSIVolumeSource.md)
 - [V1alpha1Submit](docs/V1alpha1Submit.md)
 - [V1alpha1GCSArtifact](docs/V1alpha1GCSArtifact.md)
 - [V1alpha1HTTPArtifact](docs/V1alpha1HTTPArtifact.md)
 - [V1alpha1Version](docs/V1alpha1Version.md)
 - [V1Container](docs/V1Container.md)
 - [V1alpha1ClusterWorkflowTemplate](docs/V1alpha1ClusterWorkflowTemplate.md)
 - [V1alpha1WorkflowTemplateUpdateRequest](docs/V1alpha1WorkflowTemplateUpdateRequest.md)
 - [V1FCVolumeSource](docs/V1FCVolumeSource.md)
 - [V1Affinity](docs/V1Affinity.md)
 - [V1alpha1SemaphoreHolding](docs/V1alpha1SemaphoreHolding.md)
 - [V1alpha1TTLStrategy](docs/V1alpha1TTLStrategy.md)
 - [V1alpha1WorkflowSubmitRequest](docs/V1alpha1WorkflowSubmitRequest.md)
 - [V1Handler](docs/V1Handler.md)
 - [V1HTTPHeader](docs/V1HTTPHeader.md)
 - [V1ListMeta](docs/V1ListMeta.md)
 - [V1alpha1SuspendTemplate](docs/V1alpha1SuspendTemplate.md)
 - [V1alpha1WorkflowStatus](docs/V1alpha1WorkflowStatus.md)
 - [V1alpha1WorkflowTemplateCreateRequest](docs/V1alpha1WorkflowTemplateCreateRequest.md)
 - [V1PersistentVolumeClaimStatus](docs/V1PersistentVolumeClaimStatus.md)
 - [V1alpha1Link](docs/V1alpha1Link.md)
 - [V1alpha1WorkflowSpec](docs/V1alpha1WorkflowSpec.md)
 - [V1PersistentVolumeClaimVolumeSource](docs/V1PersistentVolumeClaimVolumeSource.md)
 - [V1ResourceRequirements](docs/V1ResourceRequirements.md)
 - [V1FlockerVolumeSource](docs/V1FlockerVolumeSource.md)
 - [V1NodeSelector](docs/V1NodeSelector.md)
 - [V1alpha1ArtifactLocation](docs/V1alpha1ArtifactLocation.md)
 - [V1alpha1InfoResponse](docs/V1alpha1InfoResponse.md)
 - [V1alpha1TarStrategy](docs/V1alpha1TarStrategy.md)
 - [V1alpha1WorkflowStopRequest](docs/V1alpha1WorkflowStopRequest.md)
 - [V1ObjectFieldSelector](docs/V1ObjectFieldSelector.md)
 - [V1NFSVolumeSource](docs/V1NFSVolumeSource.md)
 - [V1alpha1Condition](docs/V1alpha1Condition.md)
 - [V1CephFSVolumeSource](docs/V1CephFSVolumeSource.md)
 - [V1alpha1ExecutorConfig](docs/V1alpha1ExecutorConfig.md)
 - [V1alpha1Prometheus](docs/V1alpha1Prometheus.md)
 - [V1alpha1GetUserInfoResponse](docs/V1alpha1GetUserInfoResponse.md)
 - [V1alpha1Backoff](docs/V1alpha1Backoff.md)
 - [V1alpha1WorkflowEventBinding](docs/V1alpha1WorkflowEventBinding.md)
 - [V1alpha1Template](docs/V1alpha1Template.md)
 - [V1QuobyteVolumeSource](docs/V1QuobyteVolumeSource.md)
 - [V1PodSecurityContext](docs/V1PodSecurityContext.md)
 - [V1WindowsSecurityContextOptions](docs/V1WindowsSecurityContextOptions.md)
 - [V1LocalObjectReference](docs/V1LocalObjectReference.md)
 - [V1NodeAffinity](docs/V1NodeAffinity.md)
 - [V1HostAlias](docs/V1HostAlias.md)
 - [V1alpha1MemoizationStatus](docs/V1alpha1MemoizationStatus.md)
 - [V1alpha1SynchronizationStatus](docs/V1alpha1SynchronizationStatus.md)
 - [V1alpha1UpdateCronWorkflowRequest](docs/V1alpha1UpdateCronWorkflowRequest.md)
 - [V1SecretVolumeSource](docs/V1SecretVolumeSource.md)
 - [V1Probe](docs/V1Probe.md)
 - [V1alpha1Cache](docs/V1alpha1Cache.md)
 - [V1alpha1RetryStrategy](docs/V1alpha1RetryStrategy.md)
 - [V1alpha1WorkflowTemplateLintRequest](docs/V1alpha1WorkflowTemplateLintRequest.md)
 - [V1alpha1WorkflowTerminateRequest](docs/V1alpha1WorkflowTerminateRequest.md)
 - [V1alpha1MutexStatus](docs/V1alpha1MutexStatus.md)
 - [V1DownwardAPIProjection](docs/V1DownwardAPIProjection.md)


## Code generation

The generated SDK will correspond to the argo version specified in the [ARGO_VERSION](./ARGO_VERSION) file.

If you wish to generate code yourself, you can do so by reproducing the build environment (image): `make builder_image`, then running `make builder_make` to generate the client.
