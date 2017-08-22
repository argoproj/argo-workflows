import * as moment from 'moment';
import { DateInterval, PricedItem } from '../../common/chartSpendings';
import { DateRange } from 'argo-ui-lib/src/components';

export interface Dictionary<T> {
    [index: string]: T;
}

export class PriciesItemGroups {
    title: string;
    type: string;
    items: PricedItem[];
}

/**
 * Contains information about supported cost ids.
 */
export const COST_IDS = {
    service: {
        title: 'Templates',
        oppositeType: 'user',
        rowId: 1,
    },
    user: {
        title: 'Users',
        oppositeType: 'service',
        rowId: 1,
    },
    app: {
        title: 'Applications',
        oppositeType: 'app',
        rowId: 2,
    }
};

const AVERAGES = {
    HOUR: {adverb: 'Hourly', timeUnit: <moment.unitOfTime.Diff> 'hour'},
    DAY: {adverb: 'Daily', timeUnit: <moment.unitOfTime.Diff> 'day'},
    WEEK: {adverb: 'Weekly', timeUnit: <moment.unitOfTime.Diff> 'week'},
    MONTH: {adverb: 'Monthly', timeUnit: <moment.unitOfTime.Diff> 'month'}
};

export interface AvgInfo {
    adverb: string;
    timeUnit: moment.unitOfTime.Diff;
}

/**
 * Represents cashboard date interval and optional step selection within this interval.
 */
export class DateIntervalCashboard extends DateInterval {

    /**
     * Deserializes interval from route parameters.
     */
    static fromRouteParams(params, defaultDays?: number): DateIntervalCashboard {
        let range = DateRange.fromRouteParams(params);
        return new DateIntervalCashboard(range.durationDays, range.endDate, params['step'] ? parseInt(params['step'], 10) : -1);
    }

    constructor(durationDays: number, endDate: moment.Moment, public selectedStep: number = -1) {
        super(durationDays, endDate, selectedStep);
    }

    /**
     * Returns two steps for current interval which might be used to calculate averages;
     */
    get averages(): AvgInfo[] {
        if (this.durationDays < 7) {
            return [AVERAGES.HOUR, AVERAGES.DAY];
        } else if (this.durationDays < 30) {
            return [AVERAGES.DAY, AVERAGES.WEEK];
        } else {
            return [AVERAGES.WEEK, AVERAGES.MONTH];
        }
    };

    clone(): DateIntervalCashboard {
        return DateIntervalCashboard.fromRouteParams(this.toRouteParams());
    }
}
