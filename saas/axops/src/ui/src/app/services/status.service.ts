import { SystemStatus } from '../model';
import { Injectable } from '@angular/core';
import { Http } from '@angular/http';
import { Observable } from 'rxjs';

@Injectable()
export class StatusService {
    constructor(private http: Http) {
    }

    public getStatusAsync(): Observable<SystemStatus> {
        return this.http.get(`v1/system/status`).map(res => res.json());
    }
}
