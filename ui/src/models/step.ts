import {ObjectMeta, Time, WatchEvent} from 'argo-ui/src/models/kubernetes';

interface Metrics {
    total?: number;
    errors?: number;
    rate?: number;
}

export interface Step {
    metadata: ObjectMeta;
    spec: {
        name: string;
        cat?: {};
        container?: {};
        expand?: {};
        filter?: string;
        flatten?: {};
        git?: {};
        group?: {};
        handler?: {};
        map?: string;
        sources: {
            name?: string;
            cron?: {schedule: string};
            stan?: {name?: string; url?: string; subject: string};
            kafka?: {
                name?: string;
                url?: string;
                topic: string;
            };
            http?: {};
        }[];
        sinks: {
            name?: string;
            log?: {};
            stan?: {name?: string; url?: string; subject: string};
            kafka?: {
                name?: string;
                url?: string;
                topic: string;
            };
            http?: {url: string};
        }[];
    };
    status?: {
        phase?: string;
        message?: string;
        replicas: number;
        lastScaledAt?: Time;
        sinkStatuses?: {[name: string]: {lastMessage?: {data: string; time: Time}; lastError?: {message: string; time: Time}; metrics?: {[name: string]: Metrics}}};
        sourceStatuses?: {
            [name: string]: {
                lastMessage?: {
                    time: Time;
                    data: string;
                };
                lastError?: {message: string; time: Time};
                pending?: number;
                metrics?: {[replica: string]: Metrics};
            };
        };
    };
}

export type StepWatchEvent = WatchEvent<Step>;
