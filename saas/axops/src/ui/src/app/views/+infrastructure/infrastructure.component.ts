import {Component, OnInit} from '@angular/core';

import {Host} from '../../model';
import {TimeOperations} from '../../common/timeOperations/timeOperations';
import {PerfDataService, UsageService} from '../../services';
import {HasLayoutSettings, LayoutSettings} from '../layout';

@Component({
    selector: 'ax-infrastructure',
    templateUrl: './infrastructure.html',
    styles: [ require('./infrastructure.scss') ],
})

export class InfrastructureComponent implements OnInit, HasLayoutSettings {
    public hosts: Host[];
    public details: any;
    public lineChartData: any;

    constructor(public perfDataService: PerfDataService, private usageService: UsageService) {
    }

    ngOnInit() {
        this.getSpendingsFor(TimeOperations.daysInSeconds(1), (TimeOperations.getCurrentTimeUtc() - TimeOperations.daysInSeconds(30)));
        this.getSpendingsDetailsFor((TimeOperations.getCurrentTimeUtc() - TimeOperations.daysInSeconds(30)),
            TimeOperations.getCurrentTimeUtc());
    }

    get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Infrastructure'
        };
    }

    getSpendingsFor(interval: number, minTime: number) {
        this.perfDataService.getSpendings({interval, startTime: minTime}).subscribe(result => {
            this.lineChartData = this.mapToLineChart(result);
        });
    }

    getSpendingsDetailsFor(minTime: number, maxTime: number) {
        this.usageService.getSpendingsDetailsForAsync(minTime, maxTime).subscribe( result => {
            this.details = result.data;
        });
    }

    mapToLineChart(data) {
        data = data.sort((a, b) => {
            let x = a.time;
            let y  = b.time;
            if ( x === y) {
                return 0;
            }
            return x < y ? -1 : 1;
        });

        return [{
            'key': 'Data',
            'values': data.map(item => {
                return [
                    item.time * 1000, // seconds to milliseconds
                    item.data === undefined ? 0 : item.data / 100 // cents to dollars
                ];
            })
        }];
    }
}
