import {GetUserInfoResponse, Info, Version} from '../../../models';

import requests from './requests';

let info: Promise<Info>; // we cache this globally rather than in localStorage so it is request once per page refresh

export class InfoService {
    public getInfo() {
        if (info) {
            return info;
        }
        info = requests.get(`api/v1/info`).then(res => res.body as Info);
        return info;
    }

    public getVersion() {
        return requests.get(`api/v1/version`).then(res => res.body as Version);
    }

    public getUserInfo() {
        return requests.get(`api/v1/userinfo`).then(res => res.body as GetUserInfoResponse);
    }

    public collectEvent(param: Map<string, string>) {
        const obj = Object.create(null);
        for (const [k, v] of param) {
            obj[k] = v;
        }
        return requests.post(`api/v1/tracking/event`).send(obj);
    }
}

export const EventParams = ['name'] as const;
export type EventParam = typeof EventParams[number];
