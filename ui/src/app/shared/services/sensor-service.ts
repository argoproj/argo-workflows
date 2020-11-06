import {SensorList, SensorLogEntry} from '../../../models';

import requests from './requests';

export class SensorService {
    public list(namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}`).then(res => res.body as SensorList);
    }

    public sensorsLogs(namespace: string, name = '', triggerName = '', tailLines = -1) {
        return requests
            .loadEventSource(
                `api/v1/stream/sensors/${namespace}/logs?name=${name || ''}&triggerName=${triggerName || ''}&podLogOptions.follow=true&${
                    tailLines >= 0 ? `podLogOptions.tailLines=${tailLines}` : ''
                }`
            )
            .map(line => JSON.parse(line).result as SensorLogEntry);
    }
}
