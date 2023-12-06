import * as kubernetes from 'argo-ui/src/models/kubernetes';

export const labels = {
    clusterWorkflowTemplate: 'workflows.argoproj.io/cluster-workflow-template',
    completed: 'workflows.argoproj.io/completed',
    creator: 'workflows.argoproj.io/creator',
    creatorEmail: 'workflows.argoproj.io/creator-email',
    creatorPreferredUsername: 'workflows.argoproj.io/creator-preferred-username',
    cronWorkflow: 'workflows.argoproj.io/cron-workflow',
    workflowTemplate: 'workflows.argoproj.io/workflow-template'
};

/**
 * Arguments to a template
 */
export interface Arguments {
    /**
     * Artifacts is the list of artifacts to pass to the template or workflow
     */
    artifacts?: Artifact[];
    /**
     * Parameters is the list of parameters to pass to the template or workflow
     */
    parameters?: Parameter[];
}

export interface AzureArtifactRepository {
    endpoint: string;
    container: string;
    blob: string;
}
export interface GCSArtifactRepository {
    endpoint: string;
    bucket: string;
    key: string;
}
export interface S3ArtifactRepository {
    endpoint: string;
    bucket: string;
    key: string;
}

export interface OSSArtifactRepository {
    endpoint: string;
    bucket: string;
    key: string;
}

export interface GitArtifactRepository {
    repo?: string;
    endpoint?: string;
    bucket?: string;
}

export interface HTTPArtifactRepository {
    url: string;
}

export interface RawArtifactRepository {
    data: string;
}

export interface ArtifactRepository {
    gcs?: GCSArtifactRepository;
    s3?: S3ArtifactRepository;
    oss?: OSSArtifactRepository;
    azure?: AzureArtifactRepository;
    git?: GitArtifactRepository;
    http?: HTTPArtifactRepository;
    raw?: RawArtifactRepository;
}
export interface ArtifactRepositoryRefStatus {
    artifactRepository: ArtifactRepository;
}

export interface AzureArtifact {
    endpoint?: string;
    container?: string;
    blob?: string;
}

export interface GCSArtifact {
    endpoint?: string;
    bucket?: string;
    key: string;
}

export interface GitArtifact {
    repo: string;
    branch?: string;
    revision?: string;
}

export interface HTTPArtifact {
    url: string;
}

export interface OSSArtifact {
    endpoint?: string;
    bucket?: string;
    key: string;
}

export interface RawArtifact {
    data: string;
}

export interface S3Artifact {
    endpoint?: string;
    bucket?: string;
    key: string;
}

/**
 * Artifact indicates an artifact to place at a specified path
 */
export interface Artifact {
    /**
     * From allows an artifact to reference an artifact from a previous step
     */
    from?: string;
    /**
     * mode bits to use on this file, must be a value between 0 and 0777 set when loading input artifacts.
     */
    mode?: number;
    /**
     * name of the artifact. must be unique within a template's inputs/outputs.
     */
    name: string;
    /**
     * Path is the container path to the artifact
     */
    path?: string;
    gcs?: GCSArtifact;
    git?: GitArtifact;
    http?: HTTPArtifact;
    oss?: OSSArtifact;
    raw?: RawArtifact;
    s3?: S3Artifact;
    azure?: AzureArtifact;
    archive?: {
        none?: {};
    };
    artifactGC?: {
        strategy?: 'OnWorkflowCompletion' | 'OnWorkflowDeletion';
    };
    deleted?: boolean;
}

/**
 * Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
 */
export interface Inputs {
    /**
     * Artifact are a list of artifacts passed as inputs
     */
    artifacts?: Artifact[];
    /**
     * Parameters are a list of parameters passed as inputs
     */
    parameters?: Parameter[];
}

/**
 * Outputs hold parameters, artifacts, and results from a step
 */
