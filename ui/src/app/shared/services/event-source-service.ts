import {EventSourceList, EventSourceLogEntry} from '../../../models';
import requests from './requests';

export class EventSourceService {
    public list(namespace: string) {
        return requests.get(`api/v1/event-sources/${namespace}`).then(res => res.body as EventSourceList);
    }

    public eventSourcesLogs(namespace: string, name = '', eventSourceType = '', eventName = '', tailLines = -1) {
        return requests
            .loadEventSource(
                `api/v1/stream/event-sources/${namespace}/logs?name=${name || ''}&eventSourceType=${eventSourceType || ''}&eventName=${eventName || ''}&podLogOptions.follow=true&${
                    tailLines >= 0 ? `podLogOptions.tailLines=${tailLines}` : ''
                }`
            )
            .map(line => JSON.parse(line).result as EventSourceLogEntry);
    }
}
