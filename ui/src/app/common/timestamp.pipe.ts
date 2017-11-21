import { Pipe, PipeTransform } from '@angular/core';
import * as moment from 'moment';

@Pipe({
  name: 'timestamp'
})
export class TimestampPipe implements PipeTransform {
  transform(value: number, args: any[]) {
    if (value === 0) {
      return '';
    } else {
      const timestamp = value * 1000;
      return moment(timestamp).format('YYYY/MM/DD') + ' ' + moment(timestamp).format('HH:mm');
    }
  }
}