export interface Outputs {
    /**
     * Artifacts holds the list of output artifacts produced by a step
     */
    artifacts?: Artifact[];
    /**
     * Parameters holds the list of output parameters produced by a step
     */
    parameters?: Parameter[];
    /**
     * Result holds the result (stdout) of a script template
     */
    result?: string;
    /**
     * ExitCode holds the exit code of a script template
     */
    exitCode?: number;
}

/**
 * Parameter indicate a passed string parameter to a service template with an optional default value
 */
export interface Parameter {
    /**
     * Default is the default value to use for an input parameter if a value was not supplied
     */
    default?: string;
    /**
     * Name is the parameter name
     */
    name: string;
    /**
     * Value is the literal value to use for the parameter. If specified in the context of an input parameter, the value takes precedence over any passed values
     */
    value?: string;
    /**
     * ValueFrom is the source for the output parameter's value
     */
    valueFrom?: ValueFrom;
    /**
     * Enum holds a list of string values to choose from, for the actual value of the parameter
     */
    enum?: Array<string>;
    /**
     * Description is the parameter description
     */
    description?: string;
}

/**
 * ResourceTemplate is a template subtype to manipulate kubernetes resources
 */
export interface ResourceTemplate {
    /**
     * Action is the action to perform to the resource. Must be one of: get, create, apply, delete, replace
     */
    action: string;
    /**
     * FailureCondition is a label selector expression which describes the conditions of the k8s resource in which the step was considered failed
     */
    failureCondition?: string;
    /**
     * Manifest contains the kubernetes manifest
     */
    manifest: string;
    /**
     * SuccessCondition is a label selector expression which describes the conditions of the k8s resource in which it is acceptable to proceed to the following step
     */
    successCondition?: string;
}

/**
 * RetryStrategy provides controls on how to retry a workflow step
 */
export interface RetryStrategy {
    /**
     * Limit is the maximum number of attempts when retrying a container
     */
    limit?: number;
}

/**
 * Script is a template subtype to enable scripting through code steps
 */
export interface Script {
    /**
     * Command is the interpreter coommand to run (e.g. [python])
     */
    command: string[];
    /**
     * Image is the container image to run
     */
    image: string;
    /**
     * Source contains the source code of the script to execute
     */
    source: string;
}

/**
 * UserContainer is is a container specified by a user.
 */
export interface UserContainer {
    /**
     * Arguments to the entrypoint. The docker image's CMD is used if this is not provided.
     * Variable references $(VAR_NAME) are expanded using the container's environment.
     * If a variable cannot be resolved, the reference in the input string will be unchanged.
     * The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME).
     * Escaped references will never be expanded, regardless of whether the variable exists or not.
     * Cannot be updated.
     */
    args?: string[];

    /**
     * Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided.
     * Variable references $(VAR_NAME) are expanded using the container's environment.
     * If a variable cannot be resolved, the reference in the input string will be unchanged.
     * The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
     * regardless of whether the variable exists or not. Cannot be updated.
     *
     */
    command?: string[];
    /**
     * List of environment variables to set in the container. Cannot be updated.
     */
    env?: kubernetes.EnvVar[];
    /**
     * List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER.
     * All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources,
     * the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.
     */
    envFrom?: kubernetes.EnvFromSource[];
    /**
     * Docker image name.
     */
    image?: string;

    /**
     * Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
     */
    imagePullPolicy?: string;
    /**
     * Actions that the management system should take in response to container lifecycle events. Cannot be updated.
     */
    lifecycle?: kubernetes.Lifecycle;

    /**
     * Periodic probe of container liveness. Container will be restarted if the probe fails.
     * Cannot be updated.
     */
    livenessProbe?: kubernetes.Probe;
    /**
     * MirrorVolumeMounts will mount the same volumes specified in the main container to the sidecar (including artifacts), at the same mountPaths.
     * This enables dind daemon to partially see the same filesystem as the main container in order to use features such as docker volume binding
     */
    mirrorVolumeMounts?: boolean;
    /**
     * Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.
     */
    name: string;
    /**
     * List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses,
     * but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed.
     * Any port which is listening on the default \"0.0.0.0\" address inside a container will be accessible from the network. Cannot be updated.
     */
    ports?: kubernetes.ContainerPort[];

