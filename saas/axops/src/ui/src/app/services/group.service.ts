import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Observable';

import {Group} from '../model';

@Injectable()
export class GroupService {
    constructor(private http: Http) {
    }

    getGroups(): Observable<{data: Group[]}> {
        return this.http.get('v1/groups').map(res => res.json());
    }
}
