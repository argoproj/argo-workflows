import * as _ from 'lodash';
import { Inject } from '@angular/core';
import { Http } from '@angular/http';
import { Observable } from 'rxjs';
import { SpendingsDetail } from '../model';

export class UsageService {
    constructor( @Inject(Http) private _http) {
    }

    getSpendingsDetailsForAsync(startTime: number, endTime: number, merge = true): Observable<{ data: SpendingsDetail[] }> {
        return this._http
            .get(`v1/spendings/detail/${startTime}/${endTime}`)
            .map((res) => {
                let result = res.json();
                if (merge) {
                    // We will hide the 'system' services from this workflow
                    let axSys: SpendingsDetail = {
                        desc: '',
                        spent: 0,
                        utilization: 0,
                        cost_id: {
                            project: 'system',
                            service: 'axsys',
                            user: 'axsys',
                            app: 'axsys',
                        }
                    }, sysFlag = false,
                        processedInterval: SpendingsDetail[] = [];

                    _.forEach(result.data, (item) => {
                        if (item && item.cost_id && item.cost_id['user']) {
                            if (item.cost_id['user'].toUpperCase() === 'axsys'.toUpperCase() ||
                                item.cost_id['user'].toUpperCase() === 'k8s'.toUpperCase()) {
                                axSys.spent = axSys.spent + item.spent;
                                sysFlag = true;
                            } else {
                                processedInterval.push(item);
                            }
                        }
                    });
                    if (sysFlag) {
                        processedInterval.push(axSys);
                    }

                    result.data = processedInterval;
                }

                return result;
            });
    }
}