    /**
     * Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails.
     */
    readinessProbe?: kubernetes.Probe;
    /**
     * Compute Resources required by this container. Cannot be updated.
     */
    resources?: kubernetes.ResourceRequirements;
    /**
     * Security options the pod should run with.
     */
    securityContext?: kubernetes.SecurityContext;
    /**
     * Whether this container should allocate a buffer for stdin in the container runtime.
     * If this is not set, reads from stdin in the container will always result in EOF. Default is false.
     */
    stdin?: boolean;
    /**
     * Whether the container runtime should close the stdin channel after it has been opened by a single attach.
     * When stdin is true the stdin stream will remain open across multiple attach sessions.
     * If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin,
     * and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted.
     * If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false
     */
    stdinOnce?: boolean;

    /**
     * Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem.
     * Message written is intended to be brief final status, such as an assertion failure message.
     * Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb.
     * Defaults to /dev/termination-log. Cannot be updated.
     */
    terminationMessagePath?: string;
    /**
     * Indicate how the termination message should be populated. File will use the contents of terminationMessagePath
     * to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output
     * if the termination message file is empty and the container exited with an error.
     * The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.
     */
    terminationMessagePolicy?: string;
    /**
     * Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.
     */
    tty?: boolean;
    /**
     * volumeDevices is the list of block devices to be used by the container. This is an alpha feature and may change in the future.
     */
    volumeDevices?: kubernetes.VolumeDevice[];
    /**
     * Pod volumes to mount into the container's filesystem. Cannot be updated.
     */
    volumeMounts?: kubernetes.VolumeMount[];
    /**
     * Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.
     */
    workingDir?: string;
}

/**
 * SidecarOptions provide a way to customize the behavior of a sidecar and how it affects the main container.
 */
export interface SidecarOptions {
    /**
     * MirrorVolumeMounts will mount the same volumes specified in the main container to the sidecar (including artifacts), at the same mountPaths.
     * This enables dind daemon to partially see the same filesystem as the main container in order to use features such as docker volume binding
     */
    mirrorVolumeMounts?: boolean;
}

export interface ContainerNode extends kubernetes.Container {
    dependencies?: string[];
}

/**
 * Template is a reusable and composable unit of execution in a workflow
 */
export interface Template {
    /**
     * Optional duration in seconds relative to the StartTime that the pod may be active on a node before the system actively tries to terminate the pod;
     * value must be positive integer This field is only applicable to container and script templates.
     */
    activeDeadlineSeconds?: number;
    /**
     * Affinity sets the pod's scheduling constraints Overrides the affinity set at the workflow level (if any)
     */
    affinity?: kubernetes.Affinity;
    /**
     * Container is the main container image to run in the pod
     */
    container?: kubernetes.Container;

    containerSet?: {
        containers: ContainerNode[];
    };
    /**
     * Daemon will allow a workflow to proceed to the next step so long as the container reaches readiness
     */
    daemon?: boolean;
    /**
     * Inputs describe what inputs parameters and artifacts are supplied to this template
     */
    inputs?: Inputs;
    /**
     * Name is the name of the template
     */
    name: string;
    /**
     * NodeSelector is a selector to schedule this step of the workflow to be run on the selected node(s). Overrides the selector set at the workflow level.
     */
    nodeSelector?: {[key: string]: string};
    /**
     * Outputs describe the parameters and artifacts that this template produces
     */
    outputs?: Outputs;
    /**
     * Resource template subtype which can run k8s resources
     */
    resource?: ResourceTemplate;
    /**
     * RetryStrategy describes how to retry a template when it fails
     */
    retryStrategy?: RetryStrategy;
    /**
     * Script runs a portion of code against an interpreter
     */
    script?: Script;
    /**
     * Sidecars is a list of containers which run alongside the main container Sidecars are automatically killed when the main container completes
     */
    sidecars?: UserContainer[];
    /**
     * archiveLocation is the location in which all files related to the step will be stored (logs, artifacts, etc...).
     * Can be overridden by individual items in outputs. If omitted, will use the default
     */
    archiveLocation?: ArtifactRepository;
    /**
     * InitContainers is a list of containers which run before the main container.
     */
    initContainers?: UserContainer[];
    /**
     * Steps define a series of sequential/parallel workflow steps
     */
    steps?: WorkflowStep[][];

