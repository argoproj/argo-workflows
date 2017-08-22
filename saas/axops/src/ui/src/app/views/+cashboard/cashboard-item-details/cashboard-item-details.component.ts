import * as moment from 'moment';
import { Observable, Subscription } from 'rxjs';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';

import { MathOperations } from '../../../common/mathOperations/mathOperations';
import { DateRange } from 'argo-ui-lib/src/components';
import { SpendingsLoader, SpendingsLoadingStrategy } from '../spendings-loader';
import { CashboardSummariesInput } from '../cashboard-summaries/cashboard-summaries.component';
import { PerfData, PerfBreakDownData } from '../../../model';
import { SpendingsChartInput, PricedItem } from '../../../common/chartSpendings';
import { COST_IDS, DateIntervalCashboard } from '../cashboard.view-models';
import { PerfDataService } from '../../../services';
import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';


@Component({
    templateUrl: './cashboard-item-details.html',
})
/**
 * Represents page which contains chart with all services spendings and one user/service utilization. Page allows to select step within
 * date interval and displays spending breakdown in table below the chart.
 */
export class CashboardItemDetailsComponent implements OnInit, HasLayoutSettings, OnDestroy, SpendingsLoadingStrategy<PerfData> {

    public spendingsChartInput: SpendingsChartInput;
    public hightlightedStep: {startTime: number, endTime: number, index: number};
    public currentInterval: DateIntervalCashboard;
    public oppositeTypeSpendings: PricedItem[] = [];
    public cashboardSummariesInput: CashboardSummariesInput;

    private groupedOppositeBreakdown: PricedItem[][] = [];
    private name: string;
    private id: string;
    private type: string;
    private spendingsLoader: SpendingsLoader<PerfData>;
    private subscriptions: Subscription[] = [];

    constructor(
        private perfDataService: PerfDataService,
        private router: Router,
        private activatedRoute: ActivatedRoute) {
    }

    get pageTitle(): string {
        return this.titleType + ' ' + this.name;
    }

    get hasExtendedBg(): boolean {
        return true;
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return [{
            title: 'Cashboard',
            routerLink: ['/app/cashboard', this.currentInterval.toRouteParams()],
        }, {
            title: this.titleType,
            routerLink: [`/app/cashboard/details/${this.type}`, this.currentInterval.toRouteParams()]
        }];
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    get oppositeType(): string {
        return COST_IDS[this.type].oppositeType;
    }

    get titleType(): string {
        let titleToType = {
            service: 'Service',
            user: 'User',
            app: 'Application',
        };
        return titleToType[this.type];
    }

    ngOnInit() {
        this.type = this.activatedRoute.snapshot.params['type'];
        this.name = this.activatedRoute.snapshot.params['name'];
        this.id = this.activatedRoute.snapshot.params['id'];

        this.spendingsLoader = new SpendingsLoader<PerfData>(
            this.activatedRoute,
            this.updateRoute.bind(this),
            this.onDataLoaded.bind(this),
            this.onStepSelected.bind(this),
            this.onDateRangeSelected.bind(this),
            this);
        this.spendingsLoader.enforceStepSelection = true;
        this.spendingsLoader.init();
        this.subscriptions.push(this.activatedRoute.params.subscribe(params => {
            let type = params['type'];
            let name = params['name'];
            let id = params['id'];
            if (this.type !== type || this.name !== name || this.id !== id) {
                this.name = name;
                this.type = type;
                this.id = id;
                this.spendingsLoader.refresh();
            }
        }));
    }

    public loadSpendings(interval: number, startTime: number, endTime?: number): Observable<PerfData[]> {
        return Observable.fromPromise(
            Promise.all<any>([
                this.perfDataService.getSpendingsBreakDownBy({name: this.type, interval, startTime, endTime}).toPromise(),
                this.perfDataService.getSpendings({interval, startTime, endTime}).toPromise(),
                this.perfDataService.getSpendingsBreakDownBy({
                    name: this.oppositeType, interval, startTime, endTime,
                    filterBy: { by: this.type, value: this.id || this.name }}).toPromise()
            ]).then(result => {
                // Contains overall spendings for whole system. Rendered as line on top chart on the page
                let breakDownData: PerfBreakDownData[] = result[0];
                // Contains selected user spendings. Rendered as stack line on top chart on the page
                let perfData: PerfData[] = result[1];
                // Contains spendings for opposite type (e.g. for services spendings for user of vice versa)
                let oppositeBreakDownData: PerfBreakDownData[] = result[2];
                this.groupedOppositeBreakdown = this.groupBreakdownByTime(oppositeBreakDownData);
                let timeToUtilization = new Map<number, number>();
                breakDownData.filter(item => {
                    if (this.id) {
                        return item.id === this.id;
                    }
                    return item.name === this.name;
                }).forEach(item => timeToUtilization.set(item.time, item.data));
                perfData.forEach(item => {
                    let roundedStepTime = moment.unix(item.time).startOf(this.spendingsLoader.currentInterval.step.unitOfTime).unix();
                    item.max = timeToUtilization.get(roundedStepTime) || 0;
                });
                return perfData;
            }));
    }

    reduceIntervalData(prev, cur): PerfData {
        return {min: prev.min + cur.min, max: prev.max + cur.max, data: prev.data + cur.data, time: Math.max(prev.time, cur.time) };
    }

    createIntervalData(item, ratio: number, time: moment.Moment): PerfData {
        return {max: item.max * ratio, min: item.min * ratio, data: item.data * ratio, time: time.unix() };
    }

    createZeroIntervals(times, loadedData): PerfData[] {
        return times.map(time => ({time: time, data: 0, min: 0, max: 0}));
    }

    ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        if (this.spendingsLoader) {
            this.spendingsLoader.dispose();
        }
    }

