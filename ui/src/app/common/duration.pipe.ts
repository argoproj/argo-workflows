import { Pipe, PipeTransform } from '@angular/core';
import * as moment from 'moment';

@Pipe({
  name: 'duration'
})
export class DurationPipe implements PipeTransform {

  transform(value: number, allowNewLines): string {
    const momentTimeStart = moment.utc(0);
    const momentTime = moment.utc(value * 1000);
    const duration = moment.duration(momentTime.diff(momentTimeStart));
    let formattedTime = '';

    if (momentTime.diff(momentTimeStart, 'hours') === 0) {
      formattedTime = ('0' + duration.minutes()).slice(-2) + ':' + ('0' + duration.seconds()).slice(-2) + ' min';
    } else {
      if (momentTime.diff(momentTimeStart, 'days') > 0) {
        formattedTime += momentTime.diff(momentTimeStart, 'days') + ' days' + (allowNewLines ? '<br>' : ' ');
      }

      formattedTime += ('0' + duration.hours()).slice(-2) + ':' + ('0' + duration.minutes()).slice(-2) + ' hours';
    }
    return value ? formattedTime : '';
  }
}
