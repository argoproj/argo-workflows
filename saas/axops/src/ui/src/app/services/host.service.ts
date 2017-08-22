import {Http, Headers} from '@angular/http';
import {Inject} from '@angular/core';

export class HostService {
    constructor(@Inject(Http) private _http) {
    }

    getHostsAsync (isUpdated = false) {
        let customHeader = new Headers();
        if (isUpdated) {
            customHeader.append('isUpdated', isUpdated.toString());
        }

        return this._http.get('v1/system/hosts', {headers: customHeader})
            .map( res => res.json());
    }

    getHostByIdAsync(clusterId: string) {
        return this._http.get(`v1/system/hosts/${clusterId}`).map(
            res => {
                return res.json();
            });
    }
}
