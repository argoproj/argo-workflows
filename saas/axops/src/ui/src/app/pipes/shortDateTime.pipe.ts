import * as moment from 'moment';
import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'axShortDateTime'
})

export class ShortDateTimePipe implements PipeTransform {
    transform(value: number, args: any[]) {
        return value ? moment.unix(value).format('YYYY/MM/DD, HH:mm') : '';
    }
}