    /**
     * DAG template
     */
    dag?: DAGTemplate;

    /**
     * Suspend template
     */
    suspend?: {};

    /**
     * Template is the name of the template which is used as the base of this template.
     */
    template?: string;

    /**
     * TemplateRef is the reference to the template resource which is used as the base of this template.
     */
    templateRef?: TemplateRef;
}

/**
 * ValueFrom describes a location in which to obtain the value to a parameter
 */
export interface ValueFrom {
    /**
     * JQFilter expression against the resource object in resource templates
     */
    jqFilter?: string;
    /**
     * JSONPath of a resource to retrieve an output parameter value from in resource templates
     */
    jsonPath?: string;
    /**
     * Parameter reference to a step or dag task in which to retrieve an output parameter value from (e.g. '{{steps.mystep.outputs.myparam}}')
     */
    parameter?: string;
    /**
     * Path in the container to retrieve an output parameter value from in container templates
     */
    path?: string;
}

/**
 * Workflow is the definition of a workflow resource
 */
export interface Workflow {
    /**
     * APIVersion defines the versioned schema of this representation of an object.
     * Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values.
     */
    apiVersion?: string;
    /**
     * Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to.
     * Cannot be updated. In CamelCase.
     */
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: WorkflowSpec;
    status?: WorkflowStatus;
}

export const execSpec = (w: Workflow) => Object.assign({}, w.status.storedWorkflowTemplateSpec, w.spec);

export const archivalStatus = 'workflows.argoproj.io/workflow-archiving-status';

export function isWorkflowInCluster(wf: Workflow): boolean {
    if (!wf) {
        return false;
    }
    return !wf.metadata.labels[archivalStatus] || wf.metadata.labels[archivalStatus] === 'Pending' || wf.metadata.labels[archivalStatus] === 'Archived';
}

export function isArchivedWorkflow(wf?: Workflow): boolean {
    if (!wf) {
        return false;
    }

    const labelValue = wf.metadata?.labels?.[archivalStatus];
    return labelValue === 'Archived' || labelValue === 'Persisted';
}

export type NodeType = 'Pod' | 'Container' | 'Steps' | 'StepGroup' | 'DAG' | 'Retry' | 'Skipped' | 'TaskGroup' | 'Suspend';

export interface NodeStatus {
    /**
     * ID is a unique identifier of a node within the worklow
     * It is implemented as a hash of the node name, which makes the ID deterministic
     */
    id: string;

    /**
     * Display name is a human readable representation of the node. Unique within a template boundary
     */
    displayName: string;

    /**
     * Name is unique name in the node tree used to generate the node ID
     */
    name: string;

    /**
     * Type indicates type of node
     */
    type: NodeType;

    /**
     * Phase a simple, high-level summary of where the node is in its lifecycle.
     * Can be used as a state machine.
     */
    phase: NodePhase;

    /**
     * BoundaryID indicates the node ID of the associated template root node in which this node belongs to
     */
    boundaryID: string;

    /**
     * A human readable message indicating details about why the node is in this condition.
     */
    message: string;

