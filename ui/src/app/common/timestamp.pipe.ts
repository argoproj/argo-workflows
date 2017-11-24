import { Pipe, PipeTransform } from '@angular/core';
import * as moment from 'moment';

@Pipe({
  name: 'timestamp'
})
export class TimestampPipe implements PipeTransform {
  transform(value: number, args: any[]) {
    console.log(value);
    if (value === 0) {
      return '';
    } else {
      return moment(value).format('YYYY/MM/DD') + ' ' + moment(value).format('HH:mm');
    }
  }
}
