import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {Condition} from './workflows';

export interface Sensor {
    metadata: kubernetes.ObjectMeta;
    spec: {
        dependencies: {
            name: string;
            eventSourceName: string;
            eventName: string;
        }[];
        triggers: {
            template?: {
                name: string;
                conditions?: string;
                argoWorkflow?: {};
                awsLambda?: {};
                custom?: {};
                http?: {};
                k8s?: {};
                kafka?: {};
                nats?: {};
                openWhisk?: {};
                slack?: {};
            };
        }[];
    };
    status?: {conditions?: Condition[]};
}

export interface SensorList {
    items: Sensor[];
}

export interface SensorLogEntry {
    namespace: string;
    sensorName: string;
    triggerName?: string;
    level: string;
    msg: string;
}
