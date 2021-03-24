import {EventSource, EventSourceList, EventSourceWatchEvent, LogEntry} from '../../../models/event-source';
import requests from './requests';

export class EventSourceService {
    public create(eventSource: EventSource, namespace: string) {
        return requests
            .post(`api/v1/event-sources/${namespace}`)
            .send({eventSource})
            .then(res => res.body as EventSource);
    }

    public list(namespace: string) {
        return requests.get(`api/v1/event-sources/${namespace}`).then(res => res.body as EventSourceList);
    }

    public get(name: string, namespace: string) {
        return requests.get(`api/v1/event-sources/${namespace}/${name}`).then(res => res.body as EventSource);
    }

    public update(eventSource: EventSource, name: string, namespace: string) {
        return requests
            .put(`api/v1/event-sources/${namespace}/${name}`)
            .send({eventSource})
            .then(res => res.body as EventSource);
    }

    public delete(name: string, namespace: string) {
        return requests.delete(`api/v1/event-sources/${namespace}/${name}`);
    }

    public watch(namespace: string) {
        return requests.loadEventSource(`api/v1/stream/event-sources/${namespace}`).map(line => line && (JSON.parse(line).result as EventSourceWatchEvent));
    }

    public eventSourcesLogs(namespace: string, name = '', eventSourceType = '', eventName = '', grep = '', tailLines = -1) {
        const params = ['podLogOptions.follow=true'];
        if (name) {
            params.push('name=' + name);
        }
        if (eventSourceType) {
            params.push('eventSourceType=' + eventSourceType);
        }
        if (eventName) {
            params.push('eventName=' + eventName);
        }
        if (grep) {
            params.push('grep=' + grep);
        }
        if (tailLines >= 0) {
            params.push('podLogOptions.tailLines=' + tailLines);
        }
        return requests.loadEventSource(`api/v1/stream/event-sources/${namespace}/logs?${params.join('&')}`).map(line => line && (JSON.parse(line).result as LogEntry));
    }
}
