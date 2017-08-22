import * as _ from 'lodash';
import * as moment from 'moment';
import { PerfData } from '../../model';
import { TimeFormatter } from '../timeFormatter/timeFormatter';
import { DateRange } from 'argo-ui-lib/src/components';

export const HOUR: TimeInterval = {
    unitOfTime: 'hour',
    seconds: 3600,
    shortName: 'h',
    fullName: 'Hour',
    dateFormatter: input => TimeFormatter.time(input),
    rangeFormatter: (start: moment.Moment, end: moment.Moment) => {
        return `${start.format('MMMM')} ${start.format('DD')}, ${start.format('H:mm')} - ${end.format('MMMM')} ${end.format('DD')}, ${end.format('H:mm')}`;
    }
};

export const DAY: TimeInterval = {
    unitOfTime: 'day',
    seconds: 3600 * 24,
    shortName: 'd',
    fullName: 'Day',
    dateFormatter: input => TimeFormatter.monthAndDay(TimeFormatter.toUtc(input)),
    rangeFormatter: (start: moment.Moment, end: moment.Moment) => {
        if (start.month() === end.month()) {
            return `${start.format('MMMM')} ${start.format('DD')} - ${end.format('DD')}`;
        } else {
            return `${start.format('MMMM')} ${start.format('DD')} - ${end.format('MMMM')} ${end.format('DD')}`;
        }
    }
};

export const WEEK: TimeInterval = {
    unitOfTime: 'week',
    seconds: 3600 * 24 * 7,
    shortName: 'w',
    fullName: 'Week',
    dateFormatter: DAY.dateFormatter,
    rangeFormatter: DAY.rangeFormatter
};

export const Utils = {
    getIntervalFromTime(x: number, perfData: PerfData[], intervalSeconds: number, includeLastStep = false):
        {index: number, left: number, right: number} {
        let result = null;
        if (perfData.length > 1) {
            let min = perfData[perfData.length - 1].time;
            let max = perfData[0].time + intervalSeconds;
            let index = Math.floor((x - min) / intervalSeconds);
            if (index > -1) {
                let left = (min + index * intervalSeconds);
                let right = Math.min(min + (index + 1) * intervalSeconds, max);
                if (includeLastStep && right >= left || right > left) {
                    result = {
                        index: perfData.length - index - 1,
                        left: left,
                        right: right
                    };
                }
            }
        }
        return result;
    }
};

export class PricedItem {
    constructor(public name: string, public cost: number, public percentage: number, public id: string) {
    }
}

export interface TimeInterval {
    seconds: number;
    unitOfTime: moment.unitOfTime.StartOf;
    shortName: string;
    fullName: string;
    dateFormatter: (input: number) => string;
    rangeFormatter: (start: moment.Moment, end: moment.Moment) => string;
}

/**
 * Represents date interval and optional step selection within this interval.
 */
export class DateInterval extends DateRange {

    constructor(durationDays: number, endDate: moment.Moment, public selectedStep: number = -1) {
        super(durationDays, endDate);
    }

    /**
     * Calculates step based on interval length.
     */
    get step(): TimeInterval {
        if (this.durationDays <= 3) {
            return HOUR;
        } else if (this.durationDays <= 30) {
            return DAY;
        } else {
            return WEEK;
        }
    }

    /**
     * Returns interval unix start time.
     */
    get startTime(): number {
        return this.startDate.unix();
    }

    /**
     * Returns interval unix end time.
     */
    get endTime(): number {
        return this.endDate.unix();
    }

    /**
     * Returns true if interval represents current day.
     */
    get isCurrentDay(): boolean {
        return this.durationDays === 1 && this.startDate.isSame(moment().startOf('day'));
    }

    /**
     * Serializes interval to route parameters.
     */
    toRouteParams() {
        return _.extend(super.toRouteParams(), {step: this.selectedStep});
    }
}

export interface SpendingsChartInput {
    perfData: PerfData[];
    interval: DateInterval;
}

export class ChartConfig {
    detailsColor: string = '';
    margin: Margin;
}

export class Margin {
    top: number = 0;
    bottom: number = 0;
    left: number = 0;
    right: number = 0;
}
