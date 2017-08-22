import { Injectable } from '@angular/core';

import { Observable } from 'rxjs/Observable';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { PerfData, PerfBreakDownData } from '../model';

@Injectable()
export class PerfDataService {
    constructor( private _http: Http) {
    }

    getSpendings(
        params?: {
        interval: number,
        startTime: number,
        endTime?: number
        },
        showLoader?: boolean
    ): Observable<PerfData[]> {
        let search = new URLSearchParams();
        search.set('min_time', params.startTime.toFixed());
        let customHeader = new Headers();
        if (showLoader) {
            customHeader.append('isUpdated', 'true');
        }
        if (params.endTime) {
            search.set('max_time', params.endTime.toFixed());
        }
        return this._http.get(`v1/spendings/perf/${params.interval.toFixed()}?${search.toString()}`,
            { headers: customHeader })
            .map(res => <PerfData[]>res.json().data);
    }

    /**
     * Loads spending breakdown data for specified cost id type (user, service etc) grouped by specified interval between
     * start date and end date.
     */
    getSpendingsBreakDownBy(
        params?: {
            name: string,
            interval: number,
            startTime: number,
            endTime?: number,
            filterBy?: { by: string, value: string },
            filter?: { by: string, value: string }[],
        },
        showLoader?: boolean
    ): Observable<PerfBreakDownData[]> {
        let search = new URLSearchParams();
        search.set('by', params.name);
        search.set('min_time', params.startTime.toFixed());

        let customHeader = new Headers();
        if (showLoader) {
            customHeader.append('isUpdated', 'true');
        }
        if (params.endTime) {
            search.set('max_time', params.endTime.toFixed());
        }
        if (params.filterBy) {
            search.set('filterBy', params.filterBy.by);
            search.set('filterByValue', params.filterBy.value);
        }
        if (params.filter && params.filter.length > 0) {
            search.set('filter', params.filter.map(f => f.by + ':' + f.value).join(';'));
        }
        return this._http.get(`v1/spendings/breakdown/${params.interval.toFixed()}?${search.toString()}`,
            { headers: customHeader })
            .map((res) => {
                let data = <PerfBreakDownData[]>res.json().data;
                let systemItem: PerfBreakDownData = null;
                for (let i = data.length - 1; i >= 0; i--) {
                    let item = data[i];
                    if (item.is_system) {
                        if (systemItem && systemItem.time === item.time) {
                            systemItem.data += item.data;
                            data.splice(i, 1);
                        } else {
                            item.name = 'axsys';
                            systemItem = item;
                        }
                    }
                }
                return data;
            });
    }
}
