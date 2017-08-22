import * as moment from 'moment';
import * as _ from 'lodash';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs/Subscription';
import { Observable } from 'rxjs/Observable';
import { DateIntervalCashboard } from './cashboard.view-models';

export interface HasTime {
    time: number;
}

/**
 * Implements methods to load, reduce and scale spending data.
 */
export interface SpendingsLoadingStrategy<T> {
    /**
     * Loads spending data starting from given time grouped by requested time interval.
     */
    loadSpendings(interval: number, startTime: number, endTime?: number): Observable<T[]>;
    /**
     * Combine two spendings together.
     */
    reduceIntervalData(first: T, second: T);
    /**
     * Creates new spending item scaled by given ratio.
     */
    createIntervalData(item: T, ratio: number, time: moment.Moment): T;

    createZeroIntervals(times: number[], loadedItems: T[]): T[];
}

export class SpendingsLoader<T extends HasTime> {
    public currentInterval: DateIntervalCashboard;
    public selectedStep: number;
    public enforceStepSelection = false;

    private perfData: T[] = [];
    private spendingsSubscription: Subscription;
    private liveSpendingsSubscription: Subscription;
    private routeChangeSubscription: Subscription;

    constructor(
        private activatedRoute: ActivatedRoute,
        private routeUpdater: (params: any) => void,
        private onDataLoaded: (perfData: T[]) => void,
        private onStepSelected: (startTime: number, endTime: number, index: number) => void,
        private onDateRangeSelected: (startTime: number, endTime, dateInterval: DateIntervalCashboard) => void,
        private spendingsLoadingStrategy: SpendingsLoadingStrategy<T>) {
    }

    public init() {
        this.routeChangeSubscription = this.activatedRoute.params.subscribe(params => {
            let interval = <DateIntervalCashboard>DateIntervalCashboard.fromRouteParams(params);
            if (this.currentInterval &&
                    this.currentInterval.durationDays === interval.durationDays &&
                    this.currentInterval.endDate.isSame(interval.endDate)) {
                if (this.selectedStep !== interval.selectedStep) {
                    this.applyStepSelection(interval.selectedStep);
                }
            } else {
                this.setCurrentInterval(interval, () => {
                    this.applyStepSelection(interval.selectedStep);
                });
            }
        });
    }

    public dispose() {
        this.stopLoadingLiveData();
        if (this.routeChangeSubscription !== null) {
            this.routeChangeSubscription.unsubscribe();
            this.routeChangeSubscription = null;
        }
    }

    public selectIntervalStep(step: number) {
        this.routeUpdater(_.extend(this.currentInterval.toRouteParams(), {step: step}));
    }

    public refresh() {
        this.setCurrentInterval(this.currentInterval, () => {
            this.applyStepSelection(this.selectedStep);
        });
    }

    private stopLoadingLiveData() {
        if (this.spendingsSubscription) {
            this.spendingsSubscription.unsubscribe();
            this.spendingsSubscription = null;
        }
        if (this.liveSpendingsSubscription) {
            this.liveSpendingsSubscription.unsubscribe();
            this.liveSpendingsSubscription = null;
        }
    }

    private loadSpendings(interval: number, startTime: number, endTime?: number): Observable<T[]> {
        return this.spendingsLoadingStrategy.loadSpendings(interval, startTime, endTime);
    }

    private updatePerfData(perf: T[], livePerf: T[], maxLivePerCount: number) {
        if (livePerf.length > 0) {
            let aggredatedLivePerf = livePerf.reduce((prev, current) => this.spendingsLoadingStrategy.reduceIntervalData(prev, current));
            let ratio = (maxLivePerCount + 1) / livePerf.length;
            perf.unshift(this.spendingsLoadingStrategy.createIntervalData(
                aggredatedLivePerf,
                ratio,
                (perf.length === 0 ? moment().startOf('hour') : moment.unix(perf[0].time)).add(1, 'hour')));
        }
        let zeroIntervals = [];
        let minTime: number = perf.length > 1 ? perf[perf.length - 1].time : moment().utc().unix();
        while (minTime - this.currentInterval.step.seconds > this.currentInterval.startDate.unix()) {
            minTime -= this.currentInterval.step.seconds;
            zeroIntervals.push(minTime);
        }
        this.spendingsLoadingStrategy.createZeroIntervals(zeroIntervals, perf).forEach(zero => perf.push(zero));
        this.onDataLoaded(perf.slice(0));
        this.perfData = perf.slice(0);
    }

    private setCurrentInterval(interval: DateIntervalCashboard, callback?: () => any) {
        this.currentInterval = interval;
        this.onDateRangeSelected(interval.startTime, interval.endTime, <DateIntervalCashboard>this.currentInterval.clone());
        let perfHistorical = this.loadSpendings(interval.step.seconds, interval.startTime, interval.endTime);

        let liveStepDurationSeconds = 60;
        let maxLivePerfCont = 3600 / 60;
        let liveIntervalStart = moment().startOf('hour');
        let liveIntervalEnd = moment().startOf('hour').add(1, 'hour');
        let perfLive = interval.isCurrentDay && moment().diff(liveIntervalStart) / 1000 > liveStepDurationSeconds ?
            this.loadSpendings(liveStepDurationSeconds, liveIntervalStart.utc().unix()) :
            Observable.of([]);

        this.stopLoadingLiveData();
        return this.spendingsSubscription = Observable.forkJoin(perfHistorical, perfLive).subscribe(results => {
            let perf = <T[]>results[0];
            this.updatePerfData(perf.slice(0), <T[]>results[1], maxLivePerfCont);
            if (callback) {
                callback();
            }

            this.stopLoadingLiveData();
            if (interval.isCurrentDay) {
                this.liveSpendingsSubscription = Observable
                    .interval(liveStepDurationSeconds * 1000)
                    .take(liveIntervalEnd.diff(moment()) / 1000 / liveStepDurationSeconds)
                    .subscribe(() => {
                        return this.loadSpendings(liveStepDurationSeconds, moment().startOf('hour').utc().unix())
                            .subscribe(live => this.updatePerfData(perf.slice(0), live, maxLivePerfCont));
                    });
            }
        });
    }

    private applyStepSelection(index: number) {
        if (index > -1 && index <= this.perfData.length - 1) {
            let startTime = this.perfData[index].time;
            let endTime = startTime + this.currentInterval.step.seconds;
            this.onStepSelected(startTime, endTime, index);
            this.selectedStep = index;
        } else if (this.enforceStepSelection && this.perfData.length > 0) {
            this.selectIntervalStep(0);
        } else {
            this.onStepSelected(null, null, -1);
            this.selectedStep = -1;
        }
    }
}
