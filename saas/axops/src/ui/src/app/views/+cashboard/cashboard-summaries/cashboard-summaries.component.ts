import * as moment from 'moment';
import { Component, Input } from '@angular/core';
import { PerfData } from '../../../model';
import { DateIntervalCashboard } from '../cashboard.view-models';
import { MathOperations } from '../../../common/mathOperations/mathOperations';

export interface CalculatedSummary {
    rangeFormatted: string;
    cost: number;
    desc: string;
    info: string;
}

export interface CashboardSummariesInput {
    data: PerfData[];
    interval: DateIntervalCashboard;
    step: { index: number, start: number, end: number };
}

@Component({
    selector: 'ax-cashboard-summaries',
    templateUrl: './cashboard-summaries.html',
})
export class CashboardSummariesComponent {

    summaries: CalculatedSummary[] = [];

    @Input()
    set input(value: CashboardSummariesInput) {
        this.summaries = value ? this.getSummaries(value.step.start, value.step.end, value.step.index, value.data, value.interval) : [];
    }

    getSummaries(stepStart: number, stepEnd: number, stepIndex: number, data: PerfData[], interval: DateIntervalCashboard) {
        let summaries = [];
        let totalSpend = 0;
        let totalUtil = 0;
        data.forEach(item => {
            totalSpend += item.data;
            totalUtil += item.max;
        });
        if (data.length > 0) {
            if (interval.isCurrentDay) {
                summaries = this.getCurrentDaySummaries(stepIndex, totalSpend, stepStart, stepEnd, data, interval);
            } else {
                summaries = this.getHistoricalSummaries(stepIndex, totalUtil, totalSpend, stepStart, stepEnd, data,
                    interval);
            }
        }
        return summaries;
    }

    /**
     * Returns summaries for current day view: current hour info and projections for future periods.
     */
    private getCurrentDaySummaries(
        index: number,
        totalSpend: number,
        stepStart: number,
        stepEnd: number,
        data: PerfData[],
        interval: DateIntervalCashboard): CalculatedSummary[] {
        let currentHourInfo = {
            rangeFormatted: 'Current Hour',
            cost: data[0].data,
            desc: 'Estimated Spending',
            info: this.formatUtilization(data[0].max, data[0].data),
        };
        let selectedStepInfo = {
            rangeFormatted: interval.step.rangeFormatter(moment.unix(stepStart), moment.unix(stepEnd)),
            cost: data[index].data,
            desc: `Spending Per ${interval.step.fullName}`,
            info: this.formatUtilization(data[index].max, data[index].data),
        };
        let todayEstimation = {
            rangeFormatted: 'Today',
            cost: totalSpend,
            desc: interval.step.rangeFormatter(moment().startOf('day'), moment().startOf('hour')),
            info: `Estimated: $${MathOperations.roundToTwoDigits(24 / moment().hour() * totalSpend / 100)}`
        };
        if (index === 0) {
            return [currentHourInfo, todayEstimation];
        } else {
            return [selectedStepInfo, currentHourInfo, todayEstimation];
        }
    }

    /**
     * Returns summaries for historical view: total/selected step spendings and averages spendings.
     */
    private getHistoricalSummaries(
        index: number,
        totalUtil: number,
        totalSpend: number,
        stepStart: number,
        stepEnd: number,
        data: PerfData[],
        interval: DateIntervalCashboard): CalculatedSummary[] {
        let start = interval.startTime;
        let end = interval.endTime;
        let avgs = interval.averages.map(avg => {
            let roundedEnd = moment.unix(end).startOf('hour').add(1, 'hour');
            let stepsCount = roundedEnd.diff(moment.unix(start), avg.timeUnit);
            let avgUtilization = totalUtil / stepsCount;
            let avgSpending = totalSpend / stepsCount;
            return {
                rangeFormatted: avg.adverb,
                cost: avgSpending,
                desc: `${avg.adverb} Average`,
                info: 'Avg. ' + this.formatUtilization(avgUtilization, avgSpending),
            };
        });
        let items = [];
        if (index === -1) {
            items = [{
                rangeFormatted: 'Total Spending',
                cost: totalSpend,
                desc: interval.step.rangeFormatter(moment.unix(start), moment.unix(end)),
                info: this.formatUtilization(totalUtil, totalSpend),
            }];
        } else {
            items = [{
                rangeFormatted: 'Total Spending',
                cost: data[index].data,
                desc: interval.step.rangeFormatter(moment.unix(stepStart), moment.unix(stepEnd)),
                info: this.formatUtilization(data[index].max, data[index].data),
            }];
        }
        return items.concat(avgs);
    }

    private formatUtilization(utilization: number, spending: number) {
        let percentage = spending === 0 ? 0 : utilization / spending * 100;
        return `Utilization: ${(percentage).toFixed()}% ($${MathOperations.roundToTwoDigits(utilization / 100)})`;
    }
}
