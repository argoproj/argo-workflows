import {EventSourceList, LogEntry} from '../../../models/event-source';
import requests from './requests';

export class EventSourceService {
    public list(namespace: string) {
        return requests.get(`api/v1/event-sources/${namespace}`).then(res => res.body as EventSourceList);
    }

    public eventSourcesLogs(namespace: string, name = '', eventSourceType = '', eventName = '', tailLines = -1) {
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
        if (tailLines >= 0) {
            params.push('podLogOptions.tailLines=' + tailLines);
        }
        return requests.loadEventSource(`api/v1/stream/event-sources/${namespace}/logs?${params.join('&')}`).map(line => JSON.parse(line).result as LogEntry);
    }
}
