import {Pipe, PipeTransform} from '@angular/core';
import {TimeFormatter} from '../common/timeFormatter/timeFormatter';

declare var jstz: any;

@Pipe({
    name: 'timestamp'
})

export class TimestampPipe implements PipeTransform {
    transform(value: number, args: any[]) {
        if (value === 0) {
            return '';
        } else {
            let timestamp = value * 1000;
            return TimeFormatter.onlyDate(timestamp) + ' ' + TimeFormatter.timeShort(timestamp);
        }
    }
}
