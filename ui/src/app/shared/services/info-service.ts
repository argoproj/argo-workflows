import {Info} from '../../../models';
import requests from './requests';

export class InfoService {
    public get() {
        return requests.get(`api/v1/info`).then(res => res.body as Info);
    }
}
