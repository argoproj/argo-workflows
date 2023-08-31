import {GetUserInfoResponse, Info, Version} from '../../../models';

import requests from './requests';

let info: Promise<Info>; // we cache this globally rather than in localStorage so it is request once per page refresh

export const InfoService = {
    getInfo() {
        if (info) {
            return info;
        }
        info = requests.get(`api/v1/info`).then(res => res.body as Info);
        return info;
    },

    getVersion() {
        return requests.get(`api/v1/version`).then(res => res.body as Version);
    },

    getUserInfo() {
        return requests.get(`api/v1/userinfo`).then(res => res.body as GetUserInfoResponse);
    },

    collectEvent(name: string) {
        return requests.post(`api/v1/tracking/event`).send({name});
    }
};
