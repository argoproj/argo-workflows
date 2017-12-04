import * as kubernetes from './kubernetes';

export type Item = any;

export interface Parameter {
    name: string;
    value: string;
    default: string;
    path: string;
}

export interface S3Bucket {
    endpoint: string;
    bucket: string;
    region: string;
    insecure: boolean;
    accessKeySecret: kubernetes.SecretKeySelector;
    secretKeySecret: kubernetes.SecretKeySelector;
}

export interface S3Artifact extends S3Bucket {
    key: string;
}

export interface GitArtifact {
    repo: string;
    revision: string;
    usernameSecret: kubernetes.SecretKeySelector;
    passwordSecret: kubernetes.SecretKeySelector;
}

export interface HTTPArtifact {
    url: string;
}

export interface ArtifactLocation {
    s3: S3Artifact;
    git: GitArtifact;
    http: HTTPArtifact;
}

export interface Artifact extends ArtifactLocation {
    name: string;
    path: string;
    mode: number;
    from: string;
}

export interface Outputs {
    parameters: Parameter[];
    artifacts: Artifact[];
    result: string;
}

export interface NodeStatus {
    id: string;
    name: string;
    phase: string;
    podIP: string;
    daemoned: boolean;
    outputs: Outputs;
    children: string[];
    startedAt: string;
    finishedAt: string;
}

export interface WorkflowStatus {
    nodes: {[name: string]: NodeStatus};
    persistentVolumeClaims: kubernetes.Volume;
}

export interface Inputs {
    parameters: Parameter[];
    artifacts: Artifact[];
}

export interface WorkflowStep {
    name: string;
    template: string;
    arguments: Arguments;
    withItems: Item[];
    when: string;
}

export interface Script {
    image: string;
    command: string[];
    source: string;
}

export interface SidecarOptions {
  mirrorVolumeMounts: boolean;
}

export interface Sidecar extends kubernetes.Container, SidecarOptions {
}

export interface Template {
    name: string;
    inputs: Inputs;
    outputs: Outputs;
    daemon: boolean;
    steps: WorkflowStep[][];
    container: kubernetes.Container;
    script: Script;
    sidecars: Sidecar[];
    archiveLocation: ArtifactLocation;
}

export interface WorkflowSpec {
    templates: Template[];
    entrypoint: string;
    arguments: Arguments;
    volumes: kubernetes.Volume[];
    volumeClaimTemplates: kubernetes.PersistentVolumeClaim[];
    timeout: string;
}

export interface Workflow extends kubernetes.TypeMeta {
    metadata: kubernetes.ObjectMeta;
    spec: WorkflowSpec;
    status: WorkflowStatus;
}

export interface WorkflowList extends kubernetes.TypeMeta {
    metadata: kubernetes.ObjectMeta;
    items: Workflow[];
}

export interface Arguments {
    parameters: Parameter[];
    artifacts: Artifact[];
}

export const NODE_PHASE = {
  RUNNING: 'Running',
  SUCCEEDED: 'Succeeded',
  SKIPPED: 'Skipped',
  FAILED: 'Failed',
  ERROR: 'Error',
};
