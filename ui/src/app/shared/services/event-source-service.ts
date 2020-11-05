import {LogEntry} from '../../../models';
import {EventSourceList} from '../../../models/event-source';
import requests from './requests';

export class EventSourceService {
    public list(namespace: string) {
        return requests.get(`api/v1/event-sources/${namespace}`).then(res => res.body as EventSourceList);
    }

    public eventSourcesLogs(namespace: string, tailLines = -1) {
        return requests
            .loadEventSource(`api/v1/stream/event-sources/${namespace}/logs?podLogOptions.follow=true&${tailLines >= 0 ? `podLogOptions.tailLines=${tailLines}` : ''}`)
            .map(line => JSON.parse(line).result as LogEntry);
    }
}
