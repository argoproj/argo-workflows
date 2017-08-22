import { Injectable } from '@angular/core';
import { Http, URLSearchParams } from '@angular/http';

import { SystemRequest, SystemRequestType } from '../model';
import { AxHeaders } from './headers';

@Injectable()
export class SystemRequestService {

    constructor(private http: Http) {}

    public getSystemRequests(params: { type?: SystemRequestType }, hideLoader?: boolean): Promise<SystemRequest[]> {
        let search = new URLSearchParams();
        if (params.type) {
            search.set('type', params.type.toString());
        }
        return this.http.get('v1/system_requests', { headers: new AxHeaders({ noLoader: hideLoader }), search: search}).toPromise().then(res => res.json().data);
    }
}
