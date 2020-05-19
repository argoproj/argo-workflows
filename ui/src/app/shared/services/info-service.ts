import {Info, User, Version} from '../../../models';

import requests from './requests';

export class InfoService {
    public getInfo() {
        return requests.get(`api/v1/info`).then(res => res.body as Info);
    }
    public getUser() {
        return requests.get(`api/v1/user`).then(res => res.body as User);
    }
    public getVersion() {
        return requests.get(`api/v1/version`).then(res => res.body as Version);
    }
}
