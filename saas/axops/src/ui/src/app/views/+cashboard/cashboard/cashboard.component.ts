import * as moment from 'moment';
import * as _ from 'lodash';
import { Component, OnInit, OnDestroy, ViewChild } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs';

import { SpendingsChartComponent } from '../spendings-chart/spendings-chart.component';
import { MathOperations } from '../../../common/mathOperations/mathOperations';
import { PerfDataService, UsageService } from '../../../services';
import { PerfData, SpendingsDetail, } from '../../../model';
import { PricedItem } from '../../../common/chartSpendings';
import { COST_IDS, DateIntervalCashboard, Dictionary, PriciesItemGroups } from '../cashboard.view-models';
import { SpendingsLoader, SpendingsLoadingStrategy } from '../spendings-loader';
import { CashboardSummariesInput } from '../cashboard-summaries/cashboard-summaries.component';
import { DateRange } from 'argo-ui-lib/src/components';
import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';

const PRICED_ITEMS_COUNT = 6;

@Component({
    selector: 'ax-cashboard',
    templateUrl: './cashboard.html',
    styles: [ require('../cashboard.scss') ],
})
export class CashboardComponent implements OnInit, OnDestroy, HasLayoutSettings, SpendingsLoadingStrategy<PerfData> {
    public priciestUsers: PricedItem[] = [];
    public priciestServices: PricedItem[] = [];
    public selectedIntervalFormatted: string = '';
    public currentInterval: DateIntervalCashboard;
    public priciesItemGroups: Dictionary<PriciesItemGroups[]> = {};
    public cashboardSummariesInput: CashboardSummariesInput;

    private spendingsLoader: SpendingsLoader<PerfData>;
    @ViewChild(SpendingsChartComponent)
    private spendingChart: SpendingsChartComponent;

    constructor(private perfDataService: PerfDataService,
                private usageService: UsageService,
                private router: Router,
                private activatedRoute: ActivatedRoute) {
    }

    get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Cashboard',
            hasExtendedBg: true,
            breadcrumb: [{
                title: `Spending for ${this.currentInterval.format()}`,
                routerLink: null,
            }]
        };
    }

    ngOnInit() {
        this.spendingsLoader = new SpendingsLoader<PerfData>(
            this.activatedRoute,
            this.updateRoute.bind(this),
            this.onDataLoaded.bind(this),
            this.onStepSelected.bind(this),
            this.onDateRangeSelected.bind(this),
            this);
        this.spendingsLoader.init();
    }

    reduceIntervalData(prev: PerfData, cur: PerfData): PerfData {
        return {
            min: prev.min + cur.min,
            max: prev.max + cur.max,
            data: prev.data + cur.data,
            time: Math.max(prev.time, cur.time)
        };
    }

    createIntervalData(item, ratio: number, time: moment.Moment): PerfData {
        return {max: item.max * ratio, min: item.min * ratio, data: item.data * ratio, time: time.unix()};
    }

    createZeroIntervals(times, loadedData): PerfData[] {
        return times.map(time => ({time: time, data: 0, min: 0, max: 0}));
    }

    loadSpendings(interval: number, startTime: number, endTime?: number): Observable<PerfData[]> {
        return this.perfDataService.getSpendings({interval, startTime, endTime});
    }

    ngOnDestroy() {
        if (this.spendingsLoader) {
            this.spendingsLoader.dispose();
        }
    }

    onDateRangeChange(range: DateRange) {
        this.router.navigate(['/app/cashboard', range.toRouteParams()]);
    }

    getRouteParams(additionalParams) {
        return _.extend(this.currentInterval.toRouteParams(), additionalParams);
    }

    private updateRoute(params: any) {
        this.router.navigate(['/app/cashboard', params]);
    }

    private onDataLoaded(perfData: PerfData[]) {
        this.spendingChart.data = {
            perfData: perfData.slice(0),
            interval: this.spendingsLoader.currentInterval,
        };
    }

    private onDateRangeSelected(startTime: number, endTime: number, dateInterval: DateIntervalCashboard) {
        this.currentInterval = dateInterval;
        this.spendingsLoader.enforceStepSelection = this.currentInterval.isCurrentDay;
    }

    private onStepSelected(startTime: number, endTime: number, index: number) {
        this.spendingChart.highlightStep(startTime, endTime, index);
        this.cashboardSummariesInput = {
            data: this.spendingChart.data.perfData,
            interval: this.currentInterval,
            step: {
                start: startTime,
                end: endTime,
                index: index
            }
        };
        if (index > -1) {
            this.usageService.getSpendingsDetailsForAsync(startTime, endTime || moment().unix())
                .subscribe(res => this.calculateSpendingsDetails(res.data));
        } else {
            this.usageService.getSpendingsDetailsForAsync(this.currentInterval.startTime, this.currentInterval.endTime)
                .subscribe(res => this.calculateSpendingsDetails(res.data));
        }
    }

    private calculateSpendingsDetails(selectedInterval: SpendingsDetail[]) {
        let total = 0;
        for (let item of selectedInterval) {
            total += item.spent;
        }

        this.priciesItemGroups = _.groupBy(Object.keys(COST_IDS).map(type => {
            let info = COST_IDS[type];
            return {
                title: `Spending by ${info.title}`,
                type: type,
                items: this.getPriciestItems(selectedInterval, total, type).slice(0, PRICED_ITEMS_COUNT),
                info: info,
            };
        }), v => v.info.rowId);
    }

    private getPriciestItems(details: SpendingsDetail[],
                             totalSpending: number,
                             property: string): PricedItem[] {
        let nameToId = new Map<string, string>();
        let nameToSpendings = new Map<string, number>();
        details.forEach(item => {
            let name = item.cost_id[property];
            if (name) {
                nameToSpendings.set(name, (nameToSpendings.get(name) || 0) + item.spent);
                nameToId.set(name, item.cost_id.id);
            }
        });
        return Array.from(nameToSpendings.entries()).map(keyValue => {
            let name = keyValue[0];
            let cost = keyValue[1];
            let percentage = MathOperations.roundToTwoDigits(totalSpending === 0 ? 0 : cost / totalSpending * 100);
            return new PricedItem(
                name, MathOperations.roundToTwoDigits(cost / 100), percentage, property !== 'service' ? nameToId.get(name) : null);
        }).sort((first, second) => second.percentage - first.percentage);
    }
}
