import {ListMeta, ObjectMeta, Time} from 'argo-ui/src/models/kubernetes';

export interface Sensor {
    metadata: ObjectMeta;
    spec: SensorSpec;
    status?: SensorStatus;
}

export interface SensorList {
    items: Sensor[];
    metadata: ListMeta;
}

export interface EventDependency {
    name: string;
    eventName: string;
    gatewayName: string;
}

export interface TriggerParameterSource {
    dependencyName: string;
}

export interface TriggerParameter {
    src?: TriggerParameterSource;
    dest: string;
}

export type KubernetesResourceOperation = "create" | "update" | "patch";

export interface StandardK8STrigger {
    operation: KubernetesResourceOperation;
    resource: string;
    parameters?: TriggerParameter[];
}

export type ArgoWorkflowOperation = "submit" | "suspend" | "resubmit" | "retry" | "resume";

export interface ArgoWorkflowTrigger {
    operation: ArgoWorkflowOperation;
    parameters?: TriggerParameter[];
}

export interface Template {
    name: string;
    k8s?: StandardK8STrigger;
    argoWorkflow?: ArgoWorkflowTrigger;
}

export interface Trigger {
    template?: Template;
}

export interface SensorSpec {
    dependencies: EventDependency[];
    triggers: Trigger[];
}

export interface Event {
    data: string;
}

export interface NodeStatus {
    name: string;
    displayName: string;
    type: string;
    phase: NodePhase;
    event?: Event;
    startedAt?: Time;
    completedAt?: Time;
    message?: string;
}

export type NodePhase = 'Complete' | 'Active' | 'Error' | '';

export type TriggerCycleState = 'Success' | 'Failure';

export interface SensorStatus {
    phase: NodePhase;
    message: string;
    startedAt?: Time;
    completedAt?: Time;
    lastCycleTime?: Time;
    triggerCycleStatus?: TriggerCycleState;
    triggerCycleCount?: number;
    nodes: { [nodeId: string]: NodeStatus };
}