    /**
     * Time at which this node started.
     */
    startedAt: kubernetes.Time;

    /**
     * Time at which this node completed.
     */
    finishedAt: kubernetes.Time;

    /**
     * Estimated duration in seconds.
     */
    estimatedDuration?: number;

    /**
     * Progress as numerator/denominator.
     */
    progress?: string;

    /**
     * How much resource was requested.
     */
    resourcesDuration?: {[resource: string]: number};

    /**
     * PodIP captures the IP of the pod for daemoned steps
     */
    podIP: string;

    /**
     * Daemoned tracks whether or not this node was daemoned and need to be terminated
     */
    daemoned: boolean;

    retryStrategy: RetryStrategy;

    /**
     * Outputs captures output parameter values and artifact locations
     */
    outputs: Outputs;

    /**
     * Children is a list of child node IDs
     */
    children: string[];

    /**
     * OutboundNodes tracks the node IDs which are considered "outbound" nodes to a template invocation.
     * For every invocation of a template, there are nodes which we considered as "outbound". Essentially,
     * these are last nodes in the execution sequence to run, before the template is considered completed.
     * These nodes are then connected as parents to a following step.
     *
     * In the case of single pod steps (i.e. container, script, resource templates), this list will be nil
     * since the pod itself is already considered the "outbound" node.
     * In the case of DAGs, outbound nodes are the "target" tasks (tasks with no children).
     * In the case of steps, outbound nodes are all the containers involved in the last step group.
     * NOTE: since templates are composable, the list of outbound nodes are carried upwards when
     * a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of
     * a template, will be a superset of the outbound nodes of its last children.
     */
    outboundNodes: string[];
    /**
     * TemplateName is the template name which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup)
     */
    templateName: string;
    /**
     * Inputs captures input parameter values and artifact locations supplied to this template invocation
     */
    inputs: Inputs;

    /**
     * TemplateRef is the reference to the template resource which this node corresponds to.
     * Not applicable to virtual nodes (e.g. Retry, StepGroup)
     */
    templateRef?: TemplateRef;

    /**
     * TemplateScope is the template scope in which the template of this node was retrieved.
     */
    templateScope?: string;

    /**
     * HostNodeName name of the Kubernetes node on which the Pod is running, if applicable.
     */
    hostNodeName: string;

    /**
     * Memoization
     */
    memoizationStatus: MemoizationStatus;
}

export interface TemplateRef {
    /**
     * Name is the resource name of the template.
     */
    name: string;
    /**
     * Template is the name of referred template in the resource.
     */
    template: string;
    /**
     * RuntimeResolution skips validation at creation time.
     * By enabling this option, you can create the referred workflow template before the actual runtime.
     */
    runtimeResolution?: boolean;
    /**
     * ClusterScope indicates the referred template is cluster scoped (i.e., a ClusterWorkflowTemplate).
     */
    clusterScope?: boolean;
}

export interface WorkflowStatus {
    /**
     * Phase a simple, high-level summary of where the workflow is in its lifecycle.
     */
    phase: WorkflowPhase;
    startedAt: kubernetes.Time;
    finishedAt: kubernetes.Time;
    /**
     * Estimated duration in seconds.
     */
    estimatedDuration?: number;

    /**
     * Progress as numerator/denominator.
     */
    progress?: string;
    /**
     * A human readable message indicating details about why the workflow is in this condition.
     */
    message: string;

    /**
     * Nodes is a mapping between a node ID and the node's status.
     */
    nodes: {[nodeId: string]: NodeStatus};

    /**
     * PersistentVolumeClaims tracks all PVCs that were created as part of the workflow.
     * The contents of this list are drained at the end of the workflow.
     */
    persistentVolumeClaims: kubernetes.Volume[];

    compressedNodes: string;

    /**
     * StoredTemplates is a mapping between a template ref and the node's status.
     */
    storedTemplates: {[name: string]: Template};

