import {Arguments, kubernetes, WorkflowTemplateRef} from './index';

// labels that Argo Events puts on the resources its Sensor triggers create
export const eventsLabels = {
    sensor: 'events.argoproj.io/sensor'
};

export interface Event {
    metadata: kubernetes.ObjectMeta;
    involvedObject: {
        name: string;
        kind: string;
    };
    reason: string;
    message: string;
    lastTimestamp: kubernetes.Time;
    type: string;
}

export interface WorkflowEventBindingList {
    metadata: kubernetes.ListMeta;
    items: WorkflowEventBinding[];
}

export interface WorkflowEventBinding {
    metadata: kubernetes.ObjectMeta;
    spec: {
        event: {selector: string};
        submit?: {
            workflowTemplateRef: WorkflowTemplateRef;
            arguments?: Arguments;
        };
    };
}
