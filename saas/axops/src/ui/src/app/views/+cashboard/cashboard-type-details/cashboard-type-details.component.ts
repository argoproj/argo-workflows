import * as _ from 'lodash';
import * as moment from 'moment';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Observable, Subscription } from 'rxjs';

import { SpendingsLoader, SpendingsLoadingStrategy } from '../spendings-loader';
import { PerfDataService } from '../../../services';
import { PerfBreakDownData,  } from '../../../model';
import { SpendingsChartInput } from '../../../common/chartSpendings';
import { DateIntervalCashboard } from '../cashboard.view-models';
import { MathOperations } from '../../../common/mathOperations/mathOperations';
import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';
import { DateRange } from 'argo-ui-lib/src/components';

/**
 * Contains step time without date interval and corresponding utilizations for different users/services.
 */
interface BreakDownPerfs {
    time: number;
    breakDownItems: PerfBreakDownData[];
}

@Component({
    templateUrl: './cashboard-type-details.html',
})
/**
 * Represents page which contains utilization trending charts of users or services. Allow selecting one step within specified date range.
 * If step is selected then percentage usage breakdown is shown instead of charts.
 */
export class CashboardTypeDetailsComponent implements
    OnInit, OnDestroy, LayoutSettings, HasLayoutSettings, SpendingsLoadingStrategy<BreakDownPerfs> {
    type: string;
    spendingCharts: {
        name: string,
        id: string,
        input: SpendingsChartInput,
        totalUtilization: number,
        stepUtilization?: number,
        stepPercentage?: number
    }[] = [];
    spendingChartData: SpendingsChartInput;
    steps: any[];
    stepWidth: number = 0;
    expandedChartName = null;
    hightlightedStep: { startTime: number, endTime: number, index: number };
    hightlightedStepText: string = '';

    private spendingsLoader: SpendingsLoader<BreakDownPerfs>;
    private subscriptions: Subscription[] = [];
    private currentInterval: DateIntervalCashboard;

    constructor(
        private router: Router,
        private activatedRoute: ActivatedRoute,
        private perfService: PerfDataService) {
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    get pageTitle(): string {
        return this.titleType;
    }

    get titleType(): string {
        let titleToType = {
            service: 'Services',
            user: 'Users',
            app: 'Application',
        };
        return titleToType[this.type];
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return [{
            title: 'Cashboard',
            routerLink: ['/app/cashboard', this.currentInterval.toRouteParams()],
        }];
    }

    get hasExtendedBg(): boolean {
        return true;
    }

    trackByName(item) {
        return item.name;
    }

    ngOnInit() {
        this.type = this.activatedRoute.snapshot.params['type'] || 'service';
        this.spendingsLoader = new SpendingsLoader<BreakDownPerfs>(
            this.activatedRoute,
            this.updateRoute.bind(this),
            this.onDataLoaded.bind(this),
            this.onStepSelected.bind(this),
            this.onDateRangeSelected.bind(this),
            this);
        this.spendingsLoader.init();
        this.subscriptions.push(this.activatedRoute.params.subscribe(params => {
            let type = params['type'] || 'service';
            if (this.type !== type) {
                this.spendingsLoader.refresh();
            }
        }));
    }

    loadSpendings(interval: number, startTime: number, endTime?: number): Observable<BreakDownPerfs[]> {
        let spendings = this.perfService.getSpendingsBreakDownBy({name: this.type, interval, startTime, endTime});
        return spendings.map(items => {
            let timeToItems = {};
            items.forEach(breakDownItem => {
                let timeItems = timeToItems[breakDownItem.time];
                if (!timeItems) {
                    timeItems = [];
                    timeToItems[breakDownItem.time] = timeItems;
                }
                timeItems.push(breakDownItem);
            });
            return Object.keys(timeToItems).map(key => {
                return {
                    time: parseInt(key, 10),
                    breakDownItems: timeToItems[key]
                };
            }).sort((first, second) => second.time - first.time);
        });
    }

    reduceIntervalData(prev: BreakDownPerfs, current: BreakDownPerfs): BreakDownPerfs {
        let time = prev.time;
        let firstDict = _.groupBy(prev.breakDownItems, 'name');
        let secondDict = _.groupBy(current.breakDownItems, 'name');
        return {
            time: time,
            breakDownItems: _.intersection(Object.keys(firstDict), Object.keys(secondDict)).map(name => {
                let id = firstDict[name][0].id;
                let first: PerfBreakDownData = firstDict[name] ?
                    firstDict[name][0] : { time: time, data: 0, name: name, is_system: false, id };
                let second: PerfBreakDownData = secondDict[name] ?
                    secondDict[name][0] : { time: time, data: 0, name: name, is_system: false, id };
                return {
                    id,
                    time: time,
                    data: first.data + second.data,
                    name: name,
                    is_system: false
                };
            }),
        };
    }

    createIntervalData(item: BreakDownPerfs, ratio: number, time: moment.Moment): BreakDownPerfs {
        return {
            breakDownItems: item.breakDownItems.map(breakDownItem => ({
                time: time.unix(),
                name: breakDownItem.name,
                data: breakDownItem.data * ratio,
                id: breakDownItem.id,
                is_system: false
            })),
            time: time.unix()
        };
    }

    createZeroIntervals(times, loadedData): BreakDownPerfs[] {
        let itemNames = new Set<string>();
        loadedData.forEach(item => item.breakDownItems.forEach(breakDownItem => {
            itemNames.add(breakDownItem.name);
        }));
        loadedData.forEach(item => {
            _.difference(Array.from(itemNames), item.breakDownItems.map(_ => _.name)).forEach(missingName => {
                item.breakDownItems.push({
                    time: item.time,
                    name: missingName,
                    data: 0
                });
            });
        });
        return times.map(time => <BreakDownPerfs>{
            time: time,
            breakDownItems: Array.from(itemNames).map(name => <PerfBreakDownData>{
                time: time,
                data: 0,
                name: name
            })
        });
    }

    ngOnDestroy() {
        this.spendingsLoader.dispose();
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    getRouteParams(additionalParams) {
        return _.extend({}, this.currentInterval.toRouteParams(), additionalParams);
    }

    onDateRangeChange(range: DateRange) {
        this.updateRoute(range.toRouteParams());
    }

    get isStepSelected(): boolean {
        return this.hightlightedStep && this.hightlightedStep.index > -1;
    }

    private onDateRangeSelected(startTime: number, endTime: number, dateInterval: DateIntervalCashboard) {
        this.currentInterval = dateInterval;
    }

    private updateRoute(params: any) {
        this.router.navigate([`/app/cashboard/details/${this.type}`, params]);
    }

    private onDataLoaded(perfData: BreakDownPerfs[]) {
        let nameToId = {};
        let nameToData = {};
        perfData.forEach(perf => {
            let time = perf.time;
            let items = perf.breakDownItems || [];
            items.forEach(item => {
                nameToId[item.name] = item.id;
                let data = nameToData[item.name];
                if (!data) {
                    data = [];
                    nameToData[item.name] = data;
                }
                data.push({
                    time: time,
                    data: 0,
                    min: 0,
                    max: item.data,
                });
            });
        });
        this.spendingCharts = Object.keys(nameToData).map(key => {
            return {
                name: key,
                id: nameToId[key],
                input: {
                    perfData: nameToData[key],
                    interval: this.spendingsLoader.currentInterval,
                },
                totalUtilization: MathOperations.roundToTwoDigits(_.sumBy(nameToData[key], (v) => v['max']) / 100)
            };
        }).sort((first, second) => second.totalUtilization - first.totalUtilization);
        this.steps = perfData;
        this.stepWidth = 100 / perfData.length;
    }

    private onStepSelected(startTime: number, endTime: number, index: number) {
        this.currentInterval.selectedStep = index;
        this.hightlightedStep = { startTime: startTime, endTime: endTime, index: index };
        if (index > -1) {
            let startTimeFormatted = this.currentInterval.step.dateFormatter(startTime * 1000);
            let endTimeFormatted = this.currentInterval.step.dateFormatter(endTime * 1000);
            this.hightlightedStepText = `${startTimeFormatted} - ${endTimeFormatted}`;
            let totalUtilization = 0;
            this.spendingCharts.forEach(chart => {
                totalUtilization += chart.input.perfData[chart.input.perfData.length - index - 1].max;
            });
            this.spendingCharts.forEach(chart => {
                let utilization = chart.input.perfData[chart.input.perfData.length - index - 1].max;
                chart.stepPercentage = MathOperations.roundToTwoDigits(utilization / totalUtilization * 100);
                chart.stepUtilization = MathOperations.roundToTwoDigits(utilization / 100);
            });
        }
    }
}
