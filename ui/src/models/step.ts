import {ObjectMeta, Time, WatchEvent} from 'argo-ui/src/models/kubernetes';

export interface Metrics {
    total?: number;
    errors?: number;
    retries?: number;
    rate?: string;
}

export interface Step {
    metadata: ObjectMeta;
    spec: {
        name: string;
        cat?: {};
        code?: {};
        container?: {};
        dedupe?: {};
        expand?: {};
        filter?: {};
        flatten?: {};
        git?: {};
        group?: {};
        map?: {};
        split?: {};
        sources: {
            name?: string;
            cron?: {schedule: string};
            db?: {};
            stan?: {name?: string; url?: string; subject: string};
            kafka?: {
                name?: string;
                url?: string;
                topic: string;
            };
            http?: {
                serviceName?: string;
            };
            s3?: {
                bucket: string;
            };
            volume?: {};
        }[];
        sinks: {
            name?: string;
            db?: {};
            log?: {};
            stan?: {name?: string; url?: string; subject: string};
            kafka?: {
                name?: string;
                url?: string;
                topic: string;
            };
            http?: {url: string};
            s3?: {
                bucket: string;
            };
            volume?: {};
        }[];
    };
    status?: StepStatus;
}

export interface StepStatus {
    phase?: string;
    message?: string;
    replicas: number;
    lastScaledAt?: Time;
    sinkStatuses?: SinkStatuses;
    sourceStatuses?: SourceStatuses;
}

interface SourceStatuses {
    [name: string]: {
        pending?: number;
        metrics?: {[name: string]: Metrics};
    };
}

interface SinkStatuses {
    [name: string]: {
        metrics?: {[replica: string]: Metrics};
    };
}

export type StepWatchEvent = WatchEvent<Step>;
