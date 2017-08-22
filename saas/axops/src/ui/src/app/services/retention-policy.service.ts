import { Inject, Injectable } from '@angular/core';
import { Http, Headers } from '@angular/http';

@Injectable()
export class RetentionPolicyService {
    constructor(@Inject(Http) private _http) {
    }

    getRetentionPolicies(hideLoader?: boolean) {
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this._http.get(`v1/retention_policies`, {headers: headers})
            .map(res => res.json());
    }

    updateRetentionPolicy(name: string, policy: number, hideLoader?: boolean) {
        let customHeader = new Headers();
        if (hideLoader) {
            customHeader.append('isUpdated', hideLoader.toString());
        }
        return this._http.put(`v1/retention_policies/${name}`, {policy: policy}, { headers: customHeader })
            .map(res => res.json());
    }
}
