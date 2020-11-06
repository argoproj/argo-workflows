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
    eventName?: string;
    level: string;
    msg: string;
}

export const eventTypes: { [key: string]: string } = {
    amqp: 'AMQPEvent',
    azureEventsHub: 'AzureEventsHubEvent',
    calendar: 'CalendarEvent',
    emitter: 'EmitterEvent',
    file: 'FileEvent',
    generic: 'GenericEvent',
    github: 'GithubEvent',
    gitlab: 'GitlabEvent',
    hdfs: 'HDFSEvent',
    kafka: 'KafkaEvent',
    minio: 'MinioEvent',
    mqtt: 'MQTTEvent',
    nats: 'NATSEvent',
    nsq: 'NSQEvent',
    pubSub: 'PubSubEvent',
    pulsar: 'PulsarEvent',
    redis: 'RedisEvent',
    resource: 'ResourceEvent',
    slack: 'SlackEvent',
    sns: 'SNSEvent',
    sqs: 'SQSEvent',
    storageGrid: 'StorageGridEvent',
    stripe: 'StripeEvent',
    webhook: 'WebhookEvent'
};