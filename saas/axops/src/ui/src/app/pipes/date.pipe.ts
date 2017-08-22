import * as moment from 'moment';
import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'date'
})

export class DatePipe implements PipeTransform {
    transform(value: number, args: any[]) {
        return moment(new Date(value)).format('YYYY/MM/DD');
    }
}
