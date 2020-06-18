import {ListMeta, ObjectMeta} from 'argo-ui/src/models/kubernetes';

export interface Sensor {
    metadata: ObjectMeta;
    spec: SensorSpec;
    status?: SensorStatus;
}

export interface SensorList {
    items: Sensor[];
    metadata: ListMeta;
}

export interface SensorDependency {
    name: string;
}

export interface SensorTriggerTemplate {
    name: string;
}

export interface SensorTrigger {
    template: SensorTriggerTemplate;
}

export interface SensorSpec {
    dependencies: SensorDependency[];
    triggers: SensorTrigger[];
}

export interface SensorStatus {
    phase: 'Complete' | 'Active' | 'Error' | '';
    message: string;
}