    /**
     * ResourcesDuration tracks how much resources were requested.
     */
    resourcesDuration?: {[resource: string]: number};

    /**
     * Conditions is a list of WorkflowConditions
     */
    conditions?: Condition[];

    /**
     * StoredWorkflowTemplateSpec is a Workflow Spec of top level WorkflowTemplate.
     */
    storedWorkflowTemplateSpec?: WorkflowSpec;

    artifactRepositoryRef?: ArtifactRepositoryRefStatus;
}

export interface Condition {
    type: ConditionType;
    status: ConditionStatus;
    message: string;
}

export type ConditionType = 'Completed' | 'SpecWarning' | 'MetricsError' | 'SubmissionError' | 'SpecError' | 'ArtifactGCError';
export type ConditionStatus = 'True' | 'False' | 'Unknown';

/**
 * WorkflowList is list of Workflow resources
 */
export interface WorkflowList {
    /**
     * APIVersion defines the versioned schema of this representation of an object.
     * Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values.
     */
    apiVersion?: string;
    items: Workflow[];
    /**
     * Kind is a string value representing the REST resource this object represents.
     * Servers may infer this from the endpoint the client submits requests to.
     */
    kind?: string;
    metadata: kubernetes.ListMeta;
}

/**
 * WorkflowSpec is the specification of a Workflow.
 */
export interface WorkflowSpec {
    /**
     * Optional duration in seconds relative to the workflow start time which the workflow is
     * allowed to run before the controller terminates the workflow. A value of zero is used to
     * terminate a Running workflow
     */
    activeDeadlineSeconds?: number;

    artifactGC?: {
        strategy?: 'OnWorkflowCompletion' | 'OnWorkflowDeletion';
    };

    /**
     * TTLStrategy limits the lifetime of a Workflow that has finished execution depending on if it
     * Succeeded or Failed. If this struct is set, once the Workflow finishes, it will be
     * deleted after the time to live expires. If this field is unset,
     * the controller config map will hold the default values.
     */
    ttlStrategy?: {
        secondsAfterCompletion?: number;
        secondsAfterSuccess?: number;
        secondsAfterFailure?: number;
    };
    /**
     * PodGC describes the strategy to use when to deleting completed pods
     */
    podGC?: {
        strategy?: string;
    };
    /**
     * SecurityContext holds pod-level security attributes and common container settings.
     */
    securityContext?: kubernetes.SecurityContext;
    /**
     * Affinity sets the scheduling constraints for all pods in the workflow. Can be overridden by an affinity specified in the template
     */
    affinity?: kubernetes.Affinity;
    /**
     * Arguments contain the parameters and artifacts sent to the workflow entrypoint.
     * Parameters are referencable globally using the 'workflow' variable prefix. e.g. {{workflow.parameters.myparam}}
     */
    arguments?: Arguments;
    /**
     * Entrypoint is a template reference to the starting point of the workflow
     */
    entrypoint?: string;
    /**
     * ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount.
     * ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.
     */
    imagePullSecrets?: kubernetes.LocalObjectReference[];

    /**
     * NodeSelector is a selector which will result in all pods of the workflow to be scheduled on the selected node(s).
     * This is able to be overridden by a nodeSelector specified in the template.
     */
    nodeSelector?: {[key: string]: string};
    /**
     * OnExit is a template reference which is invoked at the end of the workflow, irrespective of the success, failure, or error of the primary workflow.
     */
    onExit?: string;
    /**
     * ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as.
     */
    serviceAccountName?: string;
    /**
     * Templates is a list of workflow templates used in a workflow
     */
    templates?: Template[];
    /**
     * VolumeClaimTemplates is a list of claims that containers are allowed to reference.
     * The Workflow controller will create the claims at the beginning of the workflow and delete the claims upon completion of the workflow
     */
    volumeClaimTemplates?: kubernetes.PersistentVolumeClaim[];
    /**
     * Volumes is a list of volumes that can be mounted by containers in a workflow.
     */
    volumes?: kubernetes.Volume[];

