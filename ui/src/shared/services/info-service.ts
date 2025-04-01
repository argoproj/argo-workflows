import {GetUserInfoResponse, Info, Version} from '../models';
import requests from './requests';
import {retryApiCall} from './retry-utils';

// Cache info globally rather than in localStorage so it is requested once per page refresh
let infoCache: Promise<Info> | null = null;

export const InfoService = {
    getInfo() {
        if (infoCache) {
            return infoCache;
        }

        // Use retry utility for critical API calls
        infoCache = retryApiCall(() => requests.get(`api/v1/info`).then(res => res.body as Info));

        return infoCache;
    },

    getVersion() {
        return retryApiCall(() => requests.get(`api/v1/version`).then(res => res.body as Version));
    },

    getUserInfo() {
        return retryApiCall(() => requests.get(`api/v1/userinfo`).then(res => res.body as GetUserInfoResponse));
    },

    collectEvent(name: string) {
        // No need to retry for analytics events
        return requests.post(`api/v1/tracking/event`).send({name});
    },

    // Clear the cache if needed (e.g., after logout)
    clearCache() {
        infoCache = null;
    }
};
