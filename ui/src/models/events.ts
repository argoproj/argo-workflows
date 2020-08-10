import {kubernetes} from './index';

export interface Event {
    metadata: kubernetes.ObjectMeta;
    reason: string;
    message: string;
    lastTimestamp: kubernetes.Time;
    type: string;
}
