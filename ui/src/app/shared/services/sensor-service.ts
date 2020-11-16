import {LogEntry, SensorList, SensorWatchEvent} from '../../../models/sensor';
import requests from './requests';

export class SensorService {
    public list(namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}`).then(res => res.body as SensorList);
    }

    public watch(namespace: string, resourceVersion: string) {
        return requests
            .loadEventSource(`api/v1/stream/sensors/${namespace}?listOptions.resourceVersion=${resourceVersion}`)
            .map(line => line && (JSON.parse(line).result as SensorWatchEvent));
    }

    public sensorsLogs(namespace: string, name = '', triggerName = '', grep = '', tailLines = -1) {
        const params = ['podLogOptions.follow=true'];
        if (name) {
            params.push('name=' + name);
        }
        if (grep) {
            params.push('grep=' + grep);
        }
        if (triggerName) {
            params.push('triggerName=' + triggerName);
        }
        if (tailLines >= 0) {
            params.push('podLogOptions.tailLines=' + tailLines);
        }
        return requests.loadEventSource(`api/v1/stream/sensors/${namespace}/logs?${params.join('&')}`).map(line => line && (JSON.parse(line).result as LogEntry));
    }
}
