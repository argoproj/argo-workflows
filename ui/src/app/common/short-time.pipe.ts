import * as moment from 'moment';
import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'shortTime'
})
export class ShortTimePipe implements PipeTransform {
  transform(value: string, args: any[]) {
    return value ? moment(value).format('H:mm') : '';
  }
}
