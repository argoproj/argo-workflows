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
    items: EventSource[];
}

export interface EventSourceLogEntry {
    namespace: string;
    eventSourceName: string;
    eventSourceType?: string;
    eventName?: string;
    level: string;
    time: kubernetes.Time;
    msg: string;
}

export const eventTypes: {[key: string]: string} = {
    amqp: 'AMQP',
    azureEventsHub: 'AzureEventsHub',
    calendar: 'Calendar',
    emitter: 'Emitter',
    file: 'File',
    generic: 'Generic',
    github: 'Github',
    gitlab: 'Gitlab',
    hdfs: 'HDFS',
    kafka: 'Kafka',
    minio: 'Minio',
    mqtt: 'MQTT',
    nats: 'NATS',
    nsq: 'NSQ',
    pubSub: 'PubSub',
    pulsar: 'Pulsar',
    redis: 'Redis',
    resource: 'Resource',
    slack: 'Slack',
    sns: 'SNS',
    sqs: 'SQS',
    storageGrid: 'StorageGrid',
    stripe: 'Stripe',
    webhook: 'Webhook'
};
