import * as kubernetes from 'argo-ui/src/models/kubernetes';

import {Condition} from './workflows';

export interface EventSource {
    metadata: kubernetes.ObjectMeta;
    spec: {
        amqp?: {[key: string]: any};
        azureEventsHub?: {[key: string]: any};
        bitbucketserver?: {[key: string]: any};
        calendar?: {[key: string]: any};
        emitter?: {[key: string]: any};
        file?: {[key: string]: any};
        generic?: {[key: string]: any};
        github?: {[key: string]: any};
        gitlab?: {[key: string]: any};
        hdfs?: {[key: string]: any};
        kafka?: {[key: string]: any};
        minio?: {[key: string]: any};
        mqtt?: {[key: string]: any};
        nats?: {[key: string]: any};
        nsq?: {[key: string]: any};
        pubSub?: {[key: string]: any};
        pulsar?: {[key: string]: any};
        redis?: {[key: string]: any};
        resource?: {[key: string]: any};
        slack?: {[key: string]: any};
        sns?: {[key: string]: any};
        sqs?: {[key: string]: any};
        storageGrid?: {[key: string]: any};
        stripe?: {[key: string]: any};
        webhook?: {[key: string]: any};
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
    'bitbucketserver',
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

export type EventSourceType = (typeof EventSourceTypes)[number];
