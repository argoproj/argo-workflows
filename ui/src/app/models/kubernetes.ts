export type Time = any;
export type Volume = any;
export type EnvFromSource = any;
export type EnvVarSource = any;
export type ResourceRequirements = any;
export type VolumeMount = any;
export type Probe = any;
export type Lifecycle = any;
export type TerminationMessagePolicy = any;
export type PullPolicy = any;
export type SecurityContext = any;
export type PersistentVolumeClaim = any;

export interface ObjectMeta {
    name?: string;
    generateName?: string;
    namespace?: string;
    selfLink?: string;
    uid?: string;
    resourceVersion?: string;
    generation?: number;
    creationTimestamp?: Time;
    deletionTimestamp?: Time;
    deletionGracePeriodSeconds?: number;
    labels?: {[name: string]: string};
    annotations?: {[name: string]: string};
    ownerReferences?: any[];
    initializers?: any;
    finalizers?: string[];
    clusterName?: string;
}

export interface TypeMeta {
    kind: string;
    apiVersion: string;
}

export interface LocalObjectReference {
    name: string;
}

export interface SecretKeySelector extends LocalObjectReference {
    key: string;
    optional: boolean;
}

export interface ContainerPort {
    name: string;
    hostPort: number;
    containerPort: number;
    protocol: string;
    hostIP: string;
}

export interface EnvVar {
    name: string;
    value: string;
    valueFrom: EnvVarSource;
}

export interface Container {
    name: string;
    image: string;
    command: string[];
    args: string[];
    workingDir: string;
    ports: ContainerPort[];
    envFrom: EnvFromSource[];
    env: EnvVar[];
    resources: ResourceRequirements;
    volumeMounts: VolumeMount[];
    livenessProbe: Probe;
    readinessProbe: Probe;
    lifecycle: Lifecycle;
    terminationMessagePath: string;
    terminationMessagePolicy: TerminationMessagePolicy;
    imagePullPolicy: PullPolicy;
    securityContext: SecurityContext;
    stdin: boolean;
    stdinOnce: boolean;
    tty: boolean;
}
