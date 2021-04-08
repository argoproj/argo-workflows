import {ObjectMeta, WatchEvent} from 'argo-ui/src/models/kubernetes';

interface Metrics {
    total?: number;
    errors?: number;
}

export interface Step {
    metadata: ObjectMeta;
    spec: {
        name: string;
        container?: {};
        filter?: string;
        git?: {};
        group?: {};
        handler?: {};
        map?: string;
        sources: {
            name?: string;
            stan: {name?: string; url?: string; subject: string};
            kafka?: {
                name?: string;
                url?: string;
                topic: string;
            };
        }[];
        sinks: {
            name?: string;
            stan: {name?: string; url?: string; subject: string};
            kafka?: {
                name?: string;
                url?: string;
                topic: string;
            };
        }[];
    };
    status?: {
        phase?: string;
        message?: string;
        replicas: number;
        sinkStatuses?: {[name: string]: {lastMessage?: {data: string}; pending?: number; metrics?: {[name: string]: Metrics}}};
        sourceStatuses?: {[name: string]: {lastMessage?: {data: string}; pending?: number; metrics?: {[replica: string]: Metrics}}};
    };
}

export type StepWatchEvent = WatchEvent<Step>;
