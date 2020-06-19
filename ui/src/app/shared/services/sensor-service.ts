import {Sensor, SensorList} from '../../sensors/model/sensors';
import requests from './requests';

export class SensorService {
    public list(namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}`).then(res => res.body as SensorList);
    }
    public get(name: string, namespace: string) {
        return requests.get(`api/v1/sensors/${namespace}/${name}`).then(res => res.body as Sensor);
    }
}
