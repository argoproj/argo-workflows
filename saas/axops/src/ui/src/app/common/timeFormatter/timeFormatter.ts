import * as moment from 'moment';

export class TimeFormatter {
    static toUtc(time) {
        return moment(time).utc();
    }

    static dayNameShortcut(time): string {
        return (moment(time).format('dddd')).substr(0, 2);
    }

    static onlyDate(time) {
        return moment(time).format('YYYY/MM/DD');
    }

    static dateAndTime(time) {
        return moment(time).format('YYYY/MM/DD HH:mm');
    }

    static monthAndDay(time) {
        return moment(time).format('MMM DD');
    }

    static time(time) {
        return moment(time).format('HH:mm');
    }

    static timeShort(time) {
        return moment(time).format('HH:mm');
    }

    static twelveHoursTime(time) {
        return moment(time).format('hh:mm A');
    }
}
