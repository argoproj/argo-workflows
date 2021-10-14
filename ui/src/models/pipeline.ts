import {ListMeta, ObjectMeta, Time, WatchEvent} from 'argo-ui/src/models/kubernetes';

export interface Pipeline {
    metadata: ObjectMeta;
    status?: {
        conditions?: {type?: string}[];
        phase?: string;
        message?: string;
    };
}

export interface PipelineList {
    metadata: ListMeta;
    items: Pipeline[];
}

export interface LogEntry {
    namespace: string;
    stepName?: string;
    time: Time;
    msg: string;
}

export type PipelineWatchEvent = WatchEvent<Pipeline>;
