import * as moment from 'moment';

export class TimeOperations {
    static getCurrentTimeUtc(): number {
        return moment.utc().unix(); // result in seconds
    }

    static daysInSeconds(noOfDays: number): number {
        return noOfDays * 86400;
    }

    static getBeginningDayTime(time: number) {
        return (time - (time % 86400)); // beginning day in utc (in sec)
    }

    static getEndDayTime(time: number) {
        return TimeOperations.getBeginningDayTime(time) + 86400;
    }

    static unitInMilliseconds(value: number, unit: string) {
        switch (moment.normalizeUnits(<any>unit)) {
            case 'month':
                return value * 2592000000;
            case 'week':
                return value * 604800000;
            case 'day':
                return value * 86400000;
            case 'hour':
                return value * 3600000;
            case 'minute':
                return value * 60000;
            case 'second':
                return value * 1000;
            default:
                return value;
        }
    }

    static millisecondsAsDays(value: number): number {
        let duration: number = <number>moment.duration(value).asDays();
        return Math.round(duration);
    }
}
