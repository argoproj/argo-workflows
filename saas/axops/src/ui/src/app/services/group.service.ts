import { Injectable } from '@angular/core';
import { Http } from '@angular/http';
import { Observable } from 'rxjs/Observable';

import { AxHeaders } from './headers';
import { Group } from '../model';

@Injectable()
export class GroupService {
    constructor(private http: Http) {
    }

    getGroups(hideLoader = true): Observable<{data: Group[]}> {
        return this.http.get('v1/groups', { headers: new AxHeaders({noLoader: hideLoader}) }).map(res => res.json());
    }
}