    /**
     * Suspend will suspend the workflow and prevent execution of any future steps in the workflow
     */
    suspend?: boolean;

    /**
     * workflowTemplateRef is the reference to the workflow template resource to execute.
     */
    workflowTemplateRef?: WorkflowTemplateRef;
}

export interface WorkflowTemplateRef {
    /**
     * Name is the resource name of the template.
     */
    name: string;

    /**
     * ClusterScope indicates the referred template is cluster scoped (i.e., a ClusterWorkflowTemplate).
     */
    clusterScope?: boolean;
}

export interface DAGTemplate {
    /**
     * Target are one or more names of targets to execute in a DAG
     */
    targets?: string;

    /**
     * Tasks are a list of DAG tasks
     */
    tasks: DAGTask[];
}

export interface Sequence {
    start?: number;
    end?: number;
    count?: number;
}

export interface DAGTask {
    name: string;

    /**
     * Name of template to execute
     */
    template: string;

    /**
     * TemplateRef is the reference to the template resource to execute.
     */
    templateRef?: TemplateRef;

    /**
     * Arguments are the parameter and artifact arguments to the template
     */
    arguments?: Arguments;

    /**
     * Dependencies are name of other targets which this depends on
     */
    dependencies?: string[];
    onExit?: string;
    withItems?: any[];
    withParam?: string;
    withSequence?: Sequence;
}

/**
 * WorkflowStep is a reference to a template to execute in a series of step
 */
export interface WorkflowStep {
    /**
     * Arguments hold arguments to the template
     */
    arguments?: Arguments;
    /**
     * Name of the step
     */
    name?: string;
    /**
     * Template is a reference to the template to execute as the step
     */
    template?: string;
    /**
     * When is an expression in which the step should conditionally execute
     */
    when?: string;
    onExit?: string;
    /**
     * WithParam expands a step into from the value in the parameter
     */
    withParam?: string;
    withItems?: any[];
    withSequence?: Sequence;
    /**
     * TemplateRef is the reference to the template resource which is used as the base of this template.
     */
    templateRef?: TemplateRef;
}

/**
 * MemoizationStatus holds information about a node with memoization enabled.
 */

export interface MemoizationStatus {
    /**
     * Hit is true if there was a previous cache entry and false otherwise
     */
    hit: boolean;
    /**
     * Key is the value used to query the cache for an entry
     */
    key: string;
    /**
     * Cache name stores the identifier of the cache used for this node
     */
    cacheName: string;
}

export type WorkflowPhase = 'Pending' | 'Running' | 'Succeeded' | 'Failed' | 'Error';

export const WorkflowPhases: WorkflowPhase[] = ['Pending', 'Running', 'Succeeded', 'Failed', 'Error'];

export type NodePhase = '' | 'Pending' | 'Running' | 'Succeeded' | 'Skipped' | 'Failed' | 'Error' | 'Omitted';

export const NODE_PHASE = {
    PENDING: 'Pending',
    RUNNING: 'Running',
    SUCCEEDED: 'Succeeded',
    SKIPPED: 'Skipped',
    FAILED: 'Failed',
    ERROR: 'Error',
    OMITTED: 'Omitted'
};

export function getColorForNodePhase(p: NodePhase) {
    switch (p) {
        case NODE_PHASE.ERROR:
        case NODE_PHASE.FAILED:
            return '#E96D76';
        case NODE_PHASE.PENDING:
        case NODE_PHASE.RUNNING:
            return '#0DADEA';
        case NODE_PHASE.SUCCEEDED:
            return '#18BE94';
        default:
            return '#6D7F8B';
    }
}

export type ResourceScope = 'local' | 'namespaced' | 'cluster';

export interface LogEntry {
    content?: string;
    podName?: string;
}
