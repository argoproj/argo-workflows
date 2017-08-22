import { Injectable } from '@angular/core';
import { Http } from '@angular/http';
import { TimeOperations } from './../common/timeOperations/timeOperations';
import { Observable } from 'rxjs';

import { BuildPerformance } from '../model';

@Injectable()
export class PerformanceService {
    constructor(private http: Http) {
    }

    public getBuildPerformanceAsync(): Observable<BuildPerformance[]> {
        return this.http.get(`/v1/builds/perf/${TimeOperations.daysInSeconds(1)}?min_time=
            ${(TimeOperations.getCurrentTimeUtc() - TimeOperations.daysInSeconds(30))}`).map(res => res.json());
    }
}
