import {uiUrl} from '../base';
import {GetUserInfoResponse, Info, Link, Version} from '../models';
import requests from './requests';

let info: Promise<Info>; // we cache this globally rather than in localStorage so it is request once per page refresh

// resolveLinks prefixes relative link URLs with the base UI URL. A link is
// considered relative when it has no protocol (i.e. `new URL` cannot parse it).
export function resolveLinks(links?: Link[]): Link[] {
    return (links || []).map(link => {
        try {
            new URL(link.url);
            return link;
        } catch {
            // strip any leading slashes so we don't end up with a double
            // slash once uiUrl prefixes the base UI URL (which ends in `/`)
            return {
                ...link,
                url: uiUrl(link.url.replace(/^\/+/, ''))
            };
        }
    });
}

export const InfoService = {
    getInfo() {
        if (info) {
            return info;
        }
        info = requests.get(`api/v1/info`).then(res => {
            const info = res.body as Info;
            info.links = resolveLinks(info.links);
            return info;
        });
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
