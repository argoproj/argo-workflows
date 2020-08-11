import {kubernetes} from './index';

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