    onDateRangeChange(range: DateRange) {
        this.updateRoute(range.toRouteParams());
    }

    private updateRoute(params: any) {
        if (this.id) {
            params.id = this.id;
        }
        this.router.navigate([`/app/cashboard/details/${this.type}/${this.name}`, params]);
    }

    private onDataLoaded(perfData: PerfData[]) {
        this.spendingsChartInput = {
            perfData: perfData,
            interval: this.currentInterval
        };
    }

    private onDateRangeSelected(startTime: number, endTime: number, dateInterval: DateIntervalCashboard) {
        this.currentInterval = dateInterval;
    }

    private onStepSelected(startTime: number, endTime: number, index: number) {
        this.currentInterval.selectedStep = index;
        this.hightlightedStep = {startTime: startTime, endTime: endTime, index: index};
        this.oppositeTypeSpendings = this.groupedOppositeBreakdown[index];
        this.cashboardSummariesInput = {
            data: this.spendingsChartInput.perfData,
            interval: this.currentInterval,
            step: {
                start: startTime,
                end: endTime,
                index: index
            }
        };
    }

    private groupBreakdownByTime(breakDownData: PerfBreakDownData[]): PricedItem[][] {
        let timeToBreakDownItems = new Map<number, PerfBreakDownData[]>();
        breakDownData.forEach(item => {
            let list = timeToBreakDownItems.get(item.time) || [];
            list.push(item);
            timeToBreakDownItems.set(item.time, list);
        });
        return Array.from(timeToBreakDownItems.entries()).map(entry => {
            let time = entry[0];
            let items = entry[1];
            let totalSpending = 0;
            items.forEach(item => totalSpending += item.data);
            return {
                time: time,
                items: items.map(item => {
                    let percentage = MathOperations.roundToTwoDigits(totalSpending === 0 ? 0 : item.data / totalSpending * 100);
                    return new PricedItem(item.name, MathOperations.roundToTwoDigits(item.data), percentage, item.name);
                }).sort((first, second) => second.percentage - first.percentage)
            };
        }).sort((first, second) => second.time - first.time).map(timeItems => timeItems.items);
    }
}
