import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {Condition} from './workflows';

export interface EventSource {
    metadata: kubernetes.ObjectMeta;
    spec: {
        amqp?: {[key: string]: {}};
        azureEventsHub?: {[key: string]: {}};
        calendar?: {[key: string]: {}};
        emitter?: {[key: string]: {}};
        file?: {[key: string]: {}};
        generic?: {[key: string]: {}};
        github?: {[key: string]: {}};
        gitlab?: {[key: string]: {}};
        hdfs?: {[key: string]: {}};
        kafka?: {[key: string]: {}};
        minio?: {[key: string]: {}};
        mqtt?: {[key: string]: {}};
        nats?: {[key: string]: {}};
        nsq?: {[key: string]: {}};
        pubSub?: {[key: string]: {}};
        pulsar?: {[key: string]: {}};
        redis?: {[key: string]: {}};
        resource?: {[key: string]: {}};
        slack?: {[key: string]: {}};
        sns?: {[key: string]: {}};
        sqs?: {[key: string]: {}};
        storageGrid?: {[key: string]: {}};
        stripe?: {[key: string]: {}};
        webhook?: {[key: string]: {}};
    };
    status?: {conditions?: Condition[]};
}

export interface EventSourceList {
    metadata: kubernetes.ListMeta;
    items: EventSource[];
}

export interface LogEntry {
    namespace: string;
    eventSourceName: string;
    eventSourceType?: string;
    eventName?: string;
    level: string;
    time: kubernetes.Time;
    msg: string;
}

export type EventSourceWatchEvent = kubernetes.WatchEvent<EventSource>;

export const EventSourceTypes = [
    'amqp',
    'azureEventsHub',
    'calendar',
    'emitter',
    'file',
    'generic',
    'github',
    'gitlab',
    'hdfs',
    'kafka',
    'minio',
    'mqtt',
    'nats',
    'nsq',
    'pubSub',
    'pulsar',
    'redis',
    'resource',
    'slack',
    'sns',
    'sqs',
    'storageGrid',
    'stripe',
    'webhook'
] as const;

export type EventSourceType = typeof EventSourceTypes[number];
