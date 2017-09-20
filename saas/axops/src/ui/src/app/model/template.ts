export interface BaseTemplate {
    type?: string;
    version?: number;
    name?: string;
    description?: string;
    labels?: { [name: string]: string };
    id?: string;
    repo?: string;
    branch?: string;
    revision?: string;
}

export interface Input {
    description?: string;
}

export interface InputParameter extends Input {
    default?: string;
    options?: any[];
    regex?: string;
}

export interface InputArtifact extends Input {
    path?: string;
    from?: string;
}

export interface InputVolume extends Input {
    from?: string;
    mount_path?: string;
    details?: any;
}

export interface InputFixture extends Input {}

export interface InputDNSDomain extends Input {
    default?: string;
}

export interface Inputs {
    parameters?: { [name: string]: InputParameter };
    artifacts?: { [name: string]: InputArtifact };
    volumes?: { [name: string]: InputVolume };
    fixtures?: { [name: string]: InputFixture };
    dns_domains?: { [name: string]: InputDNSDomain };
}

export interface OutputArtifact {
    path?: string;
    excludes?: string[];
    archive_mode?: string;
    storage_method?: string;
    from?: string;
    retention?: string;

    // TODO (alexander)?: probably dropped.
    meta_data?: string;
}

type OutputArtifacts = { [name: string]: OutputArtifact };

export interface Outputs {
    artifacts?: OutputArtifacts;
}

export interface NameValuePair {
    name?: string;
    value?: string;
}

export interface ContainerResources {
    mem_mib?: number;
    cpu_cores?: number;
}

export interface ContainerProbeExec {
    command?: string;
}

export interface ContainerProbeHttpRequest {
    path?: string;
    port?: number;
    http_headers?: NameValuePair[];
}

export interface ContainerProbe {
    initial_delay_seconds?: number;
    timeout_seconds?: number;
    period_seconds?: number;
    failure_threshold?: number;
    success_threshold?: number;
    exec?: ContainerProbeExec;
    http_get?: ContainerProbeHttpRequest;
}

export interface ContainerTemplate extends BaseTemplate {
    inputs?: Inputs;
    outputs?: Outputs;
    resources?: ContainerResources;
    image?: string;
    command?: string[];
    args?: string[];
    env?: NameValuePair[];
    liveness_probe?: ContainerProbe;
    readiness_probe?: ContainerProbe;
    image_pull_policy?: string;
    annotations?: {[name: string]: string};
}

export interface TemplateRef {
    template?: Template;
    arguments?: { [name: string]: string};
}

export interface WorkflowStep extends InlineContainerTemplateRef {
    flags?: string;
}

export interface FixtureRequirement {
    class?: string;
    name?: string;
    attributes?: { [name: string]: string};
}

export interface FixtureTemplateRef extends TemplateRef, FixtureRequirement {}

type FixtureRequirements = { [name: string]: FixtureTemplateRef }[];

export interface VolumeRequirement {
    name?: string;
    storage_class?: string;
    size_gb?: string;
}

type VolumeRequirements = {[name: string]: VolumeRequirement};

export interface TerminationPolicy {
    spending_cents?: string;
    time_seconds?: string;
}

export interface WorkflowTemplate extends BaseTemplate {
    inputs?: Inputs;
    outputs?: Outputs;
    steps?: {[name: string]: WorkflowStep}[];
    fixtures?: FixtureRequirements;
    volumes?: VolumeRequirements;
    artifact_tags?: string[];
    terminationPolicy?: TerminationPolicy;
}

export interface Scale {
    min?: number;
    max?: number;
}

export interface ExternalRoute {
    dns_prefix?: string;
    dns_domain?: string;
    dns_name?: string;
    target_port?: string;
    ip_white_list?: string[];
    visibility?: string;
}

export interface Port {
    port?: string;
    target_port?: string;
}

export interface InternalRoute {
    name?: string;
    ports?: Port[];
}

export interface InlineContainerTemplateRef extends TemplateRef, ContainerTemplate {}

export interface RollingUpdateStrategy {
    max_surge?: string;
    max_unavailable?: string;
}

export interface Strategy {
    type?: string;
    rolling_update?: RollingUpdateStrategy;
}

export interface DeploymentTemplate extends BaseTemplate {
    inputs?: Inputs;
    application_name?: string;
    deployment_name?: string;
    scale?: Scale;
    external_routes?: ExternalRoute[];
    internal_routes?: InternalRoute[];
    containers?: {[name: string]: InlineContainerTemplateRef};
    fixtures?: FixtureRequirements;
    volumes?: VolumeRequirements;
    termination_policy?: TerminationPolicy;
    min_ready_seconds?: number;
    strategy?: Strategy;
}

export interface Template extends ContainerTemplate, WorkflowTemplate, DeploymentTemplate {
    jobs_wait?: number;
    jobs_run?: number;
    jobs_fail?: number;
    jobs_success?: number;
    cost?: number;

    // TODO (alexander)?: remove UI only field
    selected?: boolean;
}

export const TEMPLATE_TYPES = {
    container: 'container',
    workflow: 'workflow',
    deployment: 'deployment',
};

export const TemplateFieldNames = {
    id: 'id',
    type: 'type',
    version: 'version',
    name: 'name',
    dnsName: 'dns_name',
    description: 'description',
    fixtures: 'fixtures',
    cost: 'cost',
    container: 'container',
    inputs: 'inputs',
    outputs: 'outputs',
    steps: 'steps',
    branch: 'branch',
    repo: 'repo',
    jobs_wait: 'jobs_wait',
    jobs_run: 'jobs_run',
    jobs_fail: 'jobs_fail',
    jobs_success: 'jobs_success'
};
