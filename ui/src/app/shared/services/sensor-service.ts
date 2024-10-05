import {map} from 'rxjs/operators';
import {LogEntry, Sensor, SensorList, SensorWatchEvent} from '../../../models/sensor';
import requests from './requests';

export const SensorService = {
    list(namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}`).then(res => res.body as SensorList);
    },

    create(sensor: Sensor, namespace: string) {
        return requests
            .post(`api/v1/sensors/${namespace}`)
            .send({sensor})
            .then(res => res.body as Sensor);
    },

    get(name: string, namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}/${name}`).then(res => res.body as Sensor);
    },

    update(sensor: Sensor, namespace: string) {
        return requests
            .put(`api/v1/sensors/${namespace}/${sensor.metadata.name}`)
            .send({sensor})
            .then(res => res.body as Sensor);
    },

    delete(name: string, namespace: string) {
        return requests.delete(`api/v1/sensors/${namespace}/${name}`);
    },

    watch(namespace: string) {
        return requests.loadEventSource(`api/v1/stream/sensors/${namespace}`).pipe(map(line => line && (JSON.parse(line).result as SensorWatchEvent)));
    },

    sensorsLogs(namespace: string, name = '', triggerName = '', grep = '', tailLines = -1, container = 'main') {
        const params = ['podLogOptions.follow=true', `podLogOptions.container=${container}`];
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
        return requests.loadEventSource(`api/v1/stream/sensors/${namespace}/logs?${params.join('&')}`).pipe(map(line => line && (JSON.parse(line).result as LogEntry)));
    }
};
