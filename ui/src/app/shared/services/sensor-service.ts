import {LogEntry, Sensor, SensorList, SensorWatchEvent} from '../../../models/sensor';
import requests from './requests';

export class SensorService {
    public list(namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}`).then(res => res.body as SensorList);
    }

    public create(sensor: Sensor, namespace: string) {
        return requests
            .post(`api/v1/sensors/${namespace}`)
            .send({sensor})
            .then(res => res.body as Sensor);
    }

    public get(name: string, namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}/${name}`).then(res => res.body as Sensor);
    }

    public update(sensor: Sensor, namespace: string) {
        return requests
            .put(`api/v1/sensors/${namespace}/${sensor.metadata.name}`)
            .send({sensor})
            .then(res => res.body as Sensor);
    }

    public delete(name: string, namespace: string) {
        return requests.delete(`api/v1/sensors/${namespace}/${name}`);
    }

    public watch(namespace: string) {
        return requests.loadEventSource(`api/v1/stream/sensors/${namespace}`).map(line => line && (JSON.parse(line).result as SensorWatchEvent));
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
